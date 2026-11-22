// Package migrate применяет SQL-миграции из каталога sql/ при старте сервиса.
//
// Схемой владеет Go: новые изменения добавляются файлами NNNN_name.sql.
// Базу, которую раньше создал Django, сервис узнаёт по таблице accounts_user
// и помечает базовую миграцию применённой, не выполняя её.
package migrate

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"regexp"
	"sort"
	"strings"
)

//go:embed sql/*.sql
var files embed.FS

const baseline = "0001_baseline"

// lockID — ключ pg_advisory_lock: два экземпляра сервиса не применяют миграции одновременно.
const lockID = 7_316_420_501

var nameRe = regexp.MustCompile(`^(\d{4}_[a-z0-9_]+)\.sql$`)

type Migration struct {
	Version string
	SQL     string
}

// List возвращает встроенные миграции по возрастанию версии.
func List() ([]Migration, error) {
	entries, err := fs.ReadDir(files, "sql")
	if err != nil {
		return nil, err
	}
	var out []Migration
	for _, e := range entries {
		m := nameRe.FindStringSubmatch(e.Name())
		if m == nil {
			return nil, fmt.Errorf("migrate: unexpected file name %q (want NNNN_name.sql)", e.Name())
		}
		body, err := files.ReadFile("sql/" + e.Name())
		if err != nil {
			return nil, err
		}
		out = append(out, Migration{Version: m[1], SQL: string(body)})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Version < out[j].Version })
	return out, nil
}

// Up применяет недостающие миграции и возвращает применённые версии.
func Up(ctx context.Context, db *sql.DB) ([]string, error) {
	migrations, err := List()
	if err != nil {
		return nil, err
	}

	conn, err := db.Conn(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	if _, err := conn.ExecContext(ctx, "SELECT pg_advisory_lock($1)", lockID); err != nil {
		return nil, fmt.Errorf("migrate: lock: %w", err)
	}
	defer conn.ExecContext(context.WithoutCancel(ctx), "SELECT pg_advisory_unlock($1)", lockID) //nolint:errcheck

	if _, err := conn.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version text PRIMARY KEY,
		applied_at timestamptz NOT NULL DEFAULT now()
	)`); err != nil {
		return nil, fmt.Errorf("migrate: create schema_migrations: %w", err)
	}

	done := map[string]bool{}
	rows, err := conn.QueryContext(ctx, "SELECT version FROM schema_migrations")
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			rows.Close()
			return nil, err
		}
		done[v] = true
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if !done[baseline] {
		var djangoSchema bool
		if err := conn.QueryRowContext(ctx,
			"SELECT to_regclass('public.accounts_user') IS NOT NULL").Scan(&djangoSchema); err != nil {
			return nil, err
		}
		if djangoSchema {
			if _, err := conn.ExecContext(ctx, "INSERT INTO schema_migrations (version) VALUES ($1)", baseline); err != nil {
				return nil, err
			}
			done[baseline] = true
		}
	}

	var applied []string
	for _, m := range migrations {
		if done[m.Version] {
			continue
		}
		if err := apply(ctx, conn, m); err != nil {
			return applied, err
		}
		applied = append(applied, m.Version)
	}
	return applied, nil
}

func apply(ctx context.Context, conn *sql.Conn, m Migration) error {
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck
	if strings.TrimSpace(m.SQL) != "" {
		if _, err := tx.ExecContext(ctx, m.SQL); err != nil {
			return fmt.Errorf("migrate: %s: %w", m.Version, err)
		}
	}
	if _, err := tx.ExecContext(ctx, "INSERT INTO schema_migrations (version) VALUES ($1)", m.Version); err != nil {
		return fmt.Errorf("migrate: record %s: %w", m.Version, err)
	}
	return tx.Commit()
}
