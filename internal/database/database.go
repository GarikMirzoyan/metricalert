package database

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"runtime"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

// DB wraps *sql.DB and provides helpers used across the app.
type DB struct {
	Conn *sql.DB
}

// NewDBConnection establishes a PostgreSQL connection and pings it.
func NewDBConnection(connString string) (*DB, error) {
	conn, err := sql.Open("pgx", connString)
	if err != nil {
		return nil, fmt.Errorf("unable to open database: %v", err)
	}

	// Проверим, что соединение рабочее
	if err = conn.Ping(); err != nil {
		return nil, fmt.Errorf("unable to ping database: %v", err)
	}

	return &DB{Conn: conn}, nil
}

func getProjectRoot() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "../../")
}

// RunMigrations applies goose migrations from the local migrations directory.
func (db *DB) RunMigrations() error {
	goose.SetDialect("postgres")
	migrationsPath := filepath.Join(getProjectRoot(), "migrations")
	return goose.Up(db.Conn, migrationsPath)
}

// Ping checks database connectivity with context.
func (db *DB) Ping(ctx context.Context) error {
	return db.Conn.PingContext(ctx)
}

// Close closes the underlying DB connection.
func (db *DB) Close() {
	db.Conn.Close()
}

// Exec executes a statement (INSERT/UPDATE/DELETE).
func (db *DB) Exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return db.Conn.ExecContext(ctx, query, args...)
}

// QueryRow executes a query expected to return at most one row.
func (db *DB) QueryRow(ctx context.Context, query string, args ...any) *sql.Row {
	return db.Conn.QueryRowContext(ctx, query, args...)
}

// Query executes a query returning multiple rows.
func (db *DB) Query(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return db.Conn.QueryContext(ctx, query, args...)
}

// Begin starts a transaction.
func (db *DB) Begin(ctx context.Context) (*sql.Tx, error) {
	return db.Conn.BeginTx(ctx, nil)
}
