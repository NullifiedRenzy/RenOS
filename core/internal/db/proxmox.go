package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ProxmoxConnection represents a Proxmox VE API connection
type ProxmoxConnection struct {
	ID              string     `json:"id"`
	DeviceID        string     `json:"device_id"`
	Name            string     `json:"name"`
	Host            string     `json:"host"`
	Port            int        `json:"port"`
	AuthType        string     `json:"auth_type"`
	TokenID         string     `json:"token_id"`
	TokenSecret     string     `json:"token_secret"`
	Username        string     `json:"username"`
	Password        string     `json:"password"`
	VerifySSL       bool       `json:"verify_ssl"`
	Status          string     `json:"status"`
	ErrorMessage    string     `json:"error_message"`
	LastConnectedAt *time.Time `json:"last_connected_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type ProxmoxNode struct {
	ID             string    `json:"id"`
	ConnectionID   string    `json:"connection_id"`
	NodeName       string    `json:"node_name"`
	Status         string    `json:"status"`
	CPU            float64   `json:"cpu"`
	MaxCPU         int       `json:"max_cpu"`
	Memory         int64     `json:"memory"`
	MaxMemory      int64     `json:"max_memory"`
	Disk           int64     `json:"disk"`
	MaxDisk        int64     `json:"max_disk"`
	Uptime         int64     `json:"uptime"`
	SSLFingerprint string    `json:"ssl_fingerprint"`
	Version        string    `json:"version"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type ProxmoxVM struct {
	ID           string    `json:"id"`
	ConnectionID string    `json:"connection_id"`
	NodeName     string    `json:"node_name"`
	VMID         int       `json:"vm_id"`
	Name         string    `json:"name"`
	Status       string    `json:"status"`
	CPU          int       `json:"cpu"`
	Cores        int       `json:"cores"`
	Memory       int64     `json:"memory"`
	Disk         int64     `json:"disk"`
	IPAddress    string    `json:"ip_address"`
	OSType       string    `json:"os_type"`
	Template     bool      `json:"template"`
	HAState      string    `json:"ha_state"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type ProxmoxLXC struct {
	ID           string    `json:"id"`
	ConnectionID string    `json:"connection_id"`
	NodeName     string    `json:"node_name"`
	VMID         int       `json:"vm_id"`
	Name         string    `json:"name"`
	Status       string    `json:"status"`
	CPU          int       `json:"cpu"`
	Cores        int       `json:"cores"`
	Memory       int64     `json:"memory"`
	Disk         int64     `json:"disk"`
	IPAddress    string    `json:"ip_address"`
	OSType       string    `json:"os_type"`
	Template     bool      `json:"template"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type ProxmoxStorage struct {
	ID           string    `json:"id"`
	ConnectionID string    `json:"connection_id"`
	NodeName     string    `json:"node_name"`
	StorageID    string    `json:"storage_id"`
	StorageType  string    `json:"storage_type"`
	Status       string    `json:"status"`
	Total        int64     `json:"total"`
	Used         int64     `json:"used"`
	Content      string    `json:"content"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type ProxmoxNetwork struct {
	ID           string    `json:"id"`
	ConnectionID string    `json:"connection_id"`
	NodeName     string    `json:"node_name"`
	Interface    string    `json:"interface"`
	IfaceType    string    `json:"iface_type"`
	Address      string    `json:"address"`
	Netmask      string    `json:"netmask"`
	Gateway      string    `json:"gateway"`
	Bridge       string    `json:"bridge"`
	VLAN         *int      `json:"vlan,omitempty"`
	Active       bool      `json:"active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type ProxmoxSnapshot struct {
	ID           string    `json:"id"`
	ConnectionID string    `json:"connection_id"`
	NodeName     string    `json:"node_name"`
	VMID         int       `json:"vm_id"`
	SnapshotID   string    `json:"snapshot_id"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	Parent       string    `json:"parent"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type ProxmoxTask struct {
	ID           string    `json:"id"`
	ConnectionID string    `json:"connection_id"`
	NodeName     string    `json:"node_name"`
	UPID         string    `json:"upid"`
	TaskType     string    `json:"task_type"`
	Status       string    `json:"status"`
	UserName     string    `json:"user_name"`
	StartTime    int64     `json:"start_time"`
	EndTime      int64     `json:"end_time"`
	ExitStatus   string    `json:"exit_status"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// ProxmoxRepository manages all Proxmox data
type ProxmoxRepository struct {
	pool *pgxpool.Pool
}

func NewProxmoxRepository(pool *pgxpool.Pool) *ProxmoxRepository {
	return &ProxmoxRepository{pool: pool}
}

// --- Connections ---

func (r *ProxmoxRepository) CreateConnection(ctx context.Context, deviceID, name, host string, port int, authType, tokenID, tokenSecret, username, password string, verifySSL bool) (*ProxmoxConnection, error) {
	row := r.pool.QueryRow(ctx,
		`INSERT INTO proxmox_connections (device_id, name, host, port, auth_type, token_id, token_secret, username, password, verify_ssl)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		 RETURNING id, device_id, name, host, port, auth_type, token_id, token_secret, username, password, verify_ssl, status, error_message, last_connected_at, created_at, updated_at`,
		deviceID, name, host, port, authType, tokenID, tokenSecret, username, password, verifySSL,
	)
	return scanProxmoxConnection(row)
}

func (r *ProxmoxRepository) UpdateConnectionStatus(ctx context.Context, id, status, errMsg string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE proxmox_connections SET status = $1, error_message = $2, last_connected_at = CASE WHEN $1 = 'connected' THEN NOW() ELSE last_connected_at END, updated_at = NOW() WHERE id = $3`,
		status, errMsg, id,
	)
	return err
}

func (r *ProxmoxRepository) ListConnectionsByDevice(ctx context.Context, deviceID string) ([]*ProxmoxConnection, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, device_id, name, host, port, auth_type, token_id, token_secret, username, password, verify_ssl, status, error_message, last_connected_at, created_at, updated_at
		 FROM proxmox_connections WHERE device_id = $1 ORDER BY name`, deviceID,
	)
	if err != nil {
		return nil, fmt.Errorf("list proxmox connections: %w", err)
	}
	defer rows.Close()
	var list []*ProxmoxConnection
	for rows.Next() {
		c, err := scanProxmoxConnection(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, c)
	}
	return list, rows.Err()
}

func (r *ProxmoxRepository) DeleteConnection(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM proxmox_connections WHERE id = $1`, id)
	return err
}

// --- Nodes ---

func (r *ProxmoxRepository) ReplaceNodes(ctx context.Context, connectionID string, nodes []*ProxmoxNode) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `DELETE FROM proxmox_nodes WHERE connection_id = $1`, connectionID); err != nil {
		return fmt.Errorf("delete old nodes: %w", err)
	}

	for _, n := range nodes {
		n.ConnectionID = connectionID
		if _, err := tx.Exec(ctx,
			`INSERT INTO proxmox_nodes (connection_id, node_name, status, cpu, max_cpu, memory, max_memory, disk, max_disk, uptime, ssl_fingerprint, version)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
			n.ConnectionID, n.NodeName, n.Status, n.CPU, n.MaxCPU, n.Memory, n.MaxMemory, n.Disk, n.MaxDisk, n.Uptime, n.SSLFingerprint, n.Version,
		); err != nil {
			return fmt.Errorf("insert node: %w", err)
		}
	}
	return tx.Commit(ctx)
}

func (r *ProxmoxRepository) ListNodes(ctx context.Context, connectionID string) ([]*ProxmoxNode, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, connection_id, node_name, status, cpu, max_cpu, memory, max_memory, disk, max_disk, uptime, ssl_fingerprint, version, created_at, updated_at
		 FROM proxmox_nodes WHERE connection_id = $1 ORDER BY node_name`, connectionID,
	)
	if err != nil {
		return nil, fmt.Errorf("list proxmox nodes: %w", err)
	}
	defer rows.Close()
	var list []*ProxmoxNode
	for rows.Next() {
		n := &ProxmoxNode{}
		if err := rows.Scan(&n.ID, &n.ConnectionID, &n.NodeName, &n.Status, &n.CPU, &n.MaxCPU, &n.Memory, &n.MaxMemory, &n.Disk, &n.MaxDisk, &n.Uptime, &n.SSLFingerprint, &n.Version, &n.CreatedAt, &n.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan node: %w", err)
		}
		list = append(list, n)
	}
	return list, rows.Err()
}

