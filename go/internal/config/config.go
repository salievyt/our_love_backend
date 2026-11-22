// Package config читает настройки из тех же переменных окружения, что и Django,
// чтобы оба сервиса можно было запускать с одним .env.
package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"
)

type Config struct {
	SecretKey   string
	DatabaseURL string
	Addr        string
	Location    *time.Location
	// MediaRoot — каталог загрузок, общий том с Django (MEDIA_ROOT).
	MediaRoot string
	// MediaAccel — файлы отдаёт nginx через X-Accel-Redirect (MEDIA_ACCEL=1 в docker-compose).
	MediaAccel bool
}

func Load() (Config, error) {
	key := os.Getenv("SECRET_KEY")
	if len(key) < 32 || strings.HasPrefix(key, "change-me") {
		return Config{}, errors.New("SECRET_KEY не задан или слишком простой (нужно ≥32 символов, не change-me…)")
	}

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		password := os.Getenv("DB_PASSWORD")
		if password == "" {
			return Config{}, errors.New("DB_PASSWORD не задан")
		}
		u := url.URL{
			Scheme:   "postgres",
			User:     url.UserPassword(env("DB_USER", "outlove"), password),
			Host:     env("DB_HOST", "db") + ":" + env("DB_PORT", "5432"),
			Path:     "/" + env("DB_NAME", "outlove"),
			RawQuery: "sslmode=" + env("DB_SSLMODE", "disable"),
		}
		dsn = u.String()
	}

	loc, err := time.LoadLocation(env("TIME_ZONE", "Europe/Moscow"))
	if err != nil {
		return Config{}, fmt.Errorf("TIME_ZONE: %w", err)
	}

	return Config{
		SecretKey:   key,
		DatabaseURL: dsn,
		Addr:        ":" + env("PORT", "8080"),
		Location:    loc,
		MediaRoot:   env("MEDIA_ROOT", "/app/media"),
		MediaAccel:  os.Getenv("MEDIA_ACCEL") == "1",
	}, nil
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
