package database

import (
	"context"
	"database/sql"
)

// DBConn abstracts sql.DB for easier testing and mocking.
type DBConn interface {
	Ping(ctx context.Context) error

	Close()

	Exec(ctx context.Context, query string, args ...any) (sql.Result, error)

	QueryRow(ctx context.Context, query string, args ...any) *sql.Row

	Query(ctx context.Context, query string, args ...any) (*sql.Rows, error)

	Begin(ctx context.Context) (*sql.Tx, error)
}
