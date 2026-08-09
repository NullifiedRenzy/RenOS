package db

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// --- Installed Plugin ---

type InstalledPlugin struct {
	ID           string          `json:"id"`
	Name         string          `json:"name"`
	DisplayName  string          `json:"display_name"`
	Version      string          `json:"version"`
	Author       string          `json:"author"`
	Description  string          `json:"description"`
	PluginType   string          `json:"plugin_type"`
	Status       string          `json:"status"`
	ErrorMessage string          `json:"error_message"`
	Permissions  json.RawMessage `json:"permissions"`
	Config       json.RawMessage `json:"config"`
	ConfigSchema json.RawMessage `json:"config_schema"`
	Dependencies json.RawMessage `json:"dependencies"`
	EntryPoint   string          `json:"entry_point"`
	BlobPath     string          `json:"blob_path"`
	BlobHash     string          `json:"blob_hash"`
	Signature    string          `json:"signature"`
	EnabledAt    *time.Time      `json:"enabled_at,omitempty"`
	DisabledAt   *time.Time      `json:"disabled_at,omitempty"`
	LastErrorAt  *time.Time      `json:"last_error_at,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

type PluginRoute struct {
	ID          string    `json:"id"`
	PluginID    string    `json:"plugin_id"`
	Method      string    `json:"method"`
	Path        string    `json:"path"`
	HandlerName string    `json:"handler_name"`
	Description string    `json:"description"`
	RequireAuth bool      `json:"require_auth"`
	MinRole     string    `json:"min_role"`
	RateLimit   int       `json:"rate_limit"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type PluginPage struct {
	ID            string    `json:"id"`
	PluginID      string    `json:"plugin_id"`
	PagePath      string    `json:"page_path"`
	Title         string    `json:"title"`
	Description   string    `json:"description"`
	Icon          string    `json:"icon"`
	ComponentName string    `json:"component_name"`
	ParentPath    string    `json:"parent_path"`
	SortOrder     int       `json:"sort_order"`
	RequireAuth   bool      `json:"require_auth"`
	MinRole       string    `json:"min_role"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type PluginSidebarEntry struct {
	ID          string    `json:"id"`
	PluginID    string    `json:"plugin_id"`
	Label       string    `json:"label"`
	Icon        string    `json:"icon"`
	Path        string    `json:"path"`
	ParentLabel string    `json:"parent_label"`
	SortOrder   int       `json:"sort_order"`
	MinRole     string    `json:"min_role"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type PluginWorker struct {
	ID           string    `json:"id"`
	PluginID     string    `json:"plugin_id"`
	WorkerName   string    `json:"worker_name"`
	Description  string    `json:"description"`
	HandlerName  string    `json:"handler_name"`
	Concurrency  int       `json:"concurrency"`
	Status       string    `json:"status"`
	LastError    string    `json:"last_error"`
	RestartCount int       `json:"restart_count"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type PluginSchedule struct {
	ID              string     `json:"id"`
	PluginID        string     `json:"plugin_id"`
	ScheduleName    string     `json:"schedule_name"`
	Description     string     `json:"description"`
	HandlerName     string     `json:"handler_name"`
	CronExpr        string     `json:"cron_expr"`
	IntervalSeconds int        `json:"interval_seconds"`
	Enabled         bool       `json:"enabled"`
	LastRunAt       *time.Time `json:"last_run_at,omitempty"`
	NextRunAt       *time.Time `json:"next_run_at,omitempty"`
	LastStatus      string     `json:"last_status"`
	LastError       string     `json:"last_error"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// --- Plugin Repository ---

type InstalledPluginRepository struct {
	pool *pgxpool.Pool
}

func NewInstalledPluginRepository(pool *pgxpool.Pool) *InstalledPluginRepository {
	return &InstalledPluginRepository{pool: pool}
}

func (r *InstalledPluginRepository) Create(ctx context.Context, p *InstalledPlugin) error {
	row := r.pool.QueryRow(ctx,
		`INSERT INTO installed_plugins (name, display_name, version, author, description, plugin_type, status, permissions, config, config_schema, dependencies, entry_point, blob_path, blob_hash, signature)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)
		 RETURNING id, created_at, updated_at`,
		p.Name, p.DisplayName, p.Version, p.Author, p.Description, p.PluginType, p.Status,
		p.Permissions, p.Config, p.ConfigSchema, p.Dependencies,
		p.EntryPoint, p.BlobPath, p.BlobHash, p.Signature,
	)
	return row.Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt)
}

func (r *InstalledPluginRepository) FindByID(ctx context.Context, id string) (*InstalledPlugin, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, name, display_name, version, author, description, plugin_type, status, error_message,
		        permissions, config, config_schema, dependencies, entry_point, blob_path, blob_hash, signature,
		        enabled_at, disabled_at, last_error_at, created_at, updated_at
		 FROM installed_plugins WHERE id = $1`, id,
	)
	return scanInstalledPlugin(row)
}

func (r *InstalledPluginRepository) FindByName(ctx context.Context, name string) (*InstalledPlugin, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, name, display_name, version, author, description, plugin_type, status, error_message,
		        permissions, config, config_schema, dependencies, entry_point, blob_path, blob_hash, signature,
		        enabled_at, disabled_at, last_error_at, created_at, updated_at
		 FROM installed_plugins WHERE name = $1`, name,
	)
	return scanInstalledPlugin(row)
}

func (r *InstalledPluginRepository) ListAll(ctx context.Context) ([]*InstalledPlugin, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, name, display_name, version, author, description, plugin_type, status, error_message,
		        permissions, config, config_schema, dependencies, entry_point, blob_path, blob_hash, signature,
		        enabled_at, disabled_at, last_error_at, created_at, updated_at
		 FROM installed_plugins WHERE status != 'uninstalled' ORDER BY name`,
	)
	if err != nil {
		return nil, fmt.Errorf("list installed plugins: %w", err)
	}
	defer rows.Close()
	var list []*InstalledPlugin
	for rows.Next() {
		p, err := scanInstalledPlugin(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, p)
	}
	return list, rows.Err()
}

func (r *InstalledPluginRepository) ListEnabled(ctx context.Context) ([]*InstalledPlugin, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, name, display_name, version, author, description, plugin_type, status, error_message,
		        permissions, config, config_schema, dependencies, entry_point, blob_path, blob_hash, signature,
		        enabled_at, disabled_at, last_error_at, created_at, updated_at
		 FROM installed_plugins WHERE status = 'enabled' ORDER BY name`,
	)
	if err != nil {
		return nil, fmt.Errorf("list enabled plugins: %w", err)
	}
	defer rows.Close()
	var list []*InstalledPlugin
	for rows.Next() {
		p, err := scanInstalledPlugin(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, p)
	}
	return list, rows.Err()
}

