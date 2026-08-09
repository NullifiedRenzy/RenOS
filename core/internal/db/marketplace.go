package db

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type MarketplacePlugin struct {
	ID             string          `json:"id"`
	Name           string          `json:"name"`
	DisplayName    string          `json:"display_name"`
	Description    string          `json:"description"`
	Author         string          `json:"author"`
	Category       string          `json:"category"`
	Version        string          `json:"version"`
	MinRenOSVersion string         `json:"min_renos_version"`
	DownloadURL    string          `json:"download_url"`
	Homepage       string          `json:"homepage"`
	Icon           string          `json:"icon"`
	Screenshots    json.RawMessage `json:"screenshots"`
	Tags           json.RawMessage `json:"tags"`
	Downloads      int             `json:"downloads"`
	Rating         float64         `json:"rating"`
	Verified       bool            `json:"verified"`
	Signature      string          `json:"signature"`
	Checksum       string          `json:"checksum"`
	Manifest       json.RawMessage `json:"manifest"`
	LastSyncedAt   *time.Time      `json:"last_synced_at,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

type MarketplaceInstallation struct {
	ID               string          `json:"id"`
	PluginID         string          `json:"plugin_id"`
	InstalledVersion string          `json:"installed_version"`
	Status           string          `json:"status"`
	Config           json.RawMessage `json:"config"`
	InstalledAt      time.Time       `json:"installed_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
}

type MarketplaceInstallHistory struct {
	ID             string    `json:"id"`
	InstallationID string    `json:"installation_id"`
	Version        string    `json:"version"`
	Action         string    `json:"action"`
	Checksum       string    `json:"checksum"`
	Verified       bool      `json:"verified"`
	Notes          string    `json:"notes"`
	CreatedAt      time.Time `json:"created_at"`
}

type MarketplacePluginRepository struct {
	pool *pgxpool.Pool
}

func NewMarketplacePluginRepository(pool *pgxpool.Pool) *MarketplacePluginRepository {
	return &MarketplacePluginRepository{pool: pool}
}

