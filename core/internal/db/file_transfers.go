package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type FileTransfer struct {
	ID             string     `json:"id"`
	DeviceID       string     `json:"device_id"`
	UserID         string     `json:"user_id"`
	ActionID       *string    `json:"action_id,omitempty"`
	TransferType   string     `json:"transfer_type"`
	Filename       string     `json:"filename"`
	RemotePath     string     `json:"remote_path"`
	TotalSize      int64      `json:"total_size"`
	ChunkSize      int        `json:"chunk_size"`
	TotalChunks    int        `json:"total_chunks"`
	UploadedChunks int        `json:"uploaded_chunks"`
	Status         string     `json:"status"`
	MimeType       string     `json:"mime_type"`
	Error          string     `json:"error"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
}

type FileTransferRepository struct {
	pool *pgxpool.Pool
}

func NewFileTransferRepository(pool *pgxpool.Pool) *FileTransferRepository {
	return &FileTransferRepository{pool: pool}
}

func (r *FileTransferRepository) Create(ctx context.Context, deviceID, userID string, transferType, filename, remotePath string, totalSize int64, chunkSize, totalChunks int) (*FileTransfer, error) {
	row := r.pool.QueryRow(ctx,
		`INSERT INTO file_transfers (device_id, user_id, transfer_type, filename, remote_path, total_size, chunk_size, total_chunks, status)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'pending')
		 RETURNING id, device_id, user_id, action_id, transfer_type, filename, remote_path,
		           total_size, chunk_size, total_chunks, uploaded_chunks, status, mime_type, error,
		           created_at, updated_at, completed_at`,
		deviceID, userID, transferType, filename, remotePath, totalSize, chunkSize, totalChunks,
	)
	return scanFileTransfer(row)
}

func (r *FileTransferRepository) FindByID(ctx context.Context, id string) (*FileTransfer, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, device_id, user_id, action_id, transfer_type, filename, remote_path,
		        total_size, chunk_size, total_chunks, uploaded_chunks, status, mime_type, error,
		        created_at, updated_at, completed_at
		 FROM file_transfers WHERE id = $1`, id,
	)
	return scanFileTransfer(row)
}

func (r *FileTransferRepository) UpdateStatus(ctx context.Context, id, status string) (*FileTransfer, error) {
	completedClause := ""
	if status == "completed" {
		completedClause = ", completed_at = NOW()"
	}
	row := r.pool.QueryRow(ctx,
		fmt.Sprintf(`UPDATE file_transfers SET status = $1, updated_at = NOW()%s
		 WHERE id = $2
		 RETURNING id, device_id, user_id, action_id, transfer_type, filename, remote_path,
		           total_size, chunk_size, total_chunks, uploaded_chunks, status, mime_type, error,
		           created_at, updated_at, completed_at`, completedClause),
		status, id,
	)
	return scanFileTransfer(row)
}

func (r *FileTransferRepository) IncrementChunks(ctx context.Context, id string, count int) (*FileTransfer, error) {
	row := r.pool.QueryRow(ctx,
		`UPDATE file_transfers SET uploaded_chunks = uploaded_chunks + $1, updated_at = NOW()
		 WHERE id = $2
		 RETURNING id, device_id, user_id, action_id, transfer_type, filename, remote_path,
		           total_size, chunk_size, total_chunks, uploaded_chunks, status, mime_type, error,
		           created_at, updated_at, completed_at`,
		count, id,
	)
	return scanFileTransfer(row)
}

func (r *FileTransferRepository) ListByDevice(ctx context.Context, deviceID string, limit, offset int) ([]*FileTransfer, int, error) {
	var total int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM file_transfers WHERE device_id = $1`, deviceID,
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count file transfers: %w", err)
	}

	rows, err := r.pool.Query(ctx,
		`SELECT id, device_id, user_id, action_id, transfer_type, filename, remote_path,
		        total_size, chunk_size, total_chunks, uploaded_chunks, status, mime_type, error,
		        created_at, updated_at, completed_at
		 FROM file_transfers WHERE device_id = $1
		 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		deviceID, limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list file transfers: %w", err)
	}
	defer rows.Close()

	var transfers []*FileTransfer
	for rows.Next() {
		t, err := scanFileTransfer(rows)
		if err != nil {
			return nil, 0, err
		}
		transfers = append(transfers, t)
	}
	return transfers, total, rows.Err()
}

func (r *FileTransferRepository) CancelStale(ctx context.Context, maxAge time.Duration) (int, error) {
	result, err := r.pool.Exec(ctx,
		`UPDATE file_transfers SET status = 'cancelled', error = 'transfer timed out', updated_at = NOW()
		 WHERE status IN ('pending', 'uploading', 'downloading')
		   AND updated_at < NOW() - $1::interval`,
		fmt.Sprintf("%d seconds", int(maxAge.Seconds())),
	)
	if err != nil {
		return 0, fmt.Errorf("cancel stale transfers: %w", err)
	}
	return int(result.RowsAffected()), nil
}

// Chunk operations

func (r *FileTransferRepository) StoreChunk(ctx context.Context, transferID string, chunkNumber int, data []byte, checksum string) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO file_transfer_chunks (transfer_id, chunk_number, chunk_data, checksum)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (transfer_id, chunk_number) DO UPDATE SET chunk_data = $3, checksum = $4`,
		transferID, chunkNumber, data, checksum,
	)
	return err
}

func (r *FileTransferRepository) GetChunk(ctx context.Context, transferID string, chunkNumber int) ([]byte, error) {
	var data []byte
	err := r.pool.QueryRow(ctx,
		`SELECT chunk_data FROM file_transfer_chunks WHERE transfer_id = $1 AND chunk_number = $2`,
		transferID, chunkNumber,
	).Scan(&data)
	if err != nil {
		return nil, fmt.Errorf("get chunk %d: %w", chunkNumber, err)
	}
	return data, nil
}

func (r *FileTransferRepository) DeleteChunks(ctx context.Context, transferID string) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM file_transfer_chunks WHERE transfer_id = $1`, transferID,
	)
	return err
}

func scanFileTransfer(row scanner) (*FileTransfer, error) {
	t := &FileTransfer{}
	err := row.Scan(&t.ID, &t.DeviceID, &t.UserID, &t.ActionID, &t.TransferType, &t.Filename, &t.RemotePath,
		&t.TotalSize, &t.ChunkSize, &t.TotalChunks, &t.UploadedChunks, &t.Status, &t.MimeType, &t.Error,
		&t.CreatedAt, &t.UpdatedAt, &t.CompletedAt)
	if err != nil {
		return nil, fmt.Errorf("scan file transfer: %w", err)
	}
	return t, nil
}
