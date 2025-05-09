package database

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type DBConn interface {
	Ping(ctx context.Context) error

	Close()

	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)

	QueryRow(ctx context.Context, sql string, arguments ...any) pgx.Row

	Query(ctx context.Context, sql string, arguments ...any) (pgx.Rows, error)

	Begin(ctx context.Context) (pgx.Tx, error)
}
