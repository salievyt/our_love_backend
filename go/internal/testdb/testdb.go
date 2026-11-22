//go:build integration

// Package testdb — база для интеграционных тестов. Схему создают миграции Django
// (python manage.py migrate), поэтому тесты проверяют Go-код на настоящей схеме.
package testdb

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"

	"github.com/salievyt/our_love_backend/go/internal/auth"
	"github.com/salievyt/our_love_backend/go/internal/server"
)

const Secret = "test-key-for-local-test-run-only-32b"

type Env struct {
	t   *testing.T
	DB  *sql.DB
	srv http.Handler
}

// New открывает TEST_DATABASE_URL и очищает пользователей (каскадно — все их данные).
func New(t *testing.T) *Env {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL не задан")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if _, err := db.Exec("TRUNCATE accounts_user CASCADE"); err != nil {
		t.Fatal(err)
	}
	loc, _ := time.LoadLocation("Europe/Moscow")
	return &Env{t: t, DB: db, srv: server.New(server.Deps{DB: db, SecretKey: Secret, Location: loc})}
}

func (e *Env) CreateUser(username string) string {
	e.t.Helper()
	var id string
	err := e.DB.QueryRow(`
		INSERT INTO accounts_user (id, password, is_superuser, username, first_name, last_name, email,
		                           is_staff, is_active, date_joined, phone)
		VALUES (gen_random_uuid(), '!', false, $1, '', '', '', false, true, now(), '')
		RETURNING id`, username).Scan(&id)
	if err != nil {
		e.t.Fatal(err)
	}
	return id
}

func (e *Env) CreatePair(partner1, partner2 string) string {
	e.t.Helper()
	var id string
	err := e.DB.QueryRow(`
		INSERT INTO accounts_relationship (id, partner1_id, partner2_id, partner1_name, partner2_name,
		                                   partner1_emoji, partner2_emoji, relationship_start_date, created_at, updated_at)
		VALUES (gen_random_uuid(), $1, $2, 'A', 'B', '💙', '💖', '2024-01-01', now(), now())
		RETURNING id`, partner1, partner2).Scan(&id)
	if err != nil {
		e.t.Fatal(err)
	}
	return id
}

func Token(userID string) string {
	return auth.SignToken(map[string]any{
		"token_type": "access", "exp": time.Now().Add(time.Hour).Unix(),
		"iat": time.Now().Unix(), "jti": "t", "user_id": userID,
	}, Secret)
}

type Response struct {
	Status int
	Body   []byte
}

func (r Response) JSON(t *testing.T) map[string]any {
	t.Helper()
	var v map[string]any
	if err := json.Unmarshal(r.Body, &v); err != nil {
		t.Fatalf("decode %s: %v", r.Body, err)
	}
	return v
}

// Results — список results из страницы пагинации.
func (r Response) Results(t *testing.T) []any {
	t.Helper()
	res, ok := r.JSON(t)["results"].([]any)
	if !ok {
		t.Fatalf("no results in %s", r.Body)
	}
	return res
}

// Do выполняет запрос от имени userID ("" — без токена). body — JSON-значение или nil.
func (e *Env) Do(method, path, userID string, body any) Response {
	e.t.Helper()
	var rd io.Reader
	if body != nil {
		raw, _ := json.Marshal(body)
		rd = bytes.NewReader(raw)
	}
	req := httptest.NewRequest(method, path, rd)
	req.Header.Set("Content-Type", "application/json")
	if userID != "" {
		req.Header.Set("Authorization", "Bearer "+Token(userID))
	}
	rec := httptest.NewRecorder()
	e.srv.ServeHTTP(rec, req)
	return Response{Status: rec.Code, Body: rec.Body.Bytes()}
}

func (e *Env) Expect(r Response, status int) Response {
	e.t.Helper()
	if r.Status != status {
		e.t.Fatalf("status = %d, want %d; body: %s", r.Status, status, r.Body)
	}
	return r
}
