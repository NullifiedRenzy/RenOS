package db

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type DeviceTag struct {
	ID        string    `json:"id"`
	DeviceID  string    `json:"device_id"`
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type DeviceInventory struct {
	ID                string          `json:"id"`
	DeviceID          string          `json:"device_id"`
	CPUModel          string          `json:"cpu_model"`
	CPUCores          int             `json:"cpu_cores"`
	CPUThreads        int             `json:"cpu_threads"`
	CPUFreqMHz        float64         `json:"cpu_freq_mhz"`
	GPUModel          string          `json:"gpu_model"`
	GPUMemoryMB       int             `json:"gpu_memory_mb"`
	RAMTotalMB        int             `json:"ram_total_mb"`
	RAMUsedMB         int             `json:"ram_used_mb"`
	StorageTotalGB    float64         `json:"storage_total_gb"`
	StorageUsedGB     float64         `json:"storage_used_gb"`
	StorageType       string          `json:"storage_type"`
	NetworkInterfaces json.RawMessage `json:"network_interfaces"`
	KernelVersion     string          `json:"kernel_version"`
	OSFamily          string          `json:"os_family"`
	OSVersion         string          `json:"os_version"`
	InstalledPackages json.RawMessage `json:"installed_packages"`
	InstalledSoftware json.RawMessage `json:"installed_software"`
	UptimeSeconds     int64           `json:"uptime_seconds"`
	LastCollectedAt   *time.Time      `json:"last_collected_at,omitempty"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
}

type ComplianceCheck struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	CheckType   string          `json:"check_type"`
	Severity    string          `json:"severity"`
	Config      json.RawMessage `json:"config"`
	Enabled     bool            `json:"enabled"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

type ComplianceResult struct {
	ID            string          `json:"id"`
	CheckID       string          `json:"check_id"`
	DeviceID      string          `json:"device_id"`
	Status        string          `json:"status"`
	Message       string          `json:"message"`
	Details       json.RawMessage `json:"details"`
	LastCheckedAt *time.Time      `json:"last_checked_at,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

type BulkOperation struct {
	ID             string          `json:"id"`
	Name           string          `json:"name"`
	Action         string          `json:"action"`
	TargetType     string          `json:"target_type"`
	TargetIDs      json.RawMessage `json:"target_ids"`
	Status         string          `json:"status"`
	TotalCount     int             `json:"total_count"`
	CompletedCount int             `json:"completed_count"`
	FailedCount    int             `json:"failed_count"`
	Config         json.RawMessage `json:"config"`
	ErrorMessage   string          `json:"error_message"`
	CreatedBy      *string         `json:"created_by,omitempty"`
	StartedAt      *time.Time      `json:"started_at,omitempty"`
	CompletedAt    *time.Time      `json:"completed_at,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

// --- Tags ---

type DeviceTagRepository struct {
	pool *pgxpool.Pool
}

func NewDeviceTagRepository(pool *pgxpool.Pool) *DeviceTagRepository {
	return &DeviceTagRepository{pool: pool}
}

func (r *DeviceTagRepository) Set(ctx context.Context, deviceID, key, value string) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO device_tags (device_id, tag_key, tag_value) VALUES ($1, $2, $3)
		 ON CONFLICT (device_id, tag_key) DO UPDATE SET tag_value = $3, updated_at = NOW()`,
		deviceID, key, value,
	)
	return err
}

func (r *DeviceTagRepository) Remove(ctx context.Context, deviceID, key string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM device_tags WHERE device_id = $1 AND tag_key = $2`, deviceID, key)
	return err
}

func (r *DeviceTagRepository) ListByDevice(ctx context.Context, deviceID string) ([]*DeviceTag, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, device_id, tag_key, tag_value, created_at, updated_at FROM device_tags WHERE device_id = $1 ORDER BY tag_key`, deviceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tags []*DeviceTag
	for rows.Next() {
		t := &DeviceTag{}
		if err := rows.Scan(&t.ID, &t.DeviceID, &t.Key, &t.Value, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		tags = append(tags, t)
	}
	return tags, rows.Err()
}

func (r *DeviceTagRepository) FindByTag(ctx context.Context, key, value string) ([]string, error) {
	var deviceIDs []string
	var err error
	if value != "" {
		err = r.pool.QueryRow(ctx,
			`SELECT array_agg(device_id) FROM device_tags WHERE tag_key = $1 AND tag_value = $2`, key, value,
		).Scan(&deviceIDs)
	} else {
		err = r.pool.QueryRow(ctx,
			`SELECT array_agg(device_id) FROM device_tags WHERE tag_key = $1`, key,
		).Scan(&deviceIDs)
	}
	if err != nil {
		return nil, err
	}
	return deviceIDs, nil
}

func (r *DeviceTagRepository) ListAllKeys(ctx context.Context) ([]string, error) {
	rows, err := r.pool.Query(ctx, `SELECT DISTINCT tag_key FROM device_tags ORDER BY tag_key`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var keys []string
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	return keys, rows.Err()
}

func (r *DeviceTagRepository) CountDistinctKeys(ctx context.Context) (int64, error) {
	var count int64
	err := r.pool.QueryRow(ctx, `SELECT COUNT(DISTINCT tag_key) FROM device_tags`).Scan(&count)
	return count, err
}

// --- Inventory ---

type DeviceInventoryRepository struct {
	pool *pgxpool.Pool
}

func NewDeviceInventoryRepository(pool *pgxpool.Pool) *DeviceInventoryRepository {
	return &DeviceInventoryRepository{pool: pool}
}

func (r *DeviceInventoryRepository) Upsert(ctx context.Context, inv *DeviceInventory) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO device_inventory (device_id, cpu_model, cpu_cores, cpu_threads, cpu_freq_mhz, gpu_model, gpu_memory_mb, ram_total_mb, ram_used_mb, storage_total_gb, storage_used_gb, storage_type, network_interfaces, kernel_version, os_family, os_version, installed_packages, installed_software, uptime_seconds, last_collected_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20)
		 ON CONFLICT (device_id) DO UPDATE SET cpu_model=$2, cpu_cores=$3, cpu_threads=$4, cpu_freq_mhz=$5, gpu_model=$6, gpu_memory_mb=$7, ram_total_mb=$8, ram_used_mb=$9, storage_total_gb=$10, storage_used_gb=$11, storage_type=$12, network_interfaces=$13, kernel_version=$14, os_family=$15, os_version=$16, installed_packages=$17, installed_software=$18, uptime_seconds=$19, last_collected_at=$20, updated_at=NOW()`,
		inv.DeviceID, inv.CPUModel, inv.CPUCores, inv.CPUThreads, inv.CPUFreqMHz,
		inv.GPUModel, inv.GPUMemoryMB, inv.RAMTotalMB, inv.RAMUsedMB,
		inv.StorageTotalGB, inv.StorageUsedGB, inv.StorageType,
		inv.NetworkInterfaces, inv.KernelVersion, inv.OSFamily, inv.OSVersion,
		inv.InstalledPackages, inv.InstalledSoftware, inv.UptimeSeconds, inv.LastCollectedAt,
	)
	return err
}

