package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type NetworkInterface struct {
	ID               string    `json:"id"`
	DeviceID         string    `json:"device_id"`
	Name             string    `json:"name"`
	MacAddress       string    `json:"mac_address"`
	IPv4             []string  `json:"ipv4"`
	IPv6             []string  `json:"ipv6"`
	Status           string    `json:"status"`
	Speed            int64     `json:"speed"`
	InterfaceType    string    `json:"interface_type"`
	IsDefaultGateway bool      `json:"is_default_gateway"`
	DNSServers       []string  `json:"dns_servers"`
	MTU              int32     `json:"mtu"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type NetworkInterfaceRepository struct {
	pool *pgxpool.Pool
}

func NewNetworkInterfaceRepository(pool *pgxpool.Pool) *NetworkInterfaceRepository {
	return &NetworkInterfaceRepository{pool: pool}
}

func (r *NetworkInterfaceRepository) ReplaceForDevice(ctx context.Context, deviceID string, ifaces []*NetworkInterface) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `DELETE FROM network_interfaces WHERE device_id = $1`, deviceID); err != nil {
		return fmt.Errorf("delete network_interfaces: %w", err)
	}

	for _, iface := range ifaces {
		if _, err := tx.Exec(ctx,
			`INSERT INTO network_interfaces (device_id, name, mac_address, ipv4, ipv6, status, speed, interface_type, is_default_gateway, dns_servers, mtu)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
			deviceID, iface.Name, iface.MacAddress, iface.IPv4, iface.IPv6,
			iface.Status, iface.Speed, iface.InterfaceType, iface.IsDefaultGateway,
			iface.DNSServers, iface.MTU,
		); err != nil {
			return fmt.Errorf("insert network_interface: %w", err)
		}
	}

	return tx.Commit(ctx)
}

func (r *NetworkInterfaceRepository) ListByDevice(ctx context.Context, deviceID string) ([]*NetworkInterface, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, device_id, name, mac_address, ipv4, ipv6, status, speed, interface_type,
		        is_default_gateway, dns_servers, mtu, created_at, updated_at
		 FROM network_interfaces WHERE device_id = $1 ORDER BY name ASC`,
		deviceID,
	)
	if err != nil {
		return nil, fmt.Errorf("list network_interfaces: %w", err)
	}
	defer rows.Close()

	var interfaces []*NetworkInterface
	for rows.Next() {
		iface, err := scanNetworkInterface(rows)
		if err != nil {
			return nil, err
		}
		interfaces = append(interfaces, iface)
	}
	return interfaces, rows.Err()
}

func scanNetworkInterface(row scanner) (*NetworkInterface, error) {
	iface := &NetworkInterface{}
	err := row.Scan(&iface.ID, &iface.DeviceID, &iface.Name, &iface.MacAddress,
		&iface.IPv4, &iface.IPv6, &iface.Status, &iface.Speed, &iface.InterfaceType,
		&iface.IsDefaultGateway, &iface.DNSServers, &iface.MTU,
		&iface.CreatedAt, &iface.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("scan network_interface: %w", err)
	}
	return iface, nil
}
