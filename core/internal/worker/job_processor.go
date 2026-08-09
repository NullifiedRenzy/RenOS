package worker

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/renos-project/renos/core/internal/db"
	"github.com/renos-project/renos/core/internal/execctrl"
	"github.com/renos-project/renos/core/internal/ws"
)

const (
	retryBaseDelay = 30 * time.Second
	retryMaxDelay  = 10 * time.Minute
)

func (w *worker) processCommand(job *db.Job, payload *jobPayload) {
	w.processCommandWithControl(job, payload, w.pool.ctx)
}

func (w *worker) processContainerAction(job *db.Job, payload *jobPayload) {
	w.processContainerActionWithControl(job, payload, w.pool.ctx)
}

func (w *worker) processServiceAction(job *db.Job, payload *jobPayload) {
	w.processServiceActionWithControl(job, payload, w.pool.ctx)
}

func (w *worker) processCommandWithControl(job *db.Job, payload *jobPayload, jobCtx context.Context) {
	groupActionID := job.GroupActionID
	allDevices := payload.DeviceIDs
	failed, stopped := w.dispatchToDevices(job, allDevices, jobCtx, func(ctx context.Context, deviceID string) error {
		_, err := w.dispatchCommand(deviceID, job.CreatedBy, payload.Command, groupActionID)
		return err
	})
	if stopped {
		return
	}
	if len(failed) > 0 && job.Retries < job.MaxRetries {
		w.retryDevicesWithControl(job, payload, failed, jobCtx)
	}
	if groupActionID != nil {
		go w.monitorGroupAction(job.ID, *groupActionID, len(payload.DeviceIDs))
	}
}

func (w *worker) processContainerActionWithControl(job *db.Job, payload *jobPayload, jobCtx context.Context) {
	groupActionID := job.GroupActionID
	allDevices := payload.DeviceIDs
	failed, stopped := w.dispatchToDevices(job, allDevices, jobCtx, func(ctx context.Context, deviceID string) error {
		act, err := w.pool.contAct.Create(ctx, deviceID, job.CreatedBy, payload.ContainerID, payload.Action)
		if err != nil {
			return err
		}
		w.pool.hub.Broadcast(ws.Message{
			Type:     "container_action.requested",
			DeviceID: deviceID,
			Data: map[string]interface{}{
				"action_id":    act.ID,
				"container_id": act.ContainerID,
				"action":       act.Action,
				"status":       act.Status,
			},
		})
		return nil
	})
	if stopped {
		return
	}
	if len(failed) > 0 && job.Retries < job.MaxRetries {
		w.retryDevicesWithControl(job, payload, failed, jobCtx)
	}
	if groupActionID != nil {
		go w.monitorGroupAction(job.ID, *groupActionID, len(payload.DeviceIDs))
	}
}

func (w *worker) processServiceActionWithControl(job *db.Job, payload *jobPayload, jobCtx context.Context) {
	groupActionID := job.GroupActionID
	allDevices := payload.DeviceIDs
	failed, stopped := w.dispatchToDevices(job, allDevices, jobCtx, func(ctx context.Context, deviceID string) error {
		act, err := w.pool.svcAct.Create(ctx, deviceID, job.CreatedBy, payload.UnitName, payload.Action)
		if err != nil {
			return err
		}
		w.pool.hub.Broadcast(ws.Message{
			Type:     "service_action.requested",
			DeviceID: deviceID,
			Data: map[string]interface{}{
				"action_id": act.ID,
				"unit_name": act.UnitName,
				"action":    act.Action,
				"status":    act.Status,
			},
		})
		return nil
	})
	if stopped {
		return
	}
	if len(failed) > 0 && job.Retries < job.MaxRetries {
		w.retryDevicesWithControl(job, payload, failed, jobCtx)
	}
	if groupActionID != nil {
		go w.monitorGroupAction(job.ID, *groupActionID, len(payload.DeviceIDs))
	}
}

type dispatchFn func(ctx context.Context, deviceID string) error

