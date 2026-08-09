package db

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ClusterNode struct {
	ID            string          `json:"id"`
	NodeID        string          `json:"node_id"`
	Hostname      string          `json:"hostname"`
	Address       string          `json:"address"`
	Port          int             `json:"port"`
	Role          string          `json:"role"`
	Status        string          `json:"status"`
	Metadata      json.RawMessage `json:"metadata"`
	LastHeartbeat time.Time       `json:"last_heartbeat"`
	JoinedAt      time.Time       `json:"joined_at"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

type ClusterElection struct {
	ID           string     `json:"id"`
	CandidateID  string     `json:"candidate_id"`
	Term         int        `json:"term"`
	VotesFor     int        `json:"votes_for"`
	VotesAgainst int        `json:"votes_against"`
	Status       string     `json:"status"`
	StartedAt    time.Time  `json:"started_at"`
	ResolvedAt   *time.Time `json:"resolved_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

type ClusterTask struct {
	ID           string          `json:"id"`
	TaskType     string          `json:"task_type"`
	TaskData     json.RawMessage `json:"task_data"`
	AssignedTo   *string         `json:"assigned_to,omitempty"`
	Status       string          `json:"status"`
	Priority     int             `json:"priority"`
	MaxRetries   int             `json:"max_retries"`
	RetryCount   int             `json:"retry_count"`
	Result       json.RawMessage `json:"result"`
	ErrorMessage string          `json:"error_message"`
	CreatedAt    time.Time       `json:"created_at"`
	StartedAt    *time.Time      `json:"started_at,omitempty"`
	CompletedAt  *time.Time      `json:"completed_at,omitempty"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

type ClusterState struct {
	Key         string          `json:"key"`
	Value       json.RawMessage `json:"value"`
	Version     int             `json:"version"`
	Owner       string          `json:"owner"`
	TTLSeconds  int             `json:"ttl_seconds"`
	ExpiresAt   *time.Time      `json:"expires_at,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

type ClusterMetric struct {
	ID                string    `json:"id"`
	NodeID            string    `json:"node_id"`
	CPUUsage          float64   `json:"cpu_usage"`
	MemoryUsage       float64   `json:"memory_usage"`
	DiskUsage         float64   `json:"disk_usage"`
	ActiveConnections int       `json:"active_connections"`
	TaskCount         int       `json:"task_count"`
	RecordedAt        time.Time `json:"recorded_at"`
}

// --- Cluster Node Repository ---

type ClusterNodeRepository struct{ pool *pgxpool.Pool }

func NewClusterNodeRepository(pool *pgxpool.Pool) *ClusterNodeRepository {
	return &ClusterNodeRepository{pool: pool}
}

func (r *ClusterNodeRepository) Register(ctx context.Context, n *ClusterNode) error {
	row := r.pool.QueryRow(ctx,
		`INSERT INTO cluster_nodes (node_id, hostname, address, port, role, status, metadata)
		 VALUES ($1,$2,$3,$4,$5,$6,$7)
		 ON CONFLICT (node_id) DO UPDATE SET hostname=$2, address=$3, port=$4, status=$6, metadata=$7, updated_at=NOW()
		 RETURNING id, joined_at, created_at, updated_at`,
		n.NodeID, n.Hostname, n.Address, n.Port, n.Role, n.Status, n.Metadata,
	)
	return row.Scan(&n.ID, &n.JoinedAt, &n.CreatedAt, &n.UpdatedAt)
}

func (r *ClusterNodeRepository) Heartbeat(ctx context.Context, nodeID string) error {
	_, err := r.pool.Exec(ctx, `UPDATE cluster_nodes SET last_heartbeat=NOW(), updated_at=NOW() WHERE node_id=$1`, nodeID)
	return err
}

func (r *ClusterNodeRepository) ListAll(ctx context.Context) ([]*ClusterNode, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, node_id, hostname, address, port, role, status, metadata, last_heartbeat, joined_at, created_at, updated_at FROM cluster_nodes ORDER BY joined_at`)
	if err != nil { return nil, err }
	defer rows.Close()
	return scanClusterNodes(rows)
}

func (r *ClusterNodeRepository) FindByNodeID(ctx context.Context, nodeID string) (*ClusterNode, error) {
	n := &ClusterNode{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, node_id, hostname, address, port, role, status, metadata, last_heartbeat, joined_at, created_at, updated_at FROM cluster_nodes WHERE node_id=$1`, nodeID,
	).Scan(&n.ID, &n.NodeID, &n.Hostname, &n.Address, &n.Port, &n.Role, &n.Status, &n.Metadata, &n.LastHeartbeat, &n.JoinedAt, &n.CreatedAt, &n.UpdatedAt)
	if err != nil { return nil, err }
	return n, nil
}

func (r *ClusterNodeRepository) UpdateRole(ctx context.Context, nodeID, role string) error {
	_, err := r.pool.Exec(ctx, `UPDATE cluster_nodes SET role=$1, updated_at=NOW() WHERE node_id=$2`, role, nodeID)
	return err
}

func (r *ClusterNodeRepository) UpdateStatus(ctx context.Context, nodeID, status string) error {
	_, err := r.pool.Exec(ctx, `UPDATE cluster_nodes SET status=$1, updated_at=NOW() WHERE node_id=$2`, status, nodeID)
	return err
}

func (r *ClusterNodeRepository) MarkOffline(ctx context.Context, threshold time.Duration) ([]string, error) {
	thresholdTime := time.Now().Add(-threshold)
	rows, err := r.pool.Query(ctx, `UPDATE cluster_nodes SET status='offline', updated_at=NOW() WHERE last_heartbeat < $1 AND status != 'offline' RETURNING node_id`, thresholdTime)
	if err != nil { return nil, err }
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil { ids = append(ids, id) }
	}
	return ids, rows.Err()
}

