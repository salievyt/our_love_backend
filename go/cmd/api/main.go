// Команда api — Go-версия бэкенда Our Love.
//
// Сервис постепенно забирает разделы API у Django: работает с той же базой и
// принимает те же JWT. Какие пути обслуживает Go, решает nginx.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	_ "time/tzdata" // часовые пояса внутри бинарника (образ без tzdata)

	_ "github.com/lib/pq"

	"github.com/salievyt/our_love_backend/go/internal/config"
	"github.com/salievyt/our_love_backend/go/internal/db"
	"github.com/salievyt/our_love_backend/go/internal/server"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	if err := run(); err != nil {
		slog.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	conn, err := db.Open(ctx, "postgres", cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer conn.Close()

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           server.New(server.Deps{DB: conn, SecretKey: cfg.SecretKey, Location: cfg.Location}),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	errc := make(chan error, 1)
	go func() {
		slog.Info("listening", "addr", cfg.Addr)
		errc <- srv.ListenAndServe()
	}()

	select {
	case err := <-errc:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	case <-ctx.Done():
		slog.Info("shutting down")
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}