func (r *DeviceInventoryRepository) FindByDevice(ctx context.Context, deviceID string) (*DeviceInventory, error) {
	inv := &DeviceInventory{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, device_id, cpu_model, cpu_cores, cpu_threads, cpu_freq_mhz, gpu_model, gpu_memory_mb, ram_total_mb, ram_used_mb, storage_total_gb, storage_used_gb, storage_type, network_interfaces, kernel_version, os_family, os_version, installed_packages, installed_software, uptime_seconds, last_collected_at, created_at, updated_at
		 FROM device_inventory WHERE device_id = $1`, deviceID,
	).Scan(&inv.ID, &inv.DeviceID, &inv.CPUModel, &inv.CPUCores, &inv.CPUThreads, &inv.CPUFreqMHz, &inv.GPUModel, &inv.GPUMemoryMB, &inv.RAMTotalMB, &inv.RAMUsedMB, &inv.StorageTotalGB, &inv.StorageUsedGB, &inv.StorageType, &inv.NetworkInterfaces, &inv.KernelVersion, &inv.OSFamily, &inv.OSVersion, &inv.InstalledPackages, &inv.InstalledSoftware, &inv.UptimeSeconds, &inv.LastCollectedAt, &inv.CreatedAt, &inv.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return inv, nil
}

func (r *DeviceInventoryRepository) ListAll(ctx context.Context) ([]*DeviceInventory, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, device_id, cpu_model, cpu_cores, cpu_threads, cpu_freq_mhz, gpu_model, gpu_memory_mb, ram_total_mb, ram_used_mb, storage_total_gb, storage_used_gb, storage_type, network_interfaces, kernel_version, os_family, os_version, installed_packages, installed_software, uptime_seconds, last_collected_at, created_at, updated_at
		 FROM device_inventory ORDER BY device_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*DeviceInventory
	for rows.Next() {
		inv := &DeviceInventory{}
		if err := rows.Scan(&inv.ID, &inv.DeviceID, &inv.CPUModel, &inv.CPUCores, &inv.CPUThreads, &inv.CPUFreqMHz, &inv.GPUModel, &inv.GPUMemoryMB, &inv.RAMTotalMB, &inv.RAMUsedMB, &inv.StorageTotalGB, &inv.StorageUsedGB, &inv.StorageType, &inv.NetworkInterfaces, &inv.KernelVersion, &inv.OSFamily, &inv.OSVersion, &inv.InstalledPackages, &inv.InstalledSoftware, &inv.UptimeSeconds, &inv.LastCollectedAt, &inv.CreatedAt, &inv.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, inv)
	}
	return list, rows.Err()
}

func (r *DeviceInventoryRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM device_inventory`).Scan(&count)
	return count, err
}

