package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Структура для хранения пула подключений
type DB struct {
	Conn *pgx.Conn
}

// Функция для создания подключения к базе
func NewDBConnection(connString string) (*DB, error) {
	conn, err := pgx.Connect(context.Background(), connString)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to database: %v", err)
	}

	db := &DB{Conn: conn}
	if err := db.initSchema(); err != nil {
		conn.Close(context.Background())
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return &DB{Conn: conn}, nil
}

func (db *DB) Ping(ctx context.Context) error {
	return db.Conn.Ping(ctx)
}

func (db *DB) Close() {
	db.Conn.Close(context.Background())
}

func (db *DB) initSchema() error {
	_, err := db.Conn.Exec(context.Background(), `
        CREATE TABLE IF NOT EXISTS metrics (
            id SERIAL PRIMARY KEY,
            name TEXT NOT NULL UNIQUE,
            type TEXT NOT NULL,
            value DOUBLE PRECISION
        )
    `)
	return err
}

func (db *DB) Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	return db.Conn.Exec(ctx, sql, arguments...)
}

func (db *DB) QueryRow(ctx context.Context, sql string, arguments ...any) pgx.Row {
	return db.Conn.QueryRow(ctx, sql, arguments...)
}

func (db *DB) Query(ctx context.Context, sql string, arguments ...any) (pgx.Rows, error) {
	return db.Conn.Query(ctx, sql, arguments...)
}
