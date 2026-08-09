package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ContainerMetric struct {
	ID               int64     `json:"id"`
	DeviceID         string    `json:"device_id"`
	ContainerID      string    `json:"container_id"`
	ContainerName    string    `json:"container_name"`
	CPUPercent       float64   `json:"cpu_percent"`
	MemoryUsageBytes int64     `json:"memory_usage_bytes"`
	MemoryLimitBytes int64     `json:"memory_limit_bytes"`
	MemoryPercent    float64   `json:"memory_percent"`
	NetworkRxBytes   int64     `json:"network_rx_bytes"`
	NetworkTxBytes   int64     `json:"network_tx_bytes"`
	BlockReadBytes   int64     `json:"block_read_bytes"`
	BlockWriteBytes  int64     `json:"block_write_bytes"`
	Pids             int       `json:"pids"`
	RecordedAt       time.Time `json:"recorded_at"`
}

type ContainerMetricSnapshot struct {
	DeviceID   string             `json:"device_id"`
	Containers []*ContainerMetric `json:"containers"`
}

type ContainerMetricsRepository struct {
	pool *pgxpool.Pool
}

func NewContainerMetricsRepository(pool *pgxpool.Pool) *ContainerMetricsRepository {
	return &ContainerMetricsRepository{pool: pool}
}

// InsertMetrics inserts a batch of container metrics for a device
func (r *ContainerMetricsRepository) InsertMetrics(ctx context.Context, deviceID string, metrics []*ContainerMetric) error {
	if len(metrics) == 0 {
		return nil
	}

	batch := &pgx.Batch{}
	for _, m := range metrics {
		batch.Queue(
			`INSERT INTO container_metrics (device_id, container_id, container_name, cpu_percent, memory_usage_bytes, memory_limit_bytes, memory_percent, network_rx_bytes, network_tx_bytes, block_read_bytes, block_write_bytes, pids)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
			deviceID, m.ContainerID, m.ContainerName, m.CPUPercent, m.MemoryUsageBytes, m.MemoryLimitBytes,
			m.MemoryPercent, m.NetworkRxBytes, m.NetworkTxBytes, m.BlockReadBytes, m.BlockWriteBytes, m.Pids,
		)
	}

	br := r.pool.SendBatch(ctx, batch)
	for i := 0; i < batch.Len(); i++ {
		if _, err := br.Exec(); err != nil {
			br.Close()
			return fmt.Errorf("insert metric %d: %w", i, err)
		}
	}
	br.Close()
	return nil
}

// GetLatest returns the most recent metrics for each container on a device
func (r *ContainerMetricsRepository) GetLatest(ctx context.Context, deviceID string) ([]*ContainerMetric, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT DISTINCT ON (container_id)
		        id, device_id, container_id, container_name, cpu_percent, memory_usage_bytes,
		        memory_limit_bytes, memory_percent, network_rx_bytes, network_tx_bytes,
		        block_read_bytes, block_write_bytes, pids, recorded_at
		 FROM container_metrics WHERE device_id = $1
		 ORDER BY container_id, recorded_at DESC`,
		deviceID,
	)
	if err != nil {
		return nil, fmt.Errorf("get latest container metrics: %w", err)
	}
	defer rows.Close()

	var list []*ContainerMetric
	for rows.Next() {
		m := &ContainerMetric{}
		if err := rows.Scan(&m.ID, &m.DeviceID, &m.ContainerID, &m.ContainerName, &m.CPUPercent,
			&m.MemoryUsageBytes, &m.MemoryLimitBytes, &m.MemoryPercent, &m.NetworkRxBytes, &m.NetworkTxBytes,
			&m.BlockReadBytes, &m.BlockWriteBytes, &m.Pids, &m.RecordedAt); err != nil {
			return nil, fmt.Errorf("scan container metric: %w", err)
		}
		list = append(list, m)
	}
	return list, rows.Err()
}

// GetHistory returns time-series metrics for a specific container
func (r *ContainerMetricsRepository) GetHistory(ctx context.Context, deviceID, containerID string, minutes int) ([]*ContainerMetric, error) {
	if minutes <= 0 {
		minutes = 30
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id, device_id, container_id, container_name, cpu_percent, memory_usage_bytes,
		        memory_limit_bytes, memory_percent, network_rx_bytes, network_tx_bytes,
		        block_read_bytes, block_write_bytes, pids, recorded_at
		 FROM container_metrics WHERE device_id = $1 AND container_id = $2
		   AND recorded_at > NOW() - ($3 || ' minutes')::interval
		 ORDER BY recorded_at ASC`,
		deviceID, containerID, fmt.Sprintf("%d", minutes),
	)
	if err != nil {
		return nil, fmt.Errorf("get container metric history: %w", err)
	}
	defer rows.Close()

	var list []*ContainerMetric
	for rows.Next() {
		m := &ContainerMetric{}
		if err := rows.Scan(&m.ID, &m.DeviceID, &m.ContainerID, &m.ContainerName, &m.CPUPercent,
			&m.MemoryUsageBytes, &m.MemoryLimitBytes, &m.MemoryPercent, &m.NetworkRxBytes, &m.NetworkTxBytes,
			&m.BlockReadBytes, &m.BlockWriteBytes, &m.Pids, &m.RecordedAt); err != nil {
			return nil, fmt.Errorf("scan container metric: %w", err)
		}
		list = append(list, m)
	}
	return list, rows.Err()
}