func (r *MarketplacePluginRepository) Upsert(ctx context.Context, p *MarketplacePlugin) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO marketplace_plugins (name, display_name, description, author, category, version, min_renos_version, download_url, homepage, icon, screenshots, tags, downloads, rating, verified, signature, checksum, manifest, last_synced_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19)
		 ON CONFLICT (name) DO UPDATE SET display_name=$2, description=$3, author=$4, category=$5, version=$6, min_renos_version=$7, download_url=$8, homepage=$9, icon=$10, screenshots=$11, tags=$12, downloads=$13, rating=$14, verified=$15, signature=$16, checksum=$17, manifest=$18, last_synced_at=$19, updated_at=NOW()`,
		p.Name, p.DisplayName, p.Description, p.Author, p.Category, p.Version, p.MinRenOSVersion, p.DownloadURL, p.Homepage, p.Icon, p.Screenshots, p.Tags, p.Downloads, p.Rating, p.Verified, p.Signature, p.Checksum, p.Manifest, p.LastSyncedAt,
	)
	return err
}

func (r *MarketplacePluginRepository) FindByName(ctx context.Context, name string) (*MarketplacePlugin, error) {
	p := &MarketplacePlugin{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, name, display_name, description, author, category, version, min_renos_version, download_url, homepage, icon, screenshots, tags, downloads, rating, verified, signature, checksum, manifest, last_synced_at, created_at, updated_at
		 FROM marketplace_plugins WHERE name = $1`, name,
	).Scan(&p.ID, &p.Name, &p.DisplayName, &p.Description, &p.Author, &p.Category, &p.Version, &p.MinRenOSVersion, &p.DownloadURL, &p.Homepage, &p.Icon, &p.Screenshots, &p.Tags, &p.Downloads, &p.Rating, &p.Verified, &p.Signature, &p.Checksum, &p.Manifest, &p.LastSyncedAt, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (r *MarketplacePluginRepository) ListAll(ctx context.Context) ([]*MarketplacePlugin, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, name, display_name, description, author, category, version, min_renos_version, download_url, homepage, icon, screenshots, tags, downloads, rating, verified, signature, checksum, manifest, last_synced_at, created_at, updated_at
		 FROM marketplace_plugins ORDER BY downloads DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*MarketplacePlugin
	for rows.Next() {
		p := &MarketplacePlugin{}
		if err := rows.Scan(&p.ID, &p.Name, &p.DisplayName, &p.Description, &p.Author, &p.Category, &p.Version, &p.MinRenOSVersion, &p.DownloadURL, &p.Homepage, &p.Icon, &p.Screenshots, &p.Tags, &p.Downloads, &p.Rating, &p.Verified, &p.Signature, &p.Checksum, &p.Manifest, &p.LastSyncedAt, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, p)
	}
	return list, rows.Err()
}

func (r *MarketplacePluginRepository) Search(ctx context.Context, query string) ([]*MarketplacePlugin, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, name, display_name, description, author, category, version, min_renos_version, download_url, homepage, icon, screenshots, tags, downloads, rating, verified, signature, checksum, manifest, last_synced_at, created_at, updated_at
		 FROM marketplace_plugins WHERE name ILIKE '%' || $1 || '%' OR display_name ILIKE '%' || $1 || '%' OR description ILIKE '%' || $1 || '%'
		 ORDER BY downloads DESC`, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*MarketplacePlugin
	for rows.Next() {
		p := &MarketplacePlugin{}
		if err := rows.Scan(&p.ID, &p.Name, &p.DisplayName, &p.Description, &p.Author, &p.Category, &p.Version, &p.MinRenOSVersion, &p.DownloadURL, &p.Homepage, &p.Icon, &p.Screenshots, &p.Tags, &p.Downloads, &p.Rating, &p.Verified, &p.Signature, &p.Checksum, &p.Manifest, &p.LastSyncedAt, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, p)
	}
	return list, rows.Err()
}

func (r *MarketplacePluginRepository) IncrementDownloads(ctx context.Context, name string) error {
	_, err := r.pool.Exec(ctx, `UPDATE marketplace_plugins SET downloads = downloads + 1 WHERE name = $1`, name)
	return err
}

type MarketplaceInstallationRepository struct {
	pool *pgxpool.Pool
}

func NewMarketplaceInstallationRepository(pool *pgxpool.Pool) *MarketplaceInstallationRepository {
	return &MarketplaceInstallationRepository{pool: pool}
}

func (r *MarketplaceInstallationRepository) Create(ctx context.Context, inst *MarketplaceInstallation) error {
	row := r.pool.QueryRow(ctx,
		`INSERT INTO marketplace_installations (plugin_id, installed_version, status, config)
		 VALUES ($1,$2,$3,$4) RETURNING id, installed_at, updated_at`,
		inst.PluginID, inst.InstalledVersion, inst.Status, inst.Config,
	)
	return row.Scan(&inst.ID, &inst.InstalledAt, &inst.UpdatedAt)
}

func (r *MarketplaceInstallationRepository) FindByPlugin(ctx context.Context, pluginID string) (*MarketplaceInstallation, error) {
	inst := &MarketplaceInstallation{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, plugin_id, installed_version, status, config, installed_at, updated_at
		 FROM marketplace_installations WHERE plugin_id = $1`, pluginID,
	).Scan(&inst.ID, &inst.PluginID, &inst.InstalledVersion, &inst.Status, &inst.Config, &inst.InstalledAt, &inst.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return inst, nil
}

func (r *MarketplaceInstallationRepository) ListAll(ctx context.Context) ([]*MarketplaceInstallation, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, plugin_id, installed_version, status, config, installed_at, updated_at
		 FROM marketplace_installations ORDER BY installed_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*MarketplaceInstallation
	for rows.Next() {
		inst := &MarketplaceInstallation{}
		if err := rows.Scan(&inst.ID, &inst.PluginID, &inst.InstalledVersion, &inst.Status, &inst.Config, &inst.InstalledAt, &inst.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, inst)
	}
	return list, rows.Err()
}

func (r *MarketplaceInstallationRepository) UpdateStatus(ctx context.Context, id, status string) error {
	_, err := r.pool.Exec(ctx, `UPDATE marketplace_installations SET status = $1, updated_at = NOW() WHERE id = $2`, status, id)
	return err
}

func (r *MarketplaceInstallationRepository) UpdateVersion(ctx context.Context, id, version string) error {
	_, err := r.pool.Exec(ctx, `UPDATE marketplace_installations SET installed_version = $1, updated_at = NOW() WHERE id = $2`, version, id)
	return err
}

func (r *MarketplaceInstallationRepository) Delete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM marketplace_installations WHERE id = $1`, id)
	return err
}

type MarketplaceInstallHistoryRepository struct {
	pool *pgxpool.Pool
}

func NewMarketplaceInstallHistoryRepository(pool *pgxpool.Pool) *MarketplaceInstallHistoryRepository {
	return &MarketplaceInstallHistoryRepository{pool: pool}
}

func (r *MarketplaceInstallHistoryRepository) Create(ctx context.Context, h *MarketplaceInstallHistory) error {
	row := r.pool.QueryRow(ctx,
		`INSERT INTO marketplace_install_history (installation_id, version, action, checksum, verified, notes)
		 VALUES ($1,$2,$3,$4,$5,$6) RETURNING id, created_at`,
		h.InstallationID, h.Version, h.Action, h.Checksum, h.Verified, h.Notes,
	)
	return row.Scan(&h.ID, &h.CreatedAt)
}

func (r *MarketplaceInstallHistoryRepository) ListByInstallation(ctx context.Context, installationID string) ([]*MarketplaceInstallHistory, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, installation_id, version, action, checksum, verified, notes, created_at
		 FROM marketplace_install_history WHERE installation_id = $1 ORDER BY created_at DESC`, installationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*MarketplaceInstallHistory
	for rows.Next() {
		h := &MarketplaceInstallHistory{}
		if err := rows.Scan(&h.ID, &h.InstallationID, &h.Version, &h.Action, &h.Checksum, &h.Verified, &h.Notes, &h.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, h)
	}
	return list, rows.Err()
}
