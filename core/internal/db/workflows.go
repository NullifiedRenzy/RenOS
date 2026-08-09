package db

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Workflow struct {
	ID            string          `json:"id"`
	Name          string          `json:"name"`
	Description   string          `json:"description"`
	Enabled       bool            `json:"enabled"`
	Version       int             `json:"version"`
	TriggerType   string          `json:"trigger_type"`
	TriggerConfig json.RawMessage `json:"trigger_config"`
	Tags          json.RawMessage `json:"tags"`
	CreatedBy     *string         `json:"created_by,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

type WorkflowStep struct {
	ID           string          `json:"id"`
	WorkflowID   string          `json:"workflow_id"`
	StepOrder    int             `json:"step_order"`
	Name         string          `json:"name"`
	StepType     string          `json:"step_type"`
	Config       json.RawMessage `json:"config"`
	NextStepID   *string         `json:"next_step_id,omitempty"`
	ErrorHandler json.RawMessage `json:"error_handler"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

type WorkflowRun struct {
	ID            string          `json:"id"`
	WorkflowID    string          `json:"workflow_id"`
	Status        string          `json:"status"`
	TriggerType   string          `json:"trigger_type"`
	TriggerData   json.RawMessage `json:"trigger_data"`
	CurrentStepID *string         `json:"current_step_id,omitempty"`
	Context       json.RawMessage `json:"context"`
	ErrorMessage  string          `json:"error_message"`
	StartedAt     *time.Time      `json:"started_at,omitempty"`
	CompletedAt   *time.Time      `json:"completed_at,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

type WorkflowStepRun struct {
	ID           string          `json:"id"`
	RunID        string          `json:"run_id"`
	StepID       string          `json:"step_id"`
	Status       string          `json:"status"`
	Input        json.RawMessage `json:"input"`
	Output       json.RawMessage `json:"output"`
	ErrorMessage string          `json:"error_message"`
	StartedAt    *time.Time      `json:"started_at,omitempty"`
	CompletedAt  *time.Time      `json:"completed_at,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
}

type WorkflowTemplate struct {
	ID            string          `json:"id"`
	Name          string          `json:"name"`
	Description   string          `json:"description"`
	Category      string          `json:"category"`
	Icon          string          `json:"icon"`
	TriggerType   string          `json:"trigger_type"`
	TriggerConfig json.RawMessage `json:"trigger_config"`
	Steps         json.RawMessage `json:"steps"`
	Tags          json.RawMessage `json:"tags"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

type WorkflowRepository struct {
	pool *pgxpool.Pool
}

func NewWorkflowRepository(pool *pgxpool.Pool) *WorkflowRepository {
	return &WorkflowRepository{pool: pool}
}

func (r *WorkflowRepository) Create(ctx context.Context, w *Workflow) error {
	row := r.pool.QueryRow(ctx,
		`INSERT INTO workflows (name, description, enabled, trigger_type, trigger_config, tags, created_by)
		 VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING id, version, created_at, updated_at`,
		w.Name, w.Description, w.Enabled, w.TriggerType, w.TriggerConfig, w.Tags, w.CreatedBy,
	)
	return row.Scan(&w.ID, &w.Version, &w.CreatedAt, &w.UpdatedAt)
}

func (r *WorkflowRepository) FindByID(ctx context.Context, id string) (*Workflow, error) {
	w := &Workflow{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, name, description, enabled, version, trigger_type, trigger_config, tags, created_by, created_at, updated_at
		 FROM workflows WHERE id = $1`, id,
	).Scan(&w.ID, &w.Name, &w.Description, &w.Enabled, &w.Version, &w.TriggerType, &w.TriggerConfig, &w.Tags, &w.CreatedBy, &w.CreatedAt, &w.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return w, nil
}

func (r *WorkflowRepository) ListAll(ctx context.Context) ([]*Workflow, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, name, description, enabled, version, trigger_type, trigger_config, tags, created_by, created_at, updated_at
		 FROM workflows ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*Workflow
	for rows.Next() {
		w := &Workflow{}
		if err := rows.Scan(&w.ID, &w.Name, &w.Description, &w.Enabled, &w.Version, &w.TriggerType, &w.TriggerConfig, &w.Tags, &w.CreatedBy, &w.CreatedAt, &w.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, w)
	}
	return list, rows.Err()
}

func (r *WorkflowRepository) ListEnabled(ctx context.Context) ([]*Workflow, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, name, description, enabled, version, trigger_type, trigger_config, tags, created_by, created_at, updated_at
		 FROM workflows WHERE enabled = true ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*Workflow
	for rows.Next() {
		w := &Workflow{}
		if err := rows.Scan(&w.ID, &w.Name, &w.Description, &w.Enabled, &w.Version, &w.TriggerType, &w.TriggerConfig, &w.Tags, &w.CreatedBy, &w.CreatedAt, &w.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, w)
	}
	return list, rows.Err()
}

func (r *WorkflowRepository) Update(ctx context.Context, w *Workflow) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE workflows SET name=$1, description=$2, enabled=$3, trigger_type=$4, trigger_config=$5, tags=$6, version=version+1, updated_at=NOW()
		 WHERE id=$7`,
		w.Name, w.Description, w.Enabled, w.TriggerType, w.TriggerConfig, w.Tags, w.ID,
	)
	return err
}

func (r *WorkflowRepository) Delete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM workflows WHERE id = $1`, id)
	return err
}

func (r *WorkflowRepository) ToggleEnabled(ctx context.Context, id string, enabled bool) error {
	_, err := r.pool.Exec(ctx, `UPDATE workflows SET enabled = $1, updated_at = NOW() WHERE id = $2`, enabled, id)
	return err
}