// --- VMs ---

func (r *ProxmoxRepository) ReplaceVMs(ctx context.Context, connectionID string, vms []*ProxmoxVM) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `DELETE FROM proxmox_vms WHERE connection_id = $1`, connectionID); err != nil {
		return fmt.Errorf("delete old vms: %w", err)
	}

	for _, v := range vms {
		v.ConnectionID = connectionID
		if _, err := tx.Exec(ctx,
			`INSERT INTO proxmox_vms (connection_id, node_name, vm_id, name, status, cpu, cores, memory, disk, ip_address, os_type, template, ha_state)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
			v.ConnectionID, v.NodeName, v.VMID, v.Name, v.Status, v.CPU, v.Cores, v.Memory, v.Disk, v.IPAddress, v.OSType, v.Template, v.HAState,
		); err != nil {
			return fmt.Errorf("insert vm: %w", err)
		}
	}
	return tx.Commit(ctx)
}

func (r *ProxmoxRepository) ListVMs(ctx context.Context, connectionID string) ([]*ProxmoxVM, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, connection_id, node_name, vm_id, name, status, cpu, cores, memory, disk, ip_address, os_type, template, ha_state, created_at, updated_at
		 FROM proxmox_vms WHERE connection_id = $1 ORDER BY vm_id`, connectionID,
	)
	if err != nil {
		return nil, fmt.Errorf("list proxmox vms: %w", err)
	}
	defer rows.Close()
	var list []*ProxmoxVM
	for rows.Next() {
		v := &ProxmoxVM{}
		if err := rows.Scan(&v.ID, &v.ConnectionID, &v.NodeName, &v.VMID, &v.Name, &v.Status, &v.CPU, &v.Cores, &v.Memory, &v.Disk, &v.IPAddress, &v.OSType, &v.Template, &v.HAState, &v.CreatedAt, &v.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan vm: %w", err)
		}
		list = append(list, v)
	}
	return list, rows.Err()
}