func (r *ClusterNodeRepository) FindLeader(ctx context.Context) (*ClusterNode, error) {
	n := &ClusterNode{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, node_id, hostname, address, port, role, status, metadata, last_heartbeat, joined_at, created_at, updated_at FROM cluster_nodes WHERE role='leader' LIMIT 1`,
	).Scan(&n.ID, &n.NodeID, &n.Hostname, &n.Address, &n.Port, &n.Role, &n.Status, &n.Metadata, &n.LastHeartbeat, &n.JoinedAt, &n.CreatedAt, &n.UpdatedAt)
	if err != nil { return nil, err }
	return n, nil
}

func (r *ClusterNodeRepository) Delete(ctx context.Context, nodeID string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM cluster_nodes WHERE node_id=$1`, nodeID)
	return err
}

func scanClusterNodes(rows pgx.Rows) ([]*ClusterNode, error) {
	var list []*ClusterNode
	for rows.Next() {
		n := &ClusterNode{}
		if err := rows.Scan(&n.ID, &n.NodeID, &n.Hostname, &n.Address, &n.Port, &n.Role, &n.Status, &n.Metadata, &n.LastHeartbeat, &n.JoinedAt, &n.CreatedAt, &n.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, n)
	}
	return list, rows.Err()
}

// --- Cluster Election Repository ---

type ClusterElectionRepository struct{ pool *pgxpool.Pool }

func NewClusterElectionRepository(pool *pgxpool.Pool) *ClusterElectionRepository {
	return &ClusterElectionRepository{pool: pool}
}

func (r *ClusterElectionRepository) Create(ctx context.Context, e *ClusterElection) error {
	row := r.pool.QueryRow(ctx,
		`INSERT INTO cluster_elections (candidate_id, term) VALUES ($1,$2) RETURNING id, started_at, created_at`,
		e.CandidateID, e.Term,
	)
	return row.Scan(&e.ID, &e.StartedAt, &e.CreatedAt)
}

func (r *ClusterElectionRepository) UpdateResult(ctx context.Context, id, status string, votesFor, votesAgainst int) error {
	_, err := r.pool.Exec(ctx, `UPDATE cluster_elections SET status=$1, votes_for=$2, votes_against=$3, resolved_at=NOW() WHERE id=$4`, status, votesFor, votesAgainst, id)
	return err
}

func (r *ClusterElectionRepository) CurrentTerm(ctx context.Context) (int, error) {
	var term int
	err := r.pool.QueryRow(ctx, `SELECT COALESCE(MAX(term), 0) FROM cluster_elections`).Scan(&term)
	return term, err
}

