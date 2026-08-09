package worker

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/renos-project/renos/core/internal/db"
	"github.com/renos-project/renos/core/internal/execctrl"
	"github.com/renos-project/renos/core/internal/notify"
	"github.com/renos-project/renos/core/internal/ws"
)

const (
	defaultBatchSize            = 5
	defaultBatchDelayMs         = 2000
	failureContainmentThreshold = 0.5
)

type WorkerPool struct {
	workers      []*worker
	jobRepo      *db.JobRepository
	cmdRepo      *db.CommandRepository
	contAct      *db.ContainerActionRepository
	svcAct       *db.ServiceActionRepository
	grpRepo      *db.GroupRepository
	grpAct       *db.GroupActionRepository
	audits       *db.AuditRepository
	logs         *db.SystemLogRepository
	hub          *ws.Hub
	deviceRepo   *db.DeviceRepository
	notifier     *notify.Notifier
	rateLimiter  *execctrl.DeviceRateLimiter
	circuitBreak *execctrl.CircuitBreaker
	poolSize     int
	interval     time.Duration
	jobSem       chan struct{}
	deviceSem    chan struct{}
	maxJobDur    time.Duration
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
}

type worker struct {
	id   int
	pool *WorkerPool
}

type jobPayload struct {
	Command     string   `json:"command,omitempty"`
	DeviceIDs   []string `json:"device_ids,omitempty"`
	GroupID     *string  `json:"group_id,omitempty"`
	ContainerID string   `json:"container_id,omitempty"`
	Action      string   `json:"action,omitempty"`
	UnitName    string   `json:"unit_name,omitempty"`
}

func NewPool(jobRepo *db.JobRepository, cmdRepo *db.CommandRepository, contAct *db.ContainerActionRepository, svcAct *db.ServiceActionRepository, grpRepo *db.GroupRepository, grpAct *db.GroupActionRepository, audits *db.AuditRepository, logs *db.SystemLogRepository, hub *ws.Hub, deviceRepo *db.DeviceRepository, notifier *notify.Notifier, poolSize int, maxConcurrentJobs int, maxConcurrentDeviceActions int, deviceCooldown time.Duration, circuitBreakerThreshold int, circuitBreakerResetAfter time.Duration, maxJobDur time.Duration) *WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())
	if poolSize < 1 {
		poolSize = 5
	}
	if maxConcurrentJobs < 1 {
		maxConcurrentJobs = 10
	}
	if maxConcurrentDeviceActions < 1 {
		maxConcurrentDeviceActions = 20
	}
	return &WorkerPool{
		jobRepo:      jobRepo,
		cmdRepo:      cmdRepo,
		contAct:      contAct,
		svcAct:       svcAct,
		grpRepo:      grpRepo,
		grpAct:       grpAct,
		audits:       audits,
		logs:         logs,
		hub:          hub,
		deviceRepo:   deviceRepo,
		notifier:     notifier,
		rateLimiter:  execctrl.NewDeviceRateLimiter(deviceCooldown),
		circuitBreak: execctrl.NewCircuitBreaker(circuitBreakerThreshold, circuitBreakerResetAfter, deviceRepo),
		poolSize:     poolSize,
		interval:     2 * time.Second,
		jobSem:       make(chan struct{}, maxConcurrentJobs),
		deviceSem:    make(chan struct{}, maxConcurrentDeviceActions),
		maxJobDur:    maxJobDur,
		ctx:          ctx,
		cancel:       cancel,
	}
}

// notifyJobStatus sends an in-app notification for a job lifecycle event.
func (p *WorkerPool) notifyJobStatus(job *db.Job, title, body, status string) {
	if p.notifier == nil || job == nil {
		return
	}
	p.notifier.Notify(p.ctx, job.CreatedBy, "job", title, body, "job", job.ID,
		map[string]any{"type": job.Type, "status": status})
}

func (p *WorkerPool) Start() {
	for i := 0; i < p.poolSize; i++ {
		w := &worker{id: i + 1, pool: p}
		p.workers = append(p.workers, w)
		p.wg.Add(1)
		go w.run()
	}
	slog.Info("worker pool started", "size", p.poolSize, "poll_interval", p.interval)
}

func (p *WorkerPool) Stop() {
	slog.Info("worker pool shutting down")
	p.cancel()
	p.wg.Wait()
	slog.Info("worker pool stopped")
}

func (w *worker) run() {
	defer w.pool.wg.Done()
	slog.Debug("worker started", "id", w.id)

	ticker := time.NewTicker(w.pool.interval)
	defer ticker.Stop()

	for {
		select {
		case <-w.pool.ctx.Done():
			slog.Debug("worker stopping", "id", w.id)
			return
		case <-ticker.C:
			w.tryClaim()
		}
	}
}

func (w *worker) tryClaim() {
	select {
	case w.pool.jobSem <- struct{}{}:
	default:
		return
	}
	defer func() { <-w.pool.jobSem }()

	job, err := w.pool.jobRepo.Claim(w.pool.ctx)
	if err != nil {
		return
	}
	if job == nil {
		return
	}

	slog.Info("worker claimed job", "id", w.id, "job_id", job.ID, "type", job.Type)
	w.pool.audits.Create(w.pool.ctx, "system", "worker", "job.started", "job", job.ID)
	w.pool.logs.Create(w.pool.ctx, "info", "worker", "Job started", logMeta(job.ID, job.Type, ""))
	w.pool.hub.Broadcast(ws.Message{
		Type: "job.started",
		Data: map[string]interface{}{
			"job_id": job.ID,
			"type":   job.Type,
		},
	})

	w.process(job)
}

