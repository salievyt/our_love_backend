//go:build integration

package migrate_test

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/salievyt/our_love_backend/go/internal/migrate"
	_ "github.com/salievyt/our_love_backend/go/internal/testdb" // регистрирует драйвер postgres
)

func open(t *testing.T, env string) *sql.DB {
	t.Helper()
	dsn := os.Getenv(env)
	if dsn == "" {
		t.Skip(env + " не задан")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// Пустая база: базовая миграция создаёт схему, повторный запуск ничего не делает.
func TestUpOnEmptyDatabase(t *testing.T) {
	db := open(t, "TEST_EMPTY_DATABASE_URL")
	ctx := context.Background()
	if _, err := db.Exec("DROP SCHEMA public CASCADE; CREATE SCHEMA public"); err != nil {
		t.Fatal(err)
	}
	applied, err := migrate.Up(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	if len(applied) == 0 || applied[0] != "0001_baseline" {
		t.Fatalf("applied = %v", applied)
	}
	var n int
	if err := db.QueryRow("SELECT count(*) FROM information_schema.tables WHERE table_schema = 'public' AND table_name LIKE 'accounts_%'").Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n < 5 {
		t.Fatalf("accounts tables = %d", n)
	}
	again, err := migrate.Up(ctx, db)
	if err != nil || len(again) != 0 {
		t.Fatalf("second run applied %v, err %v", again, err)
	}
}

// База, созданная Django: базовая миграция только помечается, данные не трогаются.
func TestUpOnDjangoDatabase(t *testing.T) {
	db := open(t, "TEST_DATABASE_URL")
	ctx := context.Background()
	if _, err := db.Exec("DROP TABLE IF EXISTS schema_migrations"); err != nil {
		t.Fatal(err)
	}
	applied, err := migrate.Up(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range applied {
		if v == "0001_baseline" {
			t.Fatal("baseline executed on a Django database")
		}
	}
	var marked bool
	if err := db.QueryRow("SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE version = '0001_baseline')").Scan(&marked); err != nil || !marked {
		t.Fatalf("baseline not marked: %v", err)
	}
}