func (r *ClusterElectionRepository) ListRecent(ctx context.Context, limit int) ([]*ClusterElection, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, candidate_id, term, votes_for, votes_against, status, started_at, resolved_at, created_at FROM cluster_elections ORDER BY created_at DESC LIMIT $1`, limit)
	if err != nil { return nil, err }
	defer rows.Close()
	var list []*ClusterElection
	for rows.Next() {
		e := &ClusterElection{}
		if err := rows.Scan(&e.ID, &e.CandidateID, &e.Term, &e.VotesFor, &e.VotesAgainst, &e.Status, &e.StartedAt, &e.ResolvedAt, &e.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, e)
	}
	return list, rows.Err()
}

// --- Cluster Task Repository ---

type ClusterTaskRepository struct{ pool *pgxpool.Pool }

func NewClusterTaskRepository(pool *pgxpool.Pool) *ClusterTaskRepository {
	return &ClusterTaskRepository{pool: pool}
}

func (r *ClusterTaskRepository) Create(ctx context.Context, t *ClusterTask) error {
	row := r.pool.QueryRow(ctx,
		`INSERT INTO cluster_tasks (task_type, task_data, priority, max_retries)
		 VALUES ($1,$2,$3,$4) RETURNING id, created_at, updated_at`,
		t.TaskType, t.TaskData, t.Priority, t.MaxRetries,
	)
	return row.Scan(&t.ID, &t.CreatedAt, &t.UpdatedAt)
}

func (r *ClusterTaskRepository) Claim(ctx context.Context, taskID, nodeID string) error {
	result, err := r.pool.Exec(ctx, `UPDATE cluster_tasks SET assigned_to=$1, status='assigned', updated_at=NOW() WHERE id=$2 AND status='pending'`, nodeID, taskID)
	if err != nil { return err }
	if result.RowsAffected() == 0 { return errors.New("not found or already claimed") }
	return nil
}

func (r *ClusterTaskRepository) Start(ctx context.Context, taskID string) error {
	_, err := r.pool.Exec(ctx, `UPDATE cluster_tasks SET status='running', started_at=NOW(), updated_at=NOW() WHERE id=$1`, taskID)
	return err
}

func (r *ClusterTaskRepository) Complete(ctx context.Context, taskID string, result json.RawMessage) error {
	_, err := r.pool.Exec(ctx, `UPDATE cluster_tasks SET status='completed', result=$1, completed_at=NOW(), updated_at=NOW() WHERE id=$2`, result, taskID)
	return err
}

func (r *ClusterTaskRepository) Fail(ctx context.Context, taskID, errMsg string) error {
	_, err := r.pool.Exec(ctx, `UPDATE cluster_tasks SET status='failed', error_message=$1, updated_at=NOW() WHERE id=$2`, errMsg, taskID)
	return err
}

func (r *ClusterTaskRepository) ListPending(ctx context.Context, limit int) ([]*ClusterTask, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, task_type, task_data, assigned_to, status, priority, max_retries, retry_count, result, error_message, created_at, started_at, completed_at, updated_at FROM cluster_tasks WHERE status='pending' ORDER BY priority DESC, created_at ASC LIMIT $1`, limit)
	if err != nil { return nil, err }
	defer rows.Close()
	return scanClusterTasks(rows)
}

func (r *ClusterTaskRepository) ListAll(ctx context.Context, limit int) ([]*ClusterTask, error) {
	if limit <= 0 { limit = 50 }
	rows, err := r.pool.Query(ctx, `SELECT id, task_type, task_data, assigned_to, status, priority, max_retries, retry_count, result, error_message, created_at, started_at, completed_at, updated_at FROM cluster_tasks ORDER BY created_at DESC LIMIT $1`, limit)
	if err != nil { return nil, err }
	defer rows.Close()
	return scanClusterTasks(rows)
}

func (r *ClusterTaskRepository) Rebalance(ctx context.Context) (int64, error) {
	result, err := r.pool.Exec(ctx, `UPDATE cluster_tasks SET assigned_to=NULL, status='pending' WHERE status='assigned' AND assigned_to IN (SELECT node_id FROM cluster_nodes WHERE status='offline')`)
	if err != nil { return 0, err }
	return result.RowsAffected(), nil
}

func scanClusterTasks(rows pgx.Rows) ([]*ClusterTask, error) {
	var list []*ClusterTask
	for rows.Next() {
		t := &ClusterTask{}
		if err := rows.Scan(&t.ID, &t.TaskType, &t.TaskData, &t.AssignedTo, &t.Status, &t.Priority, &t.MaxRetries, &t.RetryCount, &t.Result, &t.ErrorMessage, &t.CreatedAt, &t.StartedAt, &t.CompletedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, t)
	}
	return list, rows.Err()
}

// --- Cluster State Repository ---

type ClusterStateRepository struct{ pool *pgxpool.Pool }