func logMeta(jobID, jobType, extra string) json.RawMessage {
	m := map[string]interface{}{}
	if jobID != "" {
		m["job_id"] = jobID
	}
	if jobType != "" {
		m["job_type"] = jobType
	}
	if extra != "" {
		m["detail"] = extra
	}
	b, _ := json.Marshal(m)
	return b
}

func (w *worker) process(job *db.Job) {
	var payload jobPayload
	if err := json.Unmarshal(job.Payload, &payload); err != nil {
		slog.Error("failed to parse job payload", "job_id", job.ID, "error", err)
		w.pool.jobRepo.MarkFailed(w.pool.ctx, job.ID)
		w.pool.audits.Create(w.pool.ctx, "system", "worker", "job.failed", "job", job.ID)
		w.pool.logs.Create(w.pool.ctx, "error", "worker", "Failed to parse job payload", logMeta(job.ID, job.Type, err.Error()))
		w.pool.hub.Broadcast(ws.Message{
			Type: "job.failed",
			Data: map[string]interface{}{
				"job_id":  job.ID,
				"type":    job.Type,
				"reason":  "failed to parse payload",
			},
		})
		w.pool.notifyJobStatus(job, "Job failed", "Job failed to start ("+job.Type+").", "failed")
		return
	}

	jobCtx := w.pool.ctx
	var cancel context.CancelFunc

	maxDur := time.Duration(job.MaxJobDurationMs) * time.Millisecond
	if maxDur <= 0 {
		maxDur = w.pool.maxJobDur
	}
	if maxDur > 0 {
		jobCtx, cancel = context.WithTimeout(w.pool.ctx, maxDur)
		defer cancel()
		go func() {
			<-jobCtx.Done()
			if jobCtx.Err() == context.DeadlineExceeded {
				slog.Warn("job timed out", "job_id", job.ID)
				w.pool.jobRepo.MarkStopped(jobCtx, job.ID, "job_timeout")
				w.pool.audits.Create(jobCtx, "system", "worker", "job.stopped", "job", job.ID)
				w.pool.logs.Create(jobCtx, "warn", "worker", "Job timed out", logMeta(job.ID, job.Type, "exceeded max duration"))
				w.pool.hub.Broadcast(ws.Message{
					Type: "job.stopped",
					Data: map[string]interface{}{
						"job_id": job.ID,
						"reason": "timeout",
					},
				})
				w.pool.notifyJobStatus(job, "Job timed out", "Job exceeded its maximum duration.", "stopped")
			}
		}()
	}

	jobRepo := w.pool.jobRepo
	_ = jobRepo
	ctx := jobCtx
	_ = ctx

	switch job.Type {
	case "command":
		w.processCommandWithControl(job, &payload, jobCtx)
	case "container_action":
		w.processContainerActionWithControl(job, &payload, jobCtx)
	case "service_action":
		w.processServiceActionWithControl(job, &payload, jobCtx)
	case "update":
		w.processUpdate(job, &payload)
	default:
		slog.Error("unknown job type", "job_id", job.ID, "type", job.Type)
		w.pool.jobRepo.MarkFailed(w.pool.ctx, job.ID)
		w.pool.logs.Create(w.pool.ctx, "error", "worker", "Unknown job type", logMeta(job.ID, job.Type, ""))
	}
}

func (w *worker) isCancelled(jobID string) bool {
	job, err := w.pool.jobRepo.FindByID(w.pool.ctx, jobID)
	if err != nil || job == nil {
		return false
	}
	return job.Status == "cancelled" || job.CancelledAt != nil
}

func (w *worker) dispatchCommand(deviceID, userID, command string, groupActionID *string) (string, error) {
	cmd, err := w.pool.cmdRepo.Create(w.pool.ctx, deviceID, userID, command)
	if err != nil {
		return "", err
	}
	if groupActionID != nil {
		w.pool.grpAct.LinkCommand(w.pool.ctx, *groupActionID, cmd.ID, deviceID)
	}
	w.pool.hub.Broadcast(ws.Message{
		Type:     "command.requested",
		DeviceID: deviceID,
		Data: map[string]interface{}{
			"command_id": cmd.ID,
			"command":    cmd.Command,
			"status":     cmd.Status,
		},
	})
	return cmd.ID, nil
}

func (w *worker) monitorGroupAction(jobID, groupActionID string, totalDevices int) {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-w.pool.ctx.Done():
			return
		case <-ticker.C:
			if w.isCancelled(jobID) {
				slog.Info("job cancelled, stopping monitor", "job_id", jobID)
				return
			}

			ga, err := w.pool.grpAct.FindByID(w.pool.ctx, groupActionID)
			if err != nil || ga == nil {
				continue
			}

			w.pool.jobRepo.UpdateProgress(w.pool.ctx, jobID, ga.CompletedDevices, ga.FailedDevices)

			if ga.Status == "completed" {
				slog.Info("job completed", "job_id", jobID)
				w.pool.audits.Create(w.pool.ctx, "system", "worker", "job.completed", "job", jobID)
				w.pool.logs.Create(w.pool.ctx, "info", "worker", "Job completed", logMeta(jobID, "", ""))
				w.pool.hub.Broadcast(ws.Message{
					Type: "job.completed",
					Data: map[string]interface{}{
						"job_id": jobID,
					},
				})
				if w.pool.notifier != nil {
					if job, err := w.pool.jobRepo.FindByID(w.pool.ctx, jobID); err == nil && job != nil {
						w.pool.notifyJobStatus(job, "Job completed", "Job finished successfully.", "completed")
					}
				}
				return
			}
		}
	}
}
