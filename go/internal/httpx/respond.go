// Package httpx — ответы и разбор запросов в формате, который ждут клиенты
// (он совпадает с Django REST Framework).
package httpx

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
)

func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false) // "&" в ссылках как у DRF, а не "&"
	_ = enc.Encode(v)
}

// Error — ошибка вида {"detail": "..."}.
func Error(w http.ResponseWriter, status int, detail string) {
	JSON(w, status, map[string]string{"detail": detail})
}

func NotFound(w http.ResponseWriter) { Error(w, http.StatusNotFound, "Не найдено.") }

func NoContent(w http.ResponseWriter) { w.WriteHeader(http.StatusNoContent) }

func ServerError(w http.ResponseWriter, r *http.Request, err error) {
	slog.ErrorContext(r.Context(), "request failed", "method", r.Method, "path", r.URL.Path, "err", err)
	Error(w, http.StatusInternalServerError, "Внутренняя ошибка сервера.")
}

// FieldErrors — ошибки валидации вида {"field": ["сообщение"]}.
type FieldErrors map[string][]string

func (fe FieldErrors) Add(field, msg string) { fe[field] = append(fe[field], msg) }

func (fe FieldErrors) Write(w http.ResponseWriter) { JSON(w, http.StatusBadRequest, fe) }

var uuidRe = regexp.MustCompile(`^[0-9a-fA-F]{8}-?[0-9a-fA-F]{4}-?[0-9a-fA-F]{4}-?[0-9a-fA-F]{4}-?[0-9a-fA-F]{12}$`)

func IsUUID(s string) bool { return uuidRe.MatchString(s) }

// IsUniqueViolation — ошибка уникальности Postgres (SQLSTATE 23505). Нужна, когда
// два запроса одновременно проходят предварительную проверку и упираются в constraint.
func IsUniqueViolation(err error) bool {
	var state interface{ SQLState() string }
	if errors.As(err, &state) {
		return state.SQLState() == "23505"
	}
	return err != nil && strings.Contains(err.Error(), "duplicate key value violates unique constraint")
}