func (r *InstalledPluginRepository) UpdateStatus(ctx context.Context, id, status string) error {
	now := time.Now()
	var err error
	switch status {
	case "enabled":
		_, err = r.pool.Exec(ctx, `UPDATE installed_plugins SET status = $1, enabled_at = $2, disabled_at = NULL, updated_at = NOW() WHERE id = $3`, status, now, id)
	case "disabled":
		_, err = r.pool.Exec(ctx, `UPDATE installed_plugins SET status = $1, disabled_at = $2, updated_at = NOW() WHERE id = $3`, status, now, id)
	case "error":
		_, err = r.pool.Exec(ctx, `UPDATE installed_plugins SET status = $1, last_error_at = $2, updated_at = NOW() WHERE id = $3`, status, now, id)
	default:
		_, err = r.pool.Exec(ctx, `UPDATE installed_plugins SET status = $1, updated_at = NOW() WHERE id = $2`, status, id)
	}
	return err
}

func (r *InstalledPluginRepository) UpdateConfig(ctx context.Context, id string, config json.RawMessage) error {
	_, err := r.pool.Exec(ctx, `UPDATE installed_plugins SET config = $1, updated_at = NOW() WHERE id = $2`, config, id)
	return err
}

func (r *InstalledPluginRepository) Delete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `UPDATE installed_plugins SET status = 'uninstalled', updated_at = NOW() WHERE id = $1`, id)
	return err
}

// --- Routes ---

