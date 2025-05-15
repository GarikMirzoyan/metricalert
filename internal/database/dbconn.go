package database

import (
	"context"
	"database/sql"
)

type DBConn interface {
	Ping(ctx context.Context) error

	Close()

	Exec(ctx context.Context, query string, args ...any) (sql.Result, error)

	QueryRow(ctx context.Context, query string, args ...any) *sql.Row

	Query(ctx context.Context, query string, args ...any) (*sql.Rows, error)

	Begin(ctx context.Context) (*sql.Tx, error)
}
