package db

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type WorkflowStepRepository struct {
	pool *pgxpool.Pool
}

func NewWorkflowStepRepository(pool *pgxpool.Pool) *WorkflowStepRepository {
	return &WorkflowStepRepository{pool: pool}
}

func (r *WorkflowStepRepository) Create(ctx context.Context, s *WorkflowStep) error {
	row := r.pool.QueryRow(ctx,
		`INSERT INTO workflow_steps (workflow_id, step_order, name, step_type, config, next_step_id, error_handler)
		 VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING id, created_at, updated_at`,
		s.WorkflowID, s.StepOrder, s.Name, s.StepType, s.Config, s.NextStepID, s.ErrorHandler,
	)
	return row.Scan(&s.ID, &s.CreatedAt, &s.UpdatedAt)
}

func (r *WorkflowStepRepository) ListByWorkflow(ctx context.Context, workflowID string) ([]*WorkflowStep, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, workflow_id, step_order, name, step_type, config, next_step_id, error_handler, created_at, updated_at
		 FROM workflow_steps WHERE workflow_id = $1 ORDER BY step_order`, workflowID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*WorkflowStep
	for rows.Next() {
		s := &WorkflowStep{}
		if err := rows.Scan(&s.ID, &s.WorkflowID, &s.StepOrder, &s.Name, &s.StepType, &s.Config, &s.NextStepID, &s.ErrorHandler, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, s)
	}
	return list, rows.Err()
}

func (r *WorkflowStepRepository) FindByID(ctx context.Context, id string) (*WorkflowStep, error) {
	s := &WorkflowStep{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, workflow_id, step_order, name, step_type, config, next_step_id, error_handler, created_at, updated_at
		 FROM workflow_steps WHERE id = $1`, id,
	).Scan(&s.ID, &s.WorkflowID, &s.StepOrder, &s.Name, &s.StepType, &s.Config, &s.NextStepID, &s.ErrorHandler, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (r *WorkflowStepRepository) DeleteByWorkflow(ctx context.Context, workflowID string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM workflow_steps WHERE workflow_id = $1`, workflowID)
	return err
}

type WorkflowRunRepository struct {
	pool *pgxpool.Pool
}

func NewWorkflowRunRepository(pool *pgxpool.Pool) *WorkflowRunRepository {
	return &WorkflowRunRepository{pool: pool}
}

func (r *WorkflowRunRepository) Create(ctx context.Context, run *WorkflowRun) error {
	row := r.pool.QueryRow(ctx,
		`INSERT INTO workflow_runs (workflow_id, status, trigger_type, trigger_data, context)
		 VALUES ($1,$2,$3,$4,$5) RETURNING id, created_at, updated_at`,
		run.WorkflowID, run.Status, run.TriggerType, run.TriggerData, run.Context,
	)
	return row.Scan(&run.ID, &run.CreatedAt, &run.UpdatedAt)
}

func (r *WorkflowRunRepository) FindByID(ctx context.Context, id string) (*WorkflowRun, error) {
	run := &WorkflowRun{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, workflow_id, status, trigger_type, trigger_data, current_step_id, context, error_message, started_at, completed_at, created_at, updated_at
		 FROM workflow_runs WHERE id = $1`, id,
	).Scan(&run.ID, &run.WorkflowID, &run.Status, &run.TriggerType, &run.TriggerData, &run.CurrentStepID, &run.Context, &run.ErrorMessage, &run.StartedAt, &run.CompletedAt, &run.CreatedAt, &run.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return run, nil
}

func (r *WorkflowRunRepository) ListByWorkflow(ctx context.Context, workflowID string, limit int) ([]*WorkflowRun, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id, workflow_id, status, trigger_type, trigger_data, current_step_id, context, error_message, started_at, completed_at, created_at, updated_at
		 FROM workflow_runs WHERE workflow_id = $1 ORDER BY created_at DESC LIMIT $2`, workflowID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*WorkflowRun
	for rows.Next() {
		run := &WorkflowRun{}
		if err := rows.Scan(&run.ID, &run.WorkflowID, &run.Status, &run.TriggerType, &run.TriggerData, &run.CurrentStepID, &run.Context, &run.ErrorMessage, &run.StartedAt, &run.CompletedAt, &run.CreatedAt, &run.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, run)
	}
	return list, rows.Err()
}

func (r *WorkflowRunRepository) ListAll(ctx context.Context, limit int) ([]*WorkflowRun, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id, workflow_id, status, trigger_type, trigger_data, current_step_id, context, error_message, started_at, completed_at, created_at, updated_at
		 FROM workflow_runs ORDER BY created_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*WorkflowRun
	for rows.Next() {
		run := &WorkflowRun{}
		if err := rows.Scan(&run.ID, &run.WorkflowID, &run.Status, &run.TriggerType, &run.TriggerData, &run.CurrentStepID, &run.Context, &run.ErrorMessage, &run.StartedAt, &run.CompletedAt, &run.CreatedAt, &run.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, run)
	}
	return list, rows.Err()
}

func (r *WorkflowRunRepository) UpdateStatus(ctx context.Context, id, status string) error {
	_, err := r.pool.Exec(ctx, `UPDATE workflow_runs SET status = $1, updated_at = NOW() WHERE id = $2`, status, id)
	return err
}

func (r *WorkflowRunRepository) SetCurrentStep(ctx context.Context, id, stepID string) error {
	_, err := r.pool.Exec(ctx, `UPDATE workflow_runs SET current_step_id = $1, updated_at = NOW() WHERE id = $2`, stepID, id)
	return err
}

func (r *WorkflowRunRepository) Complete(ctx context.Context, id, status, errMsg string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE workflow_runs SET status = $1, error_message = $2, completed_at = NOW(), updated_at = NOW() WHERE id = $3`,
		status, errMsg, id,
	)
	return err
}

