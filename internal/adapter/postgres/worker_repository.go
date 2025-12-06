package postgres

import (
	"context"
	"fmt"
	"time"
	"wheres-my-pizza/internal/domain"
	"wheres-my-pizza/internal/interfaces"
)

type workerRepository struct {
	db DB
}

func NewWorkerRepository(db DB) interfaces.WorkerRepository {
	return &workerRepository{db: db}
}

func (r *workerRepository) Create(ctx context.Context, worker *domain.Worker) error {
	query := `
		INSERT INTO workers (name, type, status, last_seen, orders_processed, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`
	err := r.db.QueryRow(ctx, query,
		worker.Name, worker.Type, worker.Status, worker.LastSeen, worker.OrdersProcessed, worker.CreatedAt,
	).Scan(&worker.ID)
	if err != nil {
		return fmt.Errorf("failed to create worker: %w", err)
	}
	return nil
}

func (r *workerRepository) FindByName(ctx context.Context, name string) (*domain.Worker, error) {
	query := `
		SELECT id, name, type, status, last_seen, orders_processed, created_at
		FROM workers
		WHERE name = $1
	`

	var worker domain.Worker
	err := r.db.QueryRow(ctx, query, name).Scan(
		&worker.ID, &worker.Name, &worker.Type, &worker.Status,
		&worker.LastSeen, &worker.OrdersProcessed, &worker.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("worker not found: %w", err)
	}

	return &worker, nil
}

func (r *workerRepository) Update(ctx context.Context, worker *domain.Worker) error {
	query := `
		UPDATE workers
		SET type = $1, status = $2, last_seen = $3, orders_processed = $4
		WHERE id = $5
	`
	_, err := r.db.Exec(ctx, query,
		worker.Type, worker.Status, worker.LastSeen, worker.OrdersProcessed, worker.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update worker: %w", err)
	}
	return nil
}

func (r *workerRepository) UpdateHeartbeat(ctx context.Context, name string) error {
	query := `
		UPDATE workers
		SET last_seen = $1, status = $2
		WHERE name = $3
	`
	_, err := r.db.Exec(ctx, query, time.Now(), domain.WorkerStatusOnline, name)
	if err != nil {
		return fmt.Errorf("failed to update heartbeat: %w", err)
	}
	return nil
}

func (r *workerRepository) ListAll(ctx context.Context) ([]*domain.Worker, error) {
	query := `
		SELECT id, name, type, status, last_seen, orders_processed, created_at
		FROM workers
		ORDER BY name
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list workers: %w", err)
	}
	defer rows.Close()

	var workers []*domain.Worker
	for rows.Next() {
		var worker domain.Worker
		if err := rows.Scan(
			&worker.ID, &worker.Name, &worker.Type, &worker.Status,
			&worker.LastSeen, &worker.OrdersProcessed, &worker.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan worker: %w", err)
		}
		workers = append(workers, &worker)
	}

	return workers, nil
}

func (r *workerRepository) IncrementOrdersProcessed(ctx context.Context, name string) error {
	query := `
		UPDATE workers
		SET orders_processed = orders_processed + 1
		WHERE name = $1
	`
	_, err := r.db.Exec(ctx, query, name)
	if err != nil {
		return fmt.Errorf("failed to increment orders processed: %w", err)
	}
	return nil
}