func NewClusterStateRepository(pool *pgxpool.Pool) *ClusterStateRepository {
	return &ClusterStateRepository{pool: pool}
}

func (r *ClusterStateRepository) Set(ctx context.Context, s *ClusterState) error {
	var ttl interface{} = nil
	if s.TTLSeconds > 0 {
		expires := time.Now().Add(time.Duration(s.TTLSeconds) * time.Second)
		s.ExpiresAt = &expires
		ttl = expires
	}
	row := r.pool.QueryRow(ctx,
		`INSERT INTO cluster_state (key, value, version, owner, ttl_seconds, expires_at)
		 VALUES ($1,$2,1,$3,$4,$5)
		 ON CONFLICT (key) DO UPDATE SET value=$2, version=cluster_state.version+1, owner=$3, updated_at=NOW(), expires_at=$5
		 RETURNING version, created_at, updated_at`,
		s.Key, s.Value, s.Owner, s.TTLSeconds, ttl,
	)
	return row.Scan(&s.Version, &s.CreatedAt, &s.UpdatedAt)
}

func (r *ClusterStateRepository) Get(ctx context.Context, key string) (*ClusterState, error) {
	s := &ClusterState{}
	err := r.pool.QueryRow(ctx,
		`SELECT key, value, version, owner, ttl_seconds, expires_at, created_at, updated_at FROM cluster_state WHERE key=$1 AND (expires_at IS NULL OR expires_at > NOW())`, key,
	).Scan(&s.Key, &s.Value, &s.Version, &s.Owner, &s.TTLSeconds, &s.ExpiresAt, &s.CreatedAt, &s.UpdatedAt)
	if err != nil { return nil, err }
	return s, nil
}

func (r *ClusterStateRepository) Delete(ctx context.Context, key string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM cluster_state WHERE key=$1`, key)
	return err
}

func (r *ClusterStateRepository) CleanupExpired(ctx context.Context) (int64, error) {
	result, err := r.pool.Exec(ctx, `DELETE FROM cluster_state WHERE expires_at IS NOT NULL AND expires_at < NOW()`)
	if err != nil { return 0, err }
	return result.RowsAffected(), nil
}

func (r *ClusterStateRepository) ListAll(ctx context.Context) ([]*ClusterState, error) {
	rows, err := r.pool.Query(ctx, `SELECT key, value, version, owner, ttl_seconds, expires_at, created_at, updated_at FROM cluster_state ORDER BY key`)
	if err != nil { return nil, err }
	defer rows.Close()
	var list []*ClusterState
	for rows.Next() {
		s := &ClusterState{}
		if err := rows.Scan(&s.Key, &s.Value, &s.Version, &s.Owner, &s.TTLSeconds, &s.ExpiresAt, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, s)
	}
	return list, rows.Err()
}

// --- Cluster Metrics Repository ---

type ClusterMetricsRepository struct{ pool *pgxpool.Pool }

func NewClusterMetricsRepository(pool *pgxpool.Pool) *ClusterMetricsRepository {
	return &ClusterMetricsRepository{pool: pool}
}

func (r *ClusterMetricsRepository) Record(ctx context.Context, m *ClusterMetric) error {
	row := r.pool.QueryRow(ctx,
		`INSERT INTO cluster_metrics (node_id, cpu_usage, memory_usage, disk_usage, active_connections, task_count)
		 VALUES ($1,$2,$3,$4,$5,$6) RETURNING id, recorded_at`,
		m.NodeID, m.CPUUsage, m.MemoryUsage, m.DiskUsage, m.ActiveConnections, m.TaskCount,
	)
	return row.Scan(&m.ID, &m.RecordedAt)
}

func (r *ClusterMetricsRepository) LatestByNode(ctx context.Context) ([]*ClusterMetric, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT DISTINCT ON (node_id) id, node_id, cpu_usage, memory_usage, disk_usage, active_connections, task_count, recorded_at
		FROM cluster_metrics ORDER BY node_id, recorded_at DESC`)
	if err != nil { return nil, err }
	defer rows.Close()
	var list []*ClusterMetric
	for rows.Next() {
		m := &ClusterMetric{}
		if err := rows.Scan(&m.ID, &m.NodeID, &m.CPUUsage, &m.MemoryUsage, &m.DiskUsage, &m.ActiveConnections, &m.TaskCount, &m.RecordedAt); err != nil {
			return nil, err
		}
		list = append(list, m)
	}
	return list, rows.Err()
}