func (r *InstalledPluginRepository) ReplaceRoutes(ctx context.Context, pluginID string, routes []*PluginRoute) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `DELETE FROM plugin_routes WHERE plugin_id = $1`, pluginID); err != nil {
		return err
	}
	for _, rt := range routes {
		rt.PluginID = pluginID
		if _, err := tx.Exec(ctx,
			`INSERT INTO plugin_routes (plugin_id, method, path, handler_name, description, require_auth, min_role, rate_limit)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
			rt.PluginID, rt.Method, rt.Path, rt.HandlerName, rt.Description, rt.RequireAuth, rt.MinRole, rt.RateLimit,
		); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *InstalledPluginRepository) ListRoutes(ctx context.Context, pluginID string) ([]*PluginRoute, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, plugin_id, method, path, handler_name, description, require_auth, min_role, rate_limit, created_at, updated_at
		 FROM plugin_routes WHERE plugin_id = $1 ORDER BY path`, pluginID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*PluginRoute
	for rows.Next() {
		rt := &PluginRoute{}
		if err := rows.Scan(&rt.ID, &rt.PluginID, &rt.Method, &rt.Path, &rt.HandlerName, &rt.Description, &rt.RequireAuth, &rt.MinRole, &rt.RateLimit, &rt.CreatedAt, &rt.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, rt)
	}
	return list, rows.Err()
}

func (r *InstalledPluginRepository) ListAllRoutes(ctx context.Context) ([]*PluginRoute, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT r.id, r.plugin_id, r.method, r.path, r.handler_name, r.description, r.require_auth, r.min_role, r.rate_limit, r.created_at, r.updated_at
		 FROM plugin_routes r JOIN installed_plugins p ON r.plugin_id = p.id WHERE p.status = 'enabled' ORDER BY r.path`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*PluginRoute
	for rows.Next() {
		rt := &PluginRoute{}
		if err := rows.Scan(&rt.ID, &rt.PluginID, &rt.Method, &rt.Path, &rt.HandlerName, &rt.Description, &rt.RequireAuth, &rt.MinRole, &rt.RateLimit, &rt.CreatedAt, &rt.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, rt)
	}
	return list, rows.Err()
}

// --- Pages ---

func (r *InstalledPluginRepository) ReplacePages(ctx context.Context, pluginID string, pages []*PluginPage) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `DELETE FROM plugin_pages WHERE plugin_id = $1`, pluginID); err != nil {
		return err
	}
	for _, pg := range pages {
		pg.PluginID = pluginID
		if _, err := tx.Exec(ctx,
			`INSERT INTO plugin_pages (plugin_id, page_path, title, description, icon, component_name, parent_path, sort_order, require_auth, min_role)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
			pg.PluginID, pg.PagePath, pg.Title, pg.Description, pg.Icon, pg.ComponentName, pg.ParentPath, pg.SortOrder, pg.RequireAuth, pg.MinRole,
		); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *InstalledPluginRepository) ListPages(ctx context.Context, pluginID string) ([]*PluginPage, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, plugin_id, page_path, title, description, icon, component_name, parent_path, sort_order, require_auth, min_role, created_at, updated_at
		 FROM plugin_pages WHERE plugin_id = $1 ORDER BY sort_order, page_path`, pluginID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*PluginPage
	for rows.Next() {
		pg := &PluginPage{}
		if err := rows.Scan(&pg.ID, &pg.PluginID, &pg.PagePath, &pg.Title, &pg.Description, &pg.Icon, &pg.ComponentName, &pg.ParentPath, &pg.SortOrder, &pg.RequireAuth, &pg.MinRole, &pg.CreatedAt, &pg.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, pg)
	}
	return list, rows.Err()
}

func (r *InstalledPluginRepository) ListAllPages(ctx context.Context) ([]*PluginPage, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT pg.id, pg.plugin_id, pg.page_path, pg.title, pg.description, pg.icon, pg.component_name, pg.parent_path, pg.sort_order, pg.require_auth, pg.min_role, pg.created_at, pg.updated_at
		 FROM plugin_pages pg JOIN installed_plugins p ON pg.plugin_id = p.id WHERE p.status = 'enabled' ORDER BY pg.sort_order`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*PluginPage
	for rows.Next() {
		pg := &PluginPage{}
		if err := rows.Scan(&pg.ID, &pg.PluginID, &pg.PagePath, &pg.Title, &pg.Description, &pg.Icon, &pg.ComponentName, &pg.ParentPath, &pg.SortOrder, &pg.RequireAuth, &pg.MinRole, &pg.CreatedAt, &pg.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, pg)
	}
	return list, rows.Err()
}

// --- Sidebar ---

