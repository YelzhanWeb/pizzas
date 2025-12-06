package postgres

import (
	"context"
	"fmt"
	"wheres-my-pizza/internal/config"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Используем any вместо interface{} (Go 1.18+), так как pgx/v5 использует any
type DB interface {
	Query(ctx context.Context, sql string, args ...any) (Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) Row
	Exec(ctx context.Context, sql string, args ...any) (CommandTag, error)
	Begin(ctx context.Context) (Tx, error)
	Close()
}

type Rows interface {
	Next() bool
	Scan(dest ...any) error
	Close()
}

type Row interface {
	Scan(dest ...any) error
}

type Tx interface {
	Query(ctx context.Context, sql string, args ...any) (Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) Row
	Exec(ctx context.Context, sql string, args ...any) (CommandTag, error)
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
}

type CommandTag interface {
	RowsAffected() int64
}

// --- Implementation ---

type pgxDB struct {
	pool *pgxpool.Pool
}

// pgxTx - это обертка над pgx.Tx, чтобы удовлетворить интерфейс Tx
type pgxTx struct {
	tx pgx.Tx
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

// Реализация методов для pgxDB

func (db *pgxDB) Query(ctx context.Context, sql string, args ...any) (Rows, error) {
	return db.pool.Query(ctx, sql, args...)
}

func (db *pgxDB) QueryRow(ctx context.Context, sql string, args ...any) Row {
	return db.pool.QueryRow(ctx, sql, args...)
}

func (db *pgxDB) Exec(ctx context.Context, sql string, args ...any) (CommandTag, error) {
	return db.pool.Exec(ctx, sql, args...)
}

func (db *pgxDB) Begin(ctx context.Context) (Tx, error) {
	// Мы получаем оригинальную pgx транзакцию
	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	// И оборачиваем её в нашу структуру
	return &pgxTx{tx: tx}, nil
}

func (db *pgxDB) Close() {
	db.pool.Close()
}

// Реализация методов для pgxTx (нашей обертки)

func (t *pgxTx) Query(ctx context.Context, sql string, args ...any) (Rows, error) {
	return t.tx.Query(ctx, sql, args...)
}

func (t *pgxTx) QueryRow(ctx context.Context, sql string, args ...any) Row {
	return t.tx.QueryRow(ctx, sql, args...)
}

func (t *pgxTx) Exec(ctx context.Context, sql string, args ...any) (CommandTag, error) {
	// Здесь происходит магия: pgx возвращает pgconn.CommandTag
	// pgconn.CommandTag удовлетворяет интерфейсу CommandTag (имеет метод RowsAffected)
	// Поэтому мы можем вернуть результат напрямую
	return t.tx.Exec(ctx, sql, args...)
}

func (t *pgxTx) Commit(ctx context.Context) error {
	return t.tx.Commit(ctx)
}

func (t *pgxTx) Rollback(ctx context.Context) error {
	return t.tx.Rollback(ctx)
}
