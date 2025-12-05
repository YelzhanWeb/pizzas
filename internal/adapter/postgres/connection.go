package postgres

import (
	"context"
	"fmt"
	"wheres-my-pizza/internal/config"

	"github.com/jackc/pgx/v5/pgxpool"
)

type DB interface {
	Query(ctx context.Context, sql string, args ...interface{}) (Rows, error)
	QueryRow(ctx context.Context, sql string, args ...interface{}) Row
	Exec(ctx context.Context, sql string, args ...interface{}) (CommandTag, error)
	Begin(ctx context.Context) (Tx, error)
	Close()
}

type Rows interface {
	Next() bool
	Scan(dest ...interface{}) error
	Close()
}

type Row interface {
	Scan(dest ...interface{}) error
}

type Tx interface {
	Query(ctx context.Context, sql string, args ...interface{}) (Rows, error)
	QueryRow(ctx context.Context, sql string, args ...interface{}) Row
	Exec(ctx context.Context, sql string, args ...interface{}) (CommandTag, error)
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
}

type CommandTag interface {
	RowsAffected() int64
}

type pgxDB struct {
	pool *pgxpool.Pool
}

func Connect(ctx context.Context, cfg config.DatabaseConfig) (DB, error) {
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Database)

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &pgxDB{pool: pool}, nil
}

func (db *pgxDB) Query(ctx context.Context, sql string, args ...interface{}) (Rows, error) {
	return db.pool.Query(ctx, sql, args...)
}

func (db *pgxDB) QueryRow(ctx context.Context, sql string, args ...interface{}) Row {
	return db.pool.QueryRow(ctx, sql, args...)
}

func (db *pgxDB) Exec(ctx context.Context, sql string, args ...interface{}) (CommandTag, error) {
	return db.pool.Exec(ctx, sql, args...)
}

func (db *pgxDB) Begin(ctx context.Context) (Tx, error) {
	return db.pool.Begin(ctx)
}

func (db *pgxDB) Close() {
	db.pool.Close()
}