func (w *worker) dispatchToDevices(job *db.Job, deviceIDs []string, jobCtx context.Context, fn dispatchFn) (failed []string, stopped bool) {
	total := len(deviceIDs)
	batchSize := job.BatchSize
	if batchSize <= 0 {
		batchSize = defaultBatchSize
	}
	delayMs := job.BatchDelayMs
	if delayMs <= 0 {
		delayMs = defaultBatchDelayMs
	}
	batchDelay := time.Duration(delayMs) * time.Millisecond
	maxDeviceExec := time.Duration(job.MaxDeviceExecutionMs) * time.Millisecond

	var nonRetryableFailed []string
	var retryableFailed []string
	var mu sync.Mutex

	for start := 0; start < total; start += batchSize {
		end := start + batchSize
		if end > total {
			end = total
		}
		batch := deviceIDs[start:end]

		select {
		case <-jobCtx.Done():
			return nil, true
		default:
		}

		// Check cancellation once per batch instead of per device (N+1 fix)
		cancelled := w.isCancelled(job.ID)

		var wg sync.WaitGroup
		var batchRetryable []string
		var batchNonRetryable []string
		var bmu sync.Mutex

		for _, deviceID := range batch {
			wg.Add(1)
			did := deviceID

			// Check cancellation (cached per batch)
			if cancelled {
				wg.Done()
				continue
			}

			// Check circuit breaker
			if w.pool.circuitBreak.IsUnstable(did) {
				slog.Warn("skipping unstable device", "job_id", job.ID, "device_id", did)
				bmu.Lock()
				batchNonRetryable = append(batchNonRetryable, did)
				bmu.Unlock()
				w.pool.audits.Create(jobCtx, "system", "worker", "device.blocked", "device", did)
				w.pool.logs.Create(jobCtx, "warn", "worker", "Skipped unstable device", logMeta(job.ID, job.Type, did))
				w.pool.hub.Broadcast(ws.Message{
					Type: "device.blocked",
					Data: map[string]interface{}{
						"device_id": did,
						"job_id":    job.ID,
						"reason":    "unstable",
					},
				})
				wg.Done()
				continue
			}

			// Check rate limit
			if !w.pool.rateLimiter.Allow(did) {
				slog.Debug("rate limited device", "job_id", job.ID, "device_id", did)
				bmu.Lock()
				batchRetryable = append(batchRetryable, did)
				bmu.Unlock()
				w.pool.audits.Create(jobCtx, "system", "worker", "rate.limit.triggered", "device", did)
				wg.Done()
				continue
			}

			go func() {
				defer wg.Done()

				select {
				case w.pool.deviceSem <- struct{}{}:
				default:
					bmu.Lock()
					batchRetryable = append(batchRetryable, did)
					bmu.Unlock()
					return
				}
				defer func() { <-w.pool.deviceSem }()

				devCtx := jobCtx
				var devCancel context.CancelFunc
				if maxDeviceExec > 0 {
					devCtx, devCancel = context.WithTimeout(jobCtx, maxDeviceExec)
					defer devCancel()
				}

				err := fn(devCtx, did)

				if err != nil {
					if execctrl.IsRetryable(errCode(err)) {
						bmu.Lock()
						batchRetryable = append(batchRetryable, did)
						bmu.Unlock()
					} else {
						bmu.Lock()
						batchNonRetryable = append(batchNonRetryable, did)
						bmu.Unlock()
					}
					w.pool.circuitBreak.RecordFailure(jobCtx, did)
				} else {
					w.pool.circuitBreak.RecordSuccess(did)
				}

				if err != nil && devCtx.Err() == context.DeadlineExceeded {
					slog.Warn("device execution timed out", "job_id", job.ID, "device_id", did)
				}
			}()
		}
		wg.Wait()

		bmu.Lock()
		retryableFailed = append(retryableFailed, batchRetryable...)
		nonRetryableFailed = append(nonRetryableFailed, batchNonRetryable...)
		bmu.Unlock()

		// Failure containment check after each batch
		mu.Lock()
		allFailed := retryableFailed
		allFailed = append(allFailed, nonRetryableFailed...)
		mu.Unlock()

		if w.checkFailureContainment(job, total, len(allFailed), jobCtx) {
			return allFailed, true
		}

		// Delay between batches
		if end < total {
			select {
			case <-jobCtx.Done():
				return nil, true
			case <-time.After(batchDelay):
			}
		}
	}

	mu.Lock()
	allFailed := retryableFailed
	allFailed = append(allFailed, nonRetryableFailed...)
	mu.Unlock()

	if job.StopOnFailure && len(nonRetryableFailed) > 0 {
		w.stopJob(job, "non_retryable_failure", jobCtx)
		return allFailed, true
	}

	return allFailed, false
}

