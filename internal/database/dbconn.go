package database

import "context"

type DBConn interface {
	Ping(ctx context.Context) error

	Close()
}