// --- Compliance ---

type ComplianceRepository struct {
	pool *pgxpool.Pool
}

func NewComplianceRepository(pool *pgxpool.Pool) *ComplianceRepository {
	return &ComplianceRepository{pool: pool}
}

func (r *ComplianceRepository) CreateCheck(ctx context.Context, c *ComplianceCheck) error {
	row := r.pool.QueryRow(ctx,
		`INSERT INTO compliance_checks (name, description, check_type, severity, config, enabled)
		 VALUES ($1,$2,$3,$4,$5,$6) RETURNING id, created_at, updated_at`,
		c.Name, c.Description, c.CheckType, c.Severity, c.Config, c.Enabled,
	)
	return row.Scan(&c.ID, &c.CreatedAt, &c.UpdatedAt)
}

func (r *ComplianceRepository) ListChecks(ctx context.Context) ([]*ComplianceCheck, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, name, description, check_type, severity, config, enabled, created_at, updated_at FROM compliance_checks ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*ComplianceCheck
	for rows.Next() {
		c := &ComplianceCheck{}
		if err := rows.Scan(&c.ID, &c.Name, &c.Description, &c.CheckType, &c.Severity, &c.Config, &c.Enabled, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, c)
	}
	return list, rows.Err()
}

func (r *ComplianceRepository) UpsertResult(ctx context.Context, res *ComplianceResult) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO compliance_results (check_id, device_id, status, message, details, last_checked_at)
		 VALUES ($1,$2,$3,$4,$5,$6)
		 ON CONFLICT (check_id, device_id) DO UPDATE SET status=$3, message=$4, details=$5, last_checked_at=$6, updated_at=NOW()`,
		res.CheckID, res.DeviceID, res.Status, res.Message, res.Details, res.LastCheckedAt,
	)
	return err
}

func (r *ComplianceRepository) ListResults(ctx context.Context) ([]*ComplianceResult, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, check_id, device_id, status, message, details, last_checked_at, created_at, updated_at
		 FROM compliance_results ORDER BY updated_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*ComplianceResult
	for rows.Next() {
		res := &ComplianceResult{}
		if err := rows.Scan(&res.ID, &res.CheckID, &res.DeviceID, &res.Status, &res.Message, &res.Details, &res.LastCheckedAt, &res.CreatedAt, &res.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, res)
	}
	return list, rows.Err()
}

func (r *ComplianceRepository) ListResultsByDevice(ctx context.Context, deviceID string) ([]*ComplianceResult, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, check_id, device_id, status, message, details, last_checked_at, created_at, updated_at
		 FROM compliance_results WHERE device_id = $1 ORDER BY updated_at DESC`, deviceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*ComplianceResult
	for rows.Next() {
		res := &ComplianceResult{}
		if err := rows.Scan(&res.ID, &res.CheckID, &res.DeviceID, &res.Status, &res.Message, &res.Details, &res.LastCheckedAt, &res.CreatedAt, &res.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, res)
	}
	return list, rows.Err()
}

