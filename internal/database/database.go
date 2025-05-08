package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
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
	return &DB{Conn: conn}, nil
}

func (db *DB) Ping(ctx context.Context) error {
	return db.Conn.Ping(ctx)
}

func (db *DB) Close() {
	db.Conn.Close(context.Background())
}