// --- LXCs ---

func (r *ProxmoxRepository) ReplaceLXCs(ctx context.Context, connectionID string, lxcs []*ProxmoxLXC) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `DELETE FROM proxmox_lxcs WHERE connection_id = $1`, connectionID); err != nil {
		return fmt.Errorf("delete old lxcs: %w", err)
	}

	for _, l := range lxcs {
		l.ConnectionID = connectionID
		if _, err := tx.Exec(ctx,
			`INSERT INTO proxmox_lxcs (connection_id, node_name, vm_id, name, status, cpu, cores, memory, disk, ip_address, os_type, template)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
			l.ConnectionID, l.NodeName, l.VMID, l.Name, l.Status, l.CPU, l.Cores, l.Memory, l.Disk, l.IPAddress, l.OSType, l.Template,
		); err != nil {
			return fmt.Errorf("insert lxc: %w", err)
		}
	}
	return tx.Commit(ctx)
}

func (r *ProxmoxRepository) ListLXCs(ctx context.Context, connectionID string) ([]*ProxmoxLXC, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, connection_id, node_name, vm_id, name, status, cpu, cores, memory, disk, ip_address, os_type, template, created_at, updated_at
		 FROM proxmox_lxcs WHERE connection_id = $1 ORDER BY vm_id`, connectionID,
	)
	if err != nil {
		return nil, fmt.Errorf("list proxmox lxcs: %w", err)
	}
	defer rows.Close()
	var list []*ProxmoxLXC
	for rows.Next() {
		l := &ProxmoxLXC{}
		if err := rows.Scan(&l.ID, &l.ConnectionID, &l.NodeName, &l.VMID, &l.Name, &l.Status, &l.CPU, &l.Cores, &l.Memory, &l.Disk, &l.IPAddress, &l.OSType, &l.Template, &l.CreatedAt, &l.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan lxc: %w", err)
		}
		list = append(list, l)
	}
	return list, rows.Err()
}

