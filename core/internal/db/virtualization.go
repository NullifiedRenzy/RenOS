package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type VirtualMachine struct {
	ID                string    `json:"id"`
	DeviceID          string    `json:"device_id"`
	DomainName        string    `json:"domain_name"`
	UUID              string    `json:"uuid"`
	State             string    `json:"state"`
	VCPUMax           int32     `json:"vcpu_max"`
	VCPUCurrent       int32     `json:"vcpu_current"`
	MemoryMaxBytes    int64     `json:"memory_max_bytes"`
	MemoryCurrentBytes int64    `json:"memory_current_bytes"`
	OSType            string    `json:"os_type"`
	Autostart         bool      `json:"autostart"`
	Persistent        bool      `json:"persistent"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type VirtualMachineRepository struct {
	pool *pgxpool.Pool
}

func NewVirtualMachineRepository(pool *pgxpool.Pool) *VirtualMachineRepository {
	return &VirtualMachineRepository{pool: pool}
}

func (r *VirtualMachineRepository) ReplaceForDevice(ctx context.Context, deviceID string, vms []*VirtualMachine) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `DELETE FROM virtual_machines WHERE device_id = $1`, deviceID); err != nil {
		return fmt.Errorf("delete virtual_machines: %w", err)
	}

	for _, vm := range vms {
		if _, err := tx.Exec(ctx,
			`INSERT INTO virtual_machines (device_id, domain_name, uuid, state, vcpu_max, vcpu_current, memory_max_bytes, memory_current_bytes, os_type, autostart, persistent)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
			deviceID, vm.DomainName, vm.UUID, vm.State, vm.VCPUMax, vm.VCPUCurrent,
			vm.MemoryMaxBytes, vm.MemoryCurrentBytes, vm.OSType, vm.Autostart, vm.Persistent,
		); err != nil {
			return fmt.Errorf("insert virtual_machine: %w", err)
		}
	}

	return tx.Commit(ctx)
}

func (r *VirtualMachineRepository) ListByDevice(ctx context.Context, deviceID string) ([]*VirtualMachine, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, device_id, domain_name, uuid, state, vcpu_max, vcpu_current,
		        memory_max_bytes, memory_current_bytes, os_type, autostart, persistent,
		        created_at, updated_at
		 FROM virtual_machines WHERE device_id = $1 ORDER BY domain_name ASC`,
		deviceID,
	)
	if err != nil {
		return nil, fmt.Errorf("list virtual_machines: %w", err)
	}
	defer rows.Close()

	var vms []*VirtualMachine
	for rows.Next() {
		vm, err := scanVirtualMachine(rows)
		if err != nil {
			return nil, err
		}
		vms = append(vms, vm)
	}
	return vms, rows.Err()
}

func scanVirtualMachine(row scanner) (*VirtualMachine, error) {
	vm := &VirtualMachine{}
	err := row.Scan(&vm.ID, &vm.DeviceID, &vm.DomainName, &vm.UUID, &vm.State,
		&vm.VCPUMax, &vm.VCPUCurrent, &vm.MemoryMaxBytes, &vm.MemoryCurrentBytes,
		&vm.OSType, &vm.Autostart, &vm.Persistent,
		&vm.CreatedAt, &vm.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("scan virtual_machine: %w", err)
	}
	return vm, nil
}
