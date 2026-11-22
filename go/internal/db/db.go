// Package db открывает пул соединений к той же базе Postgres, что и Django.
// Драйвер регистрируется в main (blank import), здесь только database/sql.
package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

func Open(ctx context.Context, driver, dsn string) (*sql.DB, error) {
	conn, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	conn.SetMaxOpenConns(20)
	conn.SetMaxIdleConns(10)
	conn.SetConnMaxLifetime(30 * time.Minute)

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := conn.PingContext(ctx); err != nil {
		conn.Close()
		return nil, fmt.Errorf("ping db: %w", err)
	}
	return conn, nil
}