// --- Storage ---

func (r *ProxmoxRepository) ReplaceStorage(ctx context.Context, connectionID string, items []*ProxmoxStorage) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `DELETE FROM proxmox_storage WHERE connection_id = $1`, connectionID); err != nil {
		return fmt.Errorf("delete old storage: %w", err)
	}

	for _, s := range items {
		s.ConnectionID = connectionID
		if _, err := tx.Exec(ctx,
			`INSERT INTO proxmox_storage (connection_id, node_name, storage_id, storage_type, status, total, used, content)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
			s.ConnectionID, s.NodeName, s.StorageID, s.StorageType, s.Status, s.Total, s.Used, s.Content,
		); err != nil {
			return fmt.Errorf("insert storage: %w", err)
		}
	}
	return tx.Commit(ctx)
}

func (r *ProxmoxRepository) ListStorage(ctx context.Context, connectionID string) ([]*ProxmoxStorage, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, connection_id, node_name, storage_id, storage_type, status, total, used, content, created_at, updated_at
		 FROM proxmox_storage WHERE connection_id = $1 ORDER BY node_name, storage_id`, connectionID,
	)
	if err != nil {
		return nil, fmt.Errorf("list proxmox storage: %w", err)
	}
	defer rows.Close()
	var list []*ProxmoxStorage
	for rows.Next() {
		s := &ProxmoxStorage{}
		if err := rows.Scan(&s.ID, &s.ConnectionID, &s.NodeName, &s.StorageID, &s.StorageType, &s.Status, &s.Total, &s.Used, &s.Content, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan storage: %w", err)
		}
		list = append(list, s)
	}
	return list, rows.Err()
}

// --- Networks ---

func (r *ProxmoxRepository) ReplaceNetworks(ctx context.Context, connectionID string, nets []*ProxmoxNetwork) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `DELETE FROM proxmox_networks WHERE connection_id = $1`, connectionID); err != nil {
		return fmt.Errorf("delete old networks: %w", err)
	}

	for _, n := range nets {
		n.ConnectionID = connectionID
		if _, err := tx.Exec(ctx,
			`INSERT INTO proxmox_networks (connection_id, node_name, interface, iface_type, address, netmask, gateway, bridge, vlan, active)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
			n.ConnectionID, n.NodeName, n.Interface, n.IfaceType, n.Address, n.Netmask, n.Gateway, n.Bridge, n.VLAN, n.Active,
		); err != nil {
			return fmt.Errorf("insert network: %w", err)
		}
	}
	return tx.Commit(ctx)
}

func (r *ProxmoxRepository) ListNetworks(ctx context.Context, connectionID string) ([]*ProxmoxNetwork, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, connection_id, node_name, interface, iface_type, address, netmask, gateway, bridge, vlan, active, created_at, updated_at
		 FROM proxmox_networks WHERE connection_id = $1 ORDER BY node_name, interface`, connectionID,
	)
	if err != nil {
		return nil, fmt.Errorf("list proxmox networks: %w", err)
	}
	defer rows.Close()
	var list []*ProxmoxNetwork
	for rows.Next() {
		n := &ProxmoxNetwork{}
		if err := rows.Scan(&n.ID, &n.ConnectionID, &n.NodeName, &n.Interface, &n.IfaceType, &n.Address, &n.Netmask, &n.Gateway, &n.Bridge, &n.VLAN, &n.Active, &n.CreatedAt, &n.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan network: %w", err)
		}
		list = append(list, n)
	}
	return list, rows.Err()
}

// --- Snapshots ---

