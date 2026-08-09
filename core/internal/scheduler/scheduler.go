package scheduler

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/gorhill/cronexpr"

	"github.com/renos-project/renos/core/internal/db"
	"github.com/renos-project/renos/core/internal/policy"
	"github.com/renos-project/renos/core/internal/ws"
)

type Scheduler struct {
	schedRepo  *db.ScheduleRepository
	runRepo    *db.ScheduleRunRepository
	jobRepo    *db.JobRepository
	grpAct     *db.GroupActionRepository
	grpRepo    *db.GroupRepository
	audits     *db.AuditRepository
	logs       *db.SystemLogRepository
	hub        *ws.Hub
	enforcer   *policy.Enforcer
	interval   time.Duration
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	mu         sync.Mutex
	running    map[string]struct{}
	sem        chan struct{} // limits concurrent schedule executions
}

func New(schedRepo *db.ScheduleRepository, runRepo *db.ScheduleRunRepository, jobRepo *db.JobRepository, grpAct *db.GroupActionRepository, grpRepo *db.GroupRepository, audits *db.AuditRepository, logs *db.SystemLogRepository, hub *ws.Hub, enforcer *policy.Enforcer) *Scheduler {
	ctx, cancel := context.WithCancel(context.Background())
	return &Scheduler{
		schedRepo: schedRepo,
		runRepo:   runRepo,
		jobRepo:   jobRepo,
		grpAct:    grpAct,
		grpRepo:   grpRepo,
		audits:    audits,
		logs:      logs,
		hub:       hub,
		enforcer:  enforcer,
		interval:  10 * time.Second,
		ctx:       ctx,
		cancel:    cancel,
		running:   make(map[string]struct{}),
		sem:       make(chan struct{}, 20), // max 20 concurrent executions
	}
}

func (s *Scheduler) Start() {
	s.wg.Add(1)
	go s.loop()
	slog.Info("scheduler started", "interval", s.interval)
}

func (s *Scheduler) Stop() {
	slog.Info("scheduler stopping")
	s.cancel()
	s.wg.Wait()
	slog.Info("scheduler stopped")
}

func (s *Scheduler) loop() {
	defer s.wg.Done()

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	// Run once immediately on startup
	s.tick()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.tick()
		}
	}
}

func (s *Scheduler) tick() {
	now := time.Now().UTC()

	schedules, err := s.schedRepo.ListEnabledDue(s.ctx, now)
	if err != nil {
		slog.Error("scheduler: failed to list due schedules", "error", err)
		return
	}

	for _, schedule := range schedules {
		if !s.tryLock(schedule.ID) {
			continue
		}
		// Acquire semaphore to limit concurrent executions
		s.sem <- struct{}{}
		go func(sched *db.Schedule) {
			defer func() { <-s.sem }()
			s.execute(sched, now)
		}(schedule)
	}
}

func (s *Scheduler) tryLock(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.running[id]; ok {
		return false
	}
	s.running[id] = struct{}{}
	return true
}

func (s *Scheduler) unlock(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.running, id)
}

func (s *Scheduler) execute(schedule *db.Schedule, now time.Time) {
	defer s.unlock(schedule.ID)

	slog.Info("scheduler: executing schedule", "id", schedule.ID, "name", schedule.Name)

	// 1. Resolve target device IDs
	deviceIDs, err := s.resolveTargets(schedule)
	if err != nil {
		slog.Error("scheduler: failed to resolve targets", "schedule_id", schedule.ID, "error", err)
		return
	}
	if len(deviceIDs) == 0 {
		slog.Warn("scheduler: schedule has no target devices", "schedule_id", schedule.ID)
		// Still advance next_run to avoid busy-looping
		s.advanceSchedule(schedule, now)
		return
	}

	// 2. Evaluate policies before creating the job
	if r := s.enforcer.Evaluate(s.ctx, &policy.EvalContext{
		ActionType: schedule.Type,
		DeviceIDs:  deviceIDs,
		CronExpr:   schedule.CronExpression,
	}); r.Action == "deny" {
		slog.Warn("scheduler: policy blocked schedule execution", "schedule_id", schedule.ID, "reason", r.Message)
		s.advanceSchedule(schedule, now)
		return
	}

	// 3. Create group_action
	groupID := s.extractGroupID(schedule)
	targetValue := s.extractTargetValue(schedule)

	ga, err := s.grpAct.Create(s.ctx, groupID, schedule.Type, schedule.TargetType, targetValue, schedule.CreatedBy, len(deviceIDs))
	if err != nil {
		slog.Error("scheduler: failed to create group action", "schedule_id", schedule.ID, "error", err)
		return
	}

	// 4. Create job for the worker system
	payload := s.buildJobPayload(schedule, deviceIDs)
	payloadBytes, _ := json.Marshal(payload)

	job, err := s.jobRepo.Create(s.ctx, schedule.Type, payloadBytes, len(deviceIDs), schedule.CreatedBy, &ga.ID)
	if err != nil {
		slog.Error("scheduler: failed to create job", "schedule_id", schedule.ID, "error", err)
		return
	}

	// 5. Record schedule run
	s.runRepo.Create(s.ctx, schedule.ID, job.ID, "pending")

	// 6. Audit
	s.audits.Create(s.ctx, "system", "scheduler", "schedule.run", "schedule", schedule.ID)

	// 7. Log and broadcast
	meta, _ := json.Marshal(map[string]interface{}{
		"schedule_id":   schedule.ID,
		"schedule_name": schedule.Name,
		"job_id":        job.ID,
		"device_count":  len(deviceIDs),
	})
	s.logs.Create(s.ctx, "info", "scheduler", "Schedule triggered: "+schedule.Name, meta)
	s.hub.Broadcast(ws.Message{
		Type: "schedule.triggered",
		Data: map[string]interface{}{
			"schedule_id":   schedule.ID,
			"schedule_name": schedule.Name,
			"job_id":        job.ID,
			"type":          schedule.Type,
			"device_count":  len(deviceIDs),
		},
	})

	// 8. Update schedule timestamps
	s.advanceSchedule(schedule, now)
}