func (r *ComplianceRepository) GetSummary(ctx context.Context) (map[string]int, error) {
	rows, err := r.pool.Query(ctx, `SELECT status, COUNT(*) FROM compliance_results GROUP BY status`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	summary := make(map[string]int)
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		summary[status] = count
	}
	return summary, rows.Err()
}

// --- Bulk Operations ---

type BulkOperationRepository struct {
	pool *pgxpool.Pool
}

func NewBulkOperationRepository(pool *pgxpool.Pool) *BulkOperationRepository {
	return &BulkOperationRepository{pool: pool}
}

func (r *BulkOperationRepository) Create(ctx context.Context, op *BulkOperation) error {
	row := r.pool.QueryRow(ctx,
		`INSERT INTO bulk_operations (name, action, target_type, target_ids, config, created_by)
		 VALUES ($1,$2,$3,$4,$5,$6) RETURNING id, status, total_count, completed_count, failed_count, created_at, updated_at`,
		op.Name, op.Action, op.TargetType, op.TargetIDs, op.Config, op.CreatedBy,
	)
	return row.Scan(&op.ID, &op.Status, &op.TotalCount, &op.CompletedCount, &op.FailedCount, &op.CreatedAt, &op.UpdatedAt)
}

func (r *BulkOperationRepository) FindByID(ctx context.Context, id string) (*BulkOperation, error) {
	op := &BulkOperation{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, name, action, target_type, target_ids, status, total_count, completed_count, failed_count, config, error_message, created_by, started_at, completed_at, created_at, updated_at
		 FROM bulk_operations WHERE id = $1`, id,
	).Scan(&op.ID, &op.Name, &op.Action, &op.TargetType, &op.TargetIDs, &op.Status, &op.TotalCount, &op.CompletedCount, &op.FailedCount, &op.Config, &op.ErrorMessage, &op.CreatedBy, &op.StartedAt, &op.CompletedAt, &op.CreatedAt, &op.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return op, nil
}

func (r *BulkOperationRepository) ListAll(ctx context.Context, limit int) ([]*BulkOperation, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id, name, action, target_type, target_ids, status, total_count, completed_count, failed_count, config, error_message, created_by, started_at, completed_at, created_at, updated_at
		 FROM bulk_operations ORDER BY created_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*BulkOperation
	for rows.Next() {
		op := &BulkOperation{}
		if err := rows.Scan(&op.ID, &op.Name, &op.Action, &op.TargetType, &op.TargetIDs, &op.Status, &op.TotalCount, &op.CompletedCount, &op.FailedCount, &op.Config, &op.ErrorMessage, &op.CreatedBy, &op.StartedAt, &op.CompletedAt, &op.CreatedAt, &op.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, op)
	}
	return list, rows.Err()
}

func (r *BulkOperationRepository) UpdateStatus(ctx context.Context, id, status string) error {
	var err error
	switch status {
	case "running":
		now := time.Now()
		_, err = r.pool.Exec(ctx, `UPDATE bulk_operations SET status = $1, started_at = $2, updated_at = NOW() WHERE id = $3`, status, now, id)
	case "completed", "failed", "cancelled":
		now := time.Now()
		_, err = r.pool.Exec(ctx, `UPDATE bulk_operations SET status = $1, completed_at = $2, updated_at = NOW() WHERE id = $3`, status, now, id)
	default:
		_, err = r.pool.Exec(ctx, `UPDATE bulk_operations SET status = $1, updated_at = NOW() WHERE id = $2`, status, id)
	}
	return err
}

func (r *BulkOperationRepository) IncrementProgress(ctx context.Context, id string, completed, failed bool) error {
	if completed {
		_, err := r.pool.Exec(ctx, `UPDATE bulk_operations SET completed_count = completed_count + 1, updated_at = NOW() WHERE id = $1`, id)
		return err
	}
	if failed {
		_, err := r.pool.Exec(ctx, `UPDATE bulk_operations SET failed_count = failed_count + 1, updated_at = NOW() WHERE id = $1`, id)
		return err
	}
	return nil
}