func (r *ProxmoxRepository) ReplaceSnapshots(ctx context.Context, connectionID string, snaps []*ProxmoxSnapshot) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `DELETE FROM proxmox_snapshots WHERE connection_id = $1`, connectionID); err != nil {
		return fmt.Errorf("delete old snapshots: %w", err)
	}

	for _, s := range snaps {
		s.ConnectionID = connectionID
		if _, err := tx.Exec(ctx,
			`INSERT INTO proxmox_snapshots (connection_id, node_name, vm_id, snapshot_id, name, description, parent)
			 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			s.ConnectionID, s.NodeName, s.VMID, s.SnapshotID, s.Name, s.Description, s.Parent,
		); err != nil {
			return fmt.Errorf("insert snapshot: %w", err)
		}
	}
	return tx.Commit(ctx)
}

func (r *ProxmoxRepository) ListSnapshots(ctx context.Context, connectionID string) ([]*ProxmoxSnapshot, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, connection_id, node_name, vm_id, snapshot_id, name, description, parent, created_at, updated_at
		 FROM proxmox_snapshots WHERE connection_id = $1 ORDER BY node_name, vm_id, snapshot_id`, connectionID,
	)
	if err != nil {
		return nil, fmt.Errorf("list proxmox snapshots: %w", err)
	}
	defer rows.Close()
	var list []*ProxmoxSnapshot
	for rows.Next() {
		s := &ProxmoxSnapshot{}
		if err := rows.Scan(&s.ID, &s.ConnectionID, &s.NodeName, &s.VMID, &s.SnapshotID, &s.Name, &s.Description, &s.Parent, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan snapshot: %w", err)
		}
		list = append(list, s)
	}
	return list, rows.Err()
}

// --- Tasks ---

func (r *ProxmoxRepository) ReplaceTasks(ctx context.Context, connectionID string, tasks []*ProxmoxTask) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `DELETE FROM proxmox_tasks WHERE connection_id = $1`, connectionID); err != nil {
		return fmt.Errorf("delete old tasks: %w", err)
	}

	for _, t := range tasks {
		t.ConnectionID = connectionID
		if _, err := tx.Exec(ctx,
			`INSERT INTO proxmox_tasks (connection_id, node_name, upid, task_type, status, user_name, start_time, end_time, exit_status)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
			t.ConnectionID, t.NodeName, t.UPID, t.TaskType, t.Status, t.UserName, t.StartTime, t.EndTime, t.ExitStatus,
		); err != nil {
			return fmt.Errorf("insert task: %w", err)
		}
	}
	return tx.Commit(ctx)
}

func (r *ProxmoxRepository) ListTasks(ctx context.Context, connectionID string, limit int) ([]*ProxmoxTask, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id, connection_id, node_name, upid, task_type, status, user_name, start_time, end_time, exit_status, created_at, updated_at
		 FROM proxmox_tasks WHERE connection_id = $1 ORDER BY start_time DESC LIMIT $2`, connectionID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list proxmox tasks: %w", err)
	}
	defer rows.Close()
	var list []*ProxmoxTask
	for rows.Next() {
		t := &ProxmoxTask{}
		if err := rows.Scan(&t.ID, &t.ConnectionID, &t.NodeName, &t.UPID, &t.TaskType, &t.Status, &t.UserName, &t.StartTime, &t.EndTime, &t.ExitStatus, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan task: %w", err)
		}
		list = append(list, t)
	}
	return list, rows.Err()
}

// --- Helpers ---

func scanProxmoxConnection(row pgx.Row) (*ProxmoxConnection, error) {
	c := &ProxmoxConnection{}
	err := row.Scan(&c.ID, &c.DeviceID, &c.Name, &c.Host, &c.Port, &c.AuthType, &c.TokenID, &c.TokenSecret, &c.Username, &c.Password, &c.VerifySSL, &c.Status, &c.ErrorMessage, &c.LastConnectedAt, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("scan proxmox connection: %w", err)
	}
	return c, nil
}
