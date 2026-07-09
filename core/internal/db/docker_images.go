package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type DockerImage struct {
	ID         string    `json:"id"`
	DeviceID   string    `json:"device_id"`
	ImageID    string    `json:"image_id"`
	Repository string    `json:"repository"`
	Tag        string    `json:"tag"`
	Size       string    `json:"size"`
	Created    string    `json:"created"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type DockerImageRepository struct {
	pool *pgxpool.Pool
}

func NewDockerImageRepository(pool *pgxpool.Pool) *DockerImageRepository {
	return &DockerImageRepository{pool: pool}
}

func (r *DockerImageRepository) ReplaceForDevice(ctx context.Context, deviceID string, images []*DockerImage) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `DELETE FROM docker_images WHERE device_id = $1`, deviceID); err != nil {
		return fmt.Errorf("delete docker_images: %w", err)
	}

	for _, img := range images {
		if _, err := tx.Exec(ctx,
			`INSERT INTO docker_images (device_id, image_id, repository, tag, size, created)
			 VALUES ($1, $2, $3, $4, $5, $6)`,
			deviceID, img.ImageID, img.Repository, img.Tag, img.Size, img.Created,
		); err != nil {
			return fmt.Errorf("insert docker_image: %w", err)
		}
	}

	return tx.Commit(ctx)
}

func (r *DockerImageRepository) ListByDevice(ctx context.Context, deviceID string) ([]*DockerImage, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, device_id, image_id, repository, tag, size, created, created_at, updated_at
		 FROM docker_images WHERE device_id = $1 ORDER BY repository ASC, tag ASC`,
		deviceID,
	)
	if err != nil {
		return nil, fmt.Errorf("list docker_images: %w", err)
	}
	defer rows.Close()

	var images []*DockerImage
	for rows.Next() {
		img, err := scanDockerImage(rows)
		if err != nil {
			return nil, err
		}
		images = append(images, img)
	}
	return images, rows.Err()
}

func scanDockerImage(row scanner) (*DockerImage, error) {
	img := &DockerImage{}
	err := row.Scan(&img.ID, &img.DeviceID, &img.ImageID, &img.Repository,
		&img.Tag, &img.Size, &img.Created, &img.CreatedAt, &img.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("scan docker_image: %w", err)
	}
	return img, nil
}
