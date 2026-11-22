// Package server собирает HTTP-обработчики всех перенесённых разделов.
package server

import (
	"database/sql"
	"log/slog"
	"net/http"
	"time"

	"github.com/salievyt/our_love_backend/go/internal/auth"
	"github.com/salievyt/our_love_backend/go/internal/httpx"
	"github.com/salievyt/our_love_backend/go/internal/moods"
)

type Deps struct {
	DB        *sql.DB
	SecretKey string
	Location  *time.Location
}

func New(d Deps) http.Handler {
	mux := http.NewServeMux()
	authed := func(h http.Handler) http.Handler { return auth.Middleware(d.DB, d.SecretKey, h) }

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		if err := d.DB.PingContext(r.Context()); err != nil {
			httpx.Error(w, http.StatusServiceUnavailable, "database unavailable")
			return
		}
		httpx.JSON(w, http.StatusOK, map[string]string{"status": "OK", "service": "api-go"})
	})

	(&moods.Handler{DB: d.DB, Loc: d.Location}).Register(mux, authed)

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { httpx.NotFound(w) })
	return recoverer(logRequests(mux))
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sw, r)
		slog.InfoContext(r.Context(), "request", "method", r.Method, "path", r.URL.Path,
			"status", sw.status, "duration", time.Since(start))
	})
}

func recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if v := recover(); v != nil {
				slog.ErrorContext(r.Context(), "panic", "value", v, "path", r.URL.Path)
				httpx.Error(w, http.StatusInternalServerError, "Внутренняя ошибка сервера.")
			}
		}()
		next.ServeHTTP(w, r)
	})
}