func (w *worker) checkFailureContainment(job *db.Job, total, failed int, jobCtx context.Context) bool {
	if total == 0 {
		return false
	}
	failureRate := float64(failed) / float64(total)
	if failureRate > failureContainmentThreshold {
		slog.Warn("failure containment triggered", "job_id", job.ID, "failure_rate", failureRate)
		w.stopJob(job, "failure_threshold_exceeded", jobCtx)
		return true
	}
	return false
}

func (w *worker) stopJob(job *db.Job, reason string, jobCtx context.Context) {
	slog.Warn("stopping job", "job_id", job.ID, "reason", reason)
	w.pool.jobRepo.MarkStopped(jobCtx, job.ID, reason)
	w.pool.audits.Create(jobCtx, "system", "worker", "job.stopped", "job", job.ID)
	w.pool.logs.Create(jobCtx, "warn", "worker", "Job stopped: "+reason, logMeta(job.ID, job.Type, ""))
	w.pool.hub.Broadcast(ws.Message{
		Type: "job.stopped",
		Data: map[string]interface{}{
			"job_id": job.ID,
			"reason": reason,
		},
	})
}

func (w *worker) processUpdate(job *db.Job, payload *jobPayload) {
	slog.Info("update job processed", "job_id", job.ID, "action", payload.Action)
	w.pool.audits.Create(w.pool.ctx, "system", "worker", "job.completed", "job", job.ID)
	w.pool.jobRepo.UpdateProgress(w.pool.ctx, job.ID, len(payload.DeviceIDs), 0)
}

func (w *worker) retryDevicesWithControl(job *db.Job, payload *jobPayload, failedDevices []string, jobCtx context.Context) {
	w.pool.jobRepo.IncrementRetries(w.pool.ctx, job.ID)

	delay := retryBaseDelay * (1 << job.Retries)
	if delay > retryMaxDelay {
		delay = retryMaxDelay
	}

	slog.Info("retrying failed devices",
		"job_id", job.ID,
		"retry", job.Retries+1,
		"max_retries", job.MaxRetries,
		"failed_devices", len(failedDevices),
		"backoff", delay,
	)

	select {
	case <-jobCtx.Done():
		return
	case <-time.After(delay):
	}

	if w.isCancelled(job.ID) {
		return
	}

	// Re-dispatch only to failed devices
	for _, deviceID := range failedDevices {
		if w.isCancelled(job.ID) {
			return
		}
		select {
		case w.pool.deviceSem <- struct{}{}:
		default:
			continue
		}

		var err error
		switch job.Type {
		case "command":
			_, err = w.dispatchCommand(deviceID, job.CreatedBy, payload.Command, job.GroupActionID)
		case "container_action":
			var act *db.ContainerAction
			act, err = w.pool.contAct.Create(jobCtx, deviceID, job.CreatedBy, payload.ContainerID, payload.Action)
			if err == nil {
				w.pool.hub.Broadcast(ws.Message{
					Type:     "container_action.requested",
					DeviceID: deviceID,
					Data: map[string]interface{}{
						"action_id":    act.ID,
						"container_id": act.ContainerID,
						"action":       act.Action,
						"status":       act.Status,
					},
				})
			}
		case "service_action":
			var act *db.ServiceAction
			act, err = w.pool.svcAct.Create(jobCtx, deviceID, job.CreatedBy, payload.UnitName, payload.Action)
			if err == nil {
				w.pool.hub.Broadcast(ws.Message{
					Type:     "service_action.requested",
					DeviceID: deviceID,
					Data: map[string]interface{}{
						"action_id": act.ID,
						"unit_name": act.UnitName,
						"action":    act.Action,
						"status":    act.Status,
					},
				})
			}
		}
		<-w.pool.deviceSem

		if err != nil {
			slog.Error("retry dispatch failed", "job_id", job.ID, "device_id", deviceID, "error", err)
		}
	}
	w.pool.audits.Create(jobCtx, "system", "worker", "job.retry", "job", job.ID)
}

func errCode(err error) string {
	if e, ok := err.(*execctrl.ExecError); ok {
		return e.Code
	}
	return "unknown"
}
