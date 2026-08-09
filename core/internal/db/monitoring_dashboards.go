package db

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type MonitoringDashboardRepository struct {
	pool *pgxpool.Pool
}

func NewMonitoringDashboardRepository(pool *pgxpool.Pool) *MonitoringDashboardRepository {
	return &MonitoringDashboardRepository{pool: pool}
}

func (r *MonitoringDashboardRepository) Create(ctx context.Context, d *MonitoringDashboard) error {
	row := r.pool.QueryRow(ctx,
		`INSERT INTO monitoring_dashboards (name, description, layout, is_default, created_by)
		 VALUES ($1,$2,$3,$4,$5) RETURNING id, created_at, updated_at`,
		d.Name, d.Description, d.Layout, d.IsDefault, d.CreatedBy,
	)
	return row.Scan(&d.ID, &d.CreatedAt, &d.UpdatedAt)
}

func (r *MonitoringDashboardRepository) FindByID(ctx context.Context, id string) (*MonitoringDashboard, error) {
	d := &MonitoringDashboard{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, name, description, layout, is_default, created_by, created_at, updated_at
		 FROM monitoring_dashboards WHERE id = $1`, id,
	).Scan(&d.ID, &d.Name, &d.Description, &d.Layout, &d.IsDefault, &d.CreatedBy, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return d, nil
}

func (r *MonitoringDashboardRepository) ListAll(ctx context.Context) ([]*MonitoringDashboard, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, name, description, layout, is_default, created_by, created_at, updated_at
		 FROM monitoring_dashboards ORDER BY is_default DESC, name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*MonitoringDashboard
	for rows.Next() {
		d := &MonitoringDashboard{}
		if err := rows.Scan(&d.ID, &d.Name, &d.Description, &d.Layout, &d.IsDefault, &d.CreatedBy, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, d)
	}
	return list, rows.Err()
}

func (r *MonitoringDashboardRepository) Update(ctx context.Context, d *MonitoringDashboard) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE monitoring_dashboards SET name=$1, description=$2, layout=$3, is_default=$4, updated_at=NOW() WHERE id=$5`,
		d.Name, d.Description, d.Layout, d.IsDefault, d.ID,
	)
	return err
}

func (r *MonitoringDashboardRepository) Delete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM monitoring_dashboards WHERE id = $1`, id)
	return err
}

type MonitoringWidgetRepository struct {
	pool *pgxpool.Pool
}

func NewMonitoringWidgetRepository(pool *pgxpool.Pool) *MonitoringWidgetRepository {
	return &MonitoringWidgetRepository{pool: pool}
}

func (r *MonitoringWidgetRepository) Create(ctx context.Context, w *MonitoringWidget) error {
	row := r.pool.QueryRow(ctx,
		`INSERT INTO monitoring_widgets (dashboard_id, widget_type, title, config, position_x, position_y, width, height)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8) RETURNING id, created_at`,
		w.DashboardID, w.WidgetType, w.Title, w.Config, w.PositionX, w.PositionY, w.Width, w.Height,
	)
	return row.Scan(&w.ID, &w.CreatedAt)
}

func (r *MonitoringWidgetRepository) ListByDashboard(ctx context.Context, dashboardID string) ([]*MonitoringWidget, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, dashboard_id, widget_type, title, config, position_x, position_y, width, height, created_at
		 FROM monitoring_widgets WHERE dashboard_id = $1 ORDER BY position_y, position_x`, dashboardID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*MonitoringWidget
	for rows.Next() {
		w := &MonitoringWidget{}
		if err := rows.Scan(&w.ID, &w.DashboardID, &w.WidgetType, &w.Title, &w.Config, &w.PositionX, &w.PositionY, &w.Width, &w.Height, &w.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, w)
	}
	return list, rows.Err()
}

func (r *MonitoringWidgetRepository) DeleteByID(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM monitoring_widgets WHERE id = $1`, id)
	return err
}

// ReplaceForDashboard atomically replaces every widget on a dashboard with the
// given set (used when a dashboard layout is saved from the builder).
func (r *MonitoringWidgetRepository) ReplaceForDashboard(ctx context.Context, dashboardID string, widgets []*MonitoringWidget) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `DELETE FROM monitoring_widgets WHERE dashboard_id = $1`, dashboardID); err != nil {
		return err
	}
	for _, w := range widgets {
		w.DashboardID = dashboardID
		if _, err := tx.Exec(ctx,
			`INSERT INTO monitoring_widgets (dashboard_id, widget_type, title, config, position_x, position_y, width, height)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
			w.DashboardID, w.WidgetType, w.Title, w.Config, w.PositionX, w.PositionY, w.Width, w.Height,
		); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *MonitoringWidgetRepository) DeleteByDashboard(ctx context.Context, dashboardID string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM monitoring_widgets WHERE dashboard_id = $1`, dashboardID)
	return err
}

type MonitoringLogRepository struct {
	pool *pgxpool.Pool
}

func NewMonitoringLogRepository(pool *pgxpool.Pool) *MonitoringLogRepository {
	return &MonitoringLogRepository{pool: pool}
}

func (r *MonitoringLogRepository) Create(ctx context.Context, l *MonitoringLog) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO monitoring_logs (device_id, source, level, message, fields, recorded_at)
		 VALUES ($1,$2,$3,$4,$5,$6)`,
		l.DeviceID, l.Source, l.Level, l.Message, l.Fields, l.RecordedAt,
	)
	return err
}

func (r *MonitoringLogRepository) ListRecent(ctx context.Context, limit int) ([]*MonitoringLog, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id, device_id, source, level, message, fields, recorded_at, created_at
		 FROM monitoring_logs ORDER BY recorded_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*MonitoringLog
	for rows.Next() {
		l := &MonitoringLog{}
		if err := rows.Scan(&l.ID, &l.DeviceID, &l.Source, &l.Level, &l.Message, &l.Fields, &l.RecordedAt, &l.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, l)
	}
	return list, rows.Err()
}

func (r *MonitoringLogRepository) Search(ctx context.Context, query string, level string, limit int) ([]*MonitoringLog, error) {
	if limit <= 0 {
		limit = 100
	}
	var rows pgx.Rows
	var err error
	if level != "" {
		rows, err = r.pool.Query(ctx,
			`SELECT id, device_id, source, level, message, fields, recorded_at, created_at
			 FROM monitoring_logs WHERE level = $1 AND message ILIKE '%' || $2 || '%'
			 ORDER BY recorded_at DESC LIMIT $3`, level, query, limit)
	} else {
		rows, err = r.pool.Query(ctx,
			`SELECT id, device_id, source, level, message, fields, recorded_at, created_at
			 FROM monitoring_logs WHERE message ILIKE '%' || $1 || '%'
			 ORDER BY recorded_at DESC LIMIT $2`, query, limit)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*MonitoringLog
	for rows.Next() {
		l := &MonitoringLog{}
		if err := rows.Scan(&l.ID, &l.DeviceID, &l.Source, &l.Level, &l.Message, &l.Fields, &l.RecordedAt, &l.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, l)
	}
	return list, rows.Err()
}

func (r *MonitoringLogRepository) GetStats(ctx context.Context) (map[string]int, error) {
	rows, err := r.pool.Query(ctx, `SELECT level, COUNT(*) FROM monitoring_logs GROUP BY level`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	stats := make(map[string]int)
	for rows.Next() {
		var level string
		var count int
		if err := rows.Scan(&level, &count); err != nil {
			return nil, err
		}
		stats[level] = count
	}
	return stats, rows.Err()
}