func (r *InstalledPluginRepository) ReplaceSidebar(ctx context.Context, pluginID string, entries []*PluginSidebarEntry) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `DELETE FROM plugin_sidebar WHERE plugin_id = $1`, pluginID); err != nil {
		return err
	}
	for _, e := range entries {
		e.PluginID = pluginID
		if _, err := tx.Exec(ctx,
			`INSERT INTO plugin_sidebar (plugin_id, label, icon, path, parent_label, sort_order, min_role)
			 VALUES ($1,$2,$3,$4,$5,$6,$7)`,
			e.PluginID, e.Label, e.Icon, e.Path, e.ParentLabel, e.SortOrder, e.MinRole,
		); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *InstalledPluginRepository) ListSidebar(ctx context.Context) ([]*PluginSidebarEntry, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT s.id, s.plugin_id, s.label, s.icon, s.path, s.parent_label, s.sort_order, s.min_role, s.created_at, s.updated_at
		 FROM plugin_sidebar s JOIN installed_plugins p ON s.plugin_id = p.id WHERE p.status = 'enabled' ORDER BY s.sort_order`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*PluginSidebarEntry
	for rows.Next() {
		e := &PluginSidebarEntry{}
		if err := rows.Scan(&e.ID, &e.PluginID, &e.Label, &e.Icon, &e.Path, &e.ParentLabel, &e.SortOrder, &e.MinRole, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, e)
	}
	return list, rows.Err()
}

// --- Workers ---

func (r *InstalledPluginRepository) ReplaceWorkers(ctx context.Context, pluginID string, workers []*PluginWorker) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `DELETE FROM plugin_workers WHERE plugin_id = $1`, pluginID); err != nil {
		return err
	}
	for _, w := range workers {
		w.PluginID = pluginID
		if _, err := tx.Exec(ctx,
			`INSERT INTO plugin_workers (plugin_id, worker_name, description, handler_name, concurrency, status)
			 VALUES ($1,$2,$3,$4,$5,$6)`,
			w.PluginID, w.WorkerName, w.Description, w.HandlerName, w.Concurrency, w.Status,
		); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *InstalledPluginRepository) ListWorkers(ctx context.Context) ([]*PluginWorker, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT w.id, w.plugin_id, w.worker_name, w.description, w.handler_name, w.concurrency, w.status, w.last_error, w.restart_count, w.created_at, w.updated_at
		 FROM plugin_workers w JOIN installed_plugins p ON w.plugin_id = p.id WHERE p.status = 'enabled' ORDER BY w.worker_name`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*PluginWorker
	for rows.Next() {
		w := &PluginWorker{}
		if err := rows.Scan(&w.ID, &w.PluginID, &w.WorkerName, &w.Description, &w.HandlerName, &w.Concurrency, &w.Status, &w.LastError, &w.RestartCount, &w.CreatedAt, &w.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, w)
	}
	return list, rows.Err()
}

func (r *InstalledPluginRepository) UpdateWorkerStatus(ctx context.Context, id, status, lastError string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE plugin_workers SET status = $1, last_error = $2, restart_count = CASE WHEN $1 = 'error' THEN restart_count + 1 ELSE restart_count END, updated_at = NOW() WHERE id = $3`,
		status, lastError, id,
	)
	return err
}

// --- Schedules ---

func (r *InstalledPluginRepository) ReplaceSchedules(ctx context.Context, pluginID string, schedules []*PluginSchedule) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `DELETE FROM plugin_schedules WHERE plugin_id = $1`, pluginID); err != nil {
		return err
	}
	for _, s := range schedules {
		s.PluginID = pluginID
		if _, err := tx.Exec(ctx,
			`INSERT INTO plugin_schedules (plugin_id, schedule_name, description, handler_name, cron_expr, interval_seconds, enabled)
			 VALUES ($1,$2,$3,$4,$5,$6,$7)`,
			s.PluginID, s.ScheduleName, s.Description, s.HandlerName, s.CronExpr, s.IntervalSeconds, s.Enabled,
		); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *InstalledPluginRepository) ListSchedules(ctx context.Context) ([]*PluginSchedule, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT s.id, s.plugin_id, s.schedule_name, s.description, s.handler_name, s.cron_expr, s.interval_seconds, s.enabled, s.last_run_at, s.next_run_at, s.last_status, s.last_error, s.created_at, s.updated_at
		 FROM plugin_schedules s JOIN installed_plugins p ON s.plugin_id = p.id WHERE p.status = 'enabled' AND s.enabled = true ORDER BY s.schedule_name`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*PluginSchedule
	for rows.Next() {
		s := &PluginSchedule{}
		if err := rows.Scan(&s.ID, &s.PluginID, &s.ScheduleName, &s.Description, &s.HandlerName, &s.CronExpr, &s.IntervalSeconds, &s.Enabled, &s.LastRunAt, &s.NextRunAt, &s.LastStatus, &s.LastError, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, s)
	}
	return list, rows.Err()
}

// --- Helpers ---

func scanInstalledPlugin(row pgx.Row) (*InstalledPlugin, error) {
	p := &InstalledPlugin{}
	err := row.Scan(
		&p.ID, &p.Name, &p.DisplayName, &p.Version, &p.Author, &p.Description, &p.PluginType, &p.Status, &p.ErrorMessage,
		&p.Permissions, &p.Config, &p.ConfigSchema, &p.Dependencies,
		&p.EntryPoint, &p.BlobPath, &p.BlobHash, &p.Signature,
		&p.EnabledAt, &p.DisabledAt, &p.LastErrorAt, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan installed plugin: %w", err)
	}
	return p, nil
}
