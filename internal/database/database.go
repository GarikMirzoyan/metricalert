package database

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

// Структура для хранения пула подключений
type DB struct {
	Conn *sql.DB
}

// Функция для создания подключения к базе
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

// Прогон миграций через goose
func (db *DB) RunMigrations() error {
	goose.SetDialect("postgres")
	return goose.Up(db.Conn, "../../migrations")
}

// Проверка соединения
func (db *DB) Ping(ctx context.Context) error {
	return db.Conn.PingContext(ctx)
}

// Закрытие соединения
func (db *DB) Close() {
	db.Conn.Close()
}

// Выполнение SQL-запроса (INSERT/UPDATE/DELETE)
func (db *DB) Exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return db.Conn.ExecContext(ctx, query, args...)
}

// Получение одной строки
func (db *DB) QueryRow(ctx context.Context, query string, args ...any) *sql.Row {
	return db.Conn.QueryRowContext(ctx, query, args...)
}

// Получение нескольких строк
func (db *DB) Query(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return db.Conn.QueryContext(ctx, query, args...)
}

// Начало транзакции
func (db *DB) Begin(ctx context.Context) (*sql.Tx, error) {
	return db.Conn.BeginTx(ctx, nil)
}