func (r *WorkflowRunRepository) Start(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE workflow_runs SET status = 'running', started_at = NOW(), updated_at = NOW() WHERE id = $1`, id,
	)
	return err
}

type WorkflowStepRunRepository struct {
	pool *pgxpool.Pool
}

func NewWorkflowStepRunRepository(pool *pgxpool.Pool) *WorkflowStepRunRepository {
	return &WorkflowStepRunRepository{pool: pool}
}

func (r *WorkflowStepRunRepository) Create(ctx context.Context, sr *WorkflowStepRun) error {
	row := r.pool.QueryRow(ctx,
		`INSERT INTO workflow_step_runs (run_id, step_id, status, input)
		 VALUES ($1,$2,$3,$4) RETURNING id, created_at`,
		sr.RunID, sr.StepID, sr.Status, sr.Input,
	)
	return row.Scan(&sr.ID, &sr.CreatedAt)
}

func (r *WorkflowStepRunRepository) ListByRun(ctx context.Context, runID string) ([]*WorkflowStepRun, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, run_id, step_id, status, input, output, error_message, started_at, completed_at, created_at
		 FROM workflow_step_runs WHERE run_id = $1 ORDER BY created_at`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*WorkflowStepRun
	for rows.Next() {
		sr := &WorkflowStepRun{}
		if err := rows.Scan(&sr.ID, &sr.RunID, &sr.StepID, &sr.Status, &sr.Input, &sr.Output, &sr.ErrorMessage, &sr.StartedAt, &sr.CompletedAt, &sr.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, sr)
	}
	return list, rows.Err()
}

func (r *WorkflowStepRunRepository) Complete(ctx context.Context, id, status, output, errMsg string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE workflow_step_runs SET status = $1, output = $2, error_message = $3, completed_at = NOW() WHERE id = $4`,
		status, output, errMsg, id,
	)
	return err
}

func (r *WorkflowStepRunRepository) Start(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE workflow_step_runs SET status = 'running', started_at = NOW() WHERE id = $1`, id,
	)
	return err
}

type WorkflowTemplateRepository struct {
	pool *pgxpool.Pool
}

func NewWorkflowTemplateRepository(pool *pgxpool.Pool) *WorkflowTemplateRepository {
	return &WorkflowTemplateRepository{pool: pool}
}

func (r *WorkflowTemplateRepository) Create(ctx context.Context, t *WorkflowTemplate) error {
	row := r.pool.QueryRow(ctx,
		`INSERT INTO workflow_templates (name, description, category, icon, trigger_type, trigger_config, steps, tags)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8) RETURNING id, created_at, updated_at`,
		t.Name, t.Description, t.Category, t.Icon, t.TriggerType, t.TriggerConfig, t.Steps, t.Tags,
	)
	return row.Scan(&t.ID, &t.CreatedAt, &t.UpdatedAt)
}

func (r *WorkflowTemplateRepository) ListAll(ctx context.Context) ([]*WorkflowTemplate, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, name, description, category, icon, trigger_type, trigger_config, steps, tags, created_at, updated_at
		 FROM workflow_templates ORDER BY category, name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*WorkflowTemplate
	for rows.Next() {
		t := &WorkflowTemplate{}
		if err := rows.Scan(&t.ID, &t.Name, &t.Description, &t.Category, &t.Icon, &t.TriggerType, &t.TriggerConfig, &t.Steps, &t.Tags, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, t)
	}
	return list, rows.Err()
}

func (r *WorkflowTemplateRepository) FindByID(ctx context.Context, id string) (*WorkflowTemplate, error) {
	t := &WorkflowTemplate{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, name, description, category, icon, trigger_type, trigger_config, steps, tags, created_at, updated_at
		 FROM workflow_templates WHERE id = $1`, id,
	).Scan(&t.ID, &t.Name, &t.Description, &t.Category, &t.Icon, &t.TriggerType, &t.TriggerConfig, &t.Steps, &t.Tags, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (r *WorkflowTemplateRepository) Delete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM workflow_templates WHERE id = $1`, id)
	return err
}
