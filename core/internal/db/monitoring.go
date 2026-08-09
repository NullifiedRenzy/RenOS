package db

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type MonitoringMetric struct {
	ID          string          `json:"id"`
	DeviceID    *string         `json:"device_id,omitempty"`
	MetricName  string          `json:"metric_name"`
	MetricValue float64         `json:"metric_value"`
	Labels      json.RawMessage `json:"labels"`
	RecordedAt  time.Time       `json:"recorded_at"`
	CreatedAt   time.Time       `json:"created_at"`
}

type AlertRule struct {
	ID                   string          `json:"id"`
	Name                 string          `json:"name"`
	Description          string          `json:"description"`
	MetricName           string          `json:"metric_name"`
	Condition            string          `json:"condition"`
	Threshold            float64         `json:"threshold"`
	DurationSeconds      int             `json:"duration_seconds"`
	Severity             string          `json:"severity"`
	Enabled              bool            `json:"enabled"`
	NotificationChannels json.RawMessage `json:"notification_channels"`
	CreatedAt            time.Time       `json:"created_at"`
	UpdatedAt            time.Time       `json:"updated_at"`
}

type AlertInstance struct {
	ID             string     `json:"id"`
	RuleID         string     `json:"rule_id"`
	DeviceID       *string    `json:"device_id,omitempty"`
	Status         string     `json:"status"`
	Value          float64    `json:"value"`
	Message        string     `json:"message"`
	StartedAt      time.Time  `json:"started_at"`
	ResolvedAt     *time.Time `json:"resolved_at,omitempty"`
	AcknowledgedBy *string    `json:"acknowledged_by,omitempty"`
	AcknowledgedAt *time.Time `json:"acknowledged_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type MonitoringDashboard struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Layout      json.RawMessage `json:"layout"`
	IsDefault   bool            `json:"is_default"`
	CreatedBy   *string         `json:"created_by,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

type MonitoringWidget struct {
	ID          string          `json:"id"`
	DashboardID string          `json:"dashboard_id"`
	WidgetType  string          `json:"widget_type"`
	Title       string          `json:"title"`
	Config      json.RawMessage `json:"config"`
	PositionX   int             `json:"position_x"`
	PositionY   int             `json:"position_y"`
	Width       int             `json:"width"`
	Height      int             `json:"height"`
	CreatedAt   time.Time       `json:"created_at"`
}

type MonitoringLog struct {
	ID         string          `json:"id"`
	DeviceID   *string         `json:"device_id,omitempty"`
	Source     string          `json:"source"`
	Level      string          `json:"level"`
	Message    string          `json:"message"`
	Fields     json.RawMessage `json:"fields"`
	RecordedAt time.Time       `json:"recorded_at"`
	CreatedAt  time.Time       `json:"created_at"`
}

type MonitoringMetricRepository struct {
	pool *pgxpool.Pool
}

func NewMonitoringMetricRepository(pool *pgxpool.Pool) *MonitoringMetricRepository {
	return &MonitoringMetricRepository{pool: pool}
}

func (r *MonitoringMetricRepository) Record(ctx context.Context, m *MonitoringMetric) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO monitoring_metrics (device_id, metric_name, metric_value, labels, recorded_at) VALUES ($1,$2,$3,$4,$5)`,
		m.DeviceID, m.MetricName, m.MetricValue, m.Labels, m.RecordedAt,
	)
	return err
}

func (r *MonitoringMetricRepository) Query(ctx context.Context, deviceID, metricName string, since time.Time, limit int) ([]*MonitoringMetric, error) {
	if limit <= 0 {
		limit = 500
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id, device_id, metric_name, metric_value, labels, recorded_at, created_at
		 FROM monitoring_metrics WHERE metric_name = $1 AND recorded_at >= $2
		 ORDER BY recorded_at DESC LIMIT $3`, metricName, since, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*MonitoringMetric
	for rows.Next() {
		m := &MonitoringMetric{}
		if err := rows.Scan(&m.ID, &m.DeviceID, &m.MetricName, &m.MetricValue, &m.Labels, &m.RecordedAt, &m.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, m)
	}
	return list, rows.Err()
}

func (r *MonitoringMetricRepository) QueryByDevice(ctx context.Context, deviceID string, since time.Time, limit int) ([]*MonitoringMetric, error) {
	if limit <= 0 {
		limit = 500
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id, device_id, metric_name, metric_value, labels, recorded_at, created_at
		 FROM monitoring_metrics WHERE device_id = $1 AND recorded_at >= $2
		 ORDER BY recorded_at DESC LIMIT $3`, deviceID, since, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*MonitoringMetric
	for rows.Next() {
		m := &MonitoringMetric{}
		if err := rows.Scan(&m.ID, &m.DeviceID, &m.MetricName, &m.MetricValue, &m.Labels, &m.RecordedAt, &m.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, m)
	}
	return list, rows.Err()
}

func (r *MonitoringMetricRepository) LatestByDevice(ctx context.Context, deviceID string) ([]*MonitoringMetric, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT DISTINCT ON (metric_name) id, device_id, metric_name, metric_value, labels, recorded_at, created_at
		 FROM monitoring_metrics WHERE device_id = $1 ORDER BY metric_name, recorded_at DESC`, deviceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*MonitoringMetric
	for rows.Next() {
		m := &MonitoringMetric{}
		if err := rows.Scan(&m.ID, &m.DeviceID, &m.MetricName, &m.MetricValue, &m.Labels, &m.RecordedAt, &m.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, m)
	}
	return list, rows.Err()
}