func (s *Scheduler) advanceSchedule(schedule *db.Schedule, now time.Time) {
	nextRun := s.computeNextRun(schedule.CronExpression, now)
	s.schedRepo.UpdateRunTimes(s.ctx, schedule.ID, now, nextRun)
}

func (s *Scheduler) resolveTargets(schedule *db.Schedule) ([]string, error) {
	var targetIDs []string
	if err := json.Unmarshal(schedule.TargetIDs, &targetIDs); err != nil {
		return nil, err
	}

	switch schedule.TargetType {
	case "device":
		return targetIDs, nil
	case "group":
		if len(targetIDs) == 0 {
			return nil, nil
		}
		return s.grpRepo.ListDeviceIDsByGroupIDs(s.ctx, targetIDs)
	case "multi":
		return targetIDs, nil
	default:
		return targetIDs, nil
	}
}

func (s *Scheduler) extractGroupID(schedule *db.Schedule) *string {
	if schedule.TargetType != "group" {
		return nil
	}
	var targetIDs []string
	if err := json.Unmarshal(schedule.TargetIDs, &targetIDs); err != nil || len(targetIDs) == 0 {
		return nil
	}
	return &targetIDs[0]
}

func (s *Scheduler) extractTargetValue(schedule *db.Schedule) string {
	switch schedule.Type {
	case "command":
		var payload struct {
			Command string `json:"command"`
		}
		if err := json.Unmarshal(schedule.Payload, &payload); err != nil {
			return ""
		}
		return payload.Command
	case "container_action":
		var payload struct {
			ContainerID string `json:"container_id"`
			Action      string `json:"action"`
		}
		if err := json.Unmarshal(schedule.Payload, &payload); err != nil {
			return ""
		}
		return payload.ContainerID + ":" + payload.Action
	case "service_action":
		var payload struct {
			UnitName string `json:"unit_name"`
			Action   string `json:"action"`
		}
		if err := json.Unmarshal(schedule.Payload, &payload); err != nil {
			return ""
		}
		return payload.UnitName + ":" + payload.Action
	case "update":
		var payload struct {
			Action string `json:"action"`
		}
		if err := json.Unmarshal(schedule.Payload, &payload); err != nil {
			return ""
		}
		return payload.Action
	}
	return ""
}

func (s *Scheduler) buildJobPayload(schedule *db.Schedule, deviceIDs []string) map[string]interface{} {
	p := map[string]interface{}{
		"device_ids": deviceIDs,
	}

	var extra map[string]interface{}
	if err := json.Unmarshal(schedule.Payload, &extra); err == nil {
		for k, v := range extra {
			p[k] = v
		}
	}

	if schedule.TargetType == "group" {
		var targetIDs []string
		if err := json.Unmarshal(schedule.TargetIDs, &targetIDs); err == nil && len(targetIDs) > 0 {
			p["group_id"] = targetIDs[0]
		}
	}

	return p
}

func (s *Scheduler) computeNextRun(cronExpr string, after time.Time) time.Time {
	expr, err := cronexpr.Parse(cronExpr)
	if err != nil {
		slog.Error("scheduler: invalid cron expression", "cron", cronExpr, "error", err)
		return after.Add(10 * time.Second)
	}
	return expr.Next(after)
}

func ValidateCronExpression(expr string) error {
	_, err := cronexpr.Parse(expr)
	return err
}
