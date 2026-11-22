// Package dates — /api/v1/dates/: важные даты пары.
// Порт apps/dates из Django; видимость — только даты своей пары.
package dates

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/salievyt/our_love_backend/go/internal/auth"
	"github.com/salievyt/our_love_backend/go/internal/httpx"
)

const (
	defaultEmoji = "❤️"
	upcomingMax  = 5
	columns      = "id, relationship_id, title, date, emoji, is_annually, created_at"
)

type Handler struct {
	DB  *sql.DB
	Loc *time.Location
}

type importantDate struct {
	ID           string `json:"id"`
	Relationship string `json:"relationship"`
	Title        string `json:"title"`
	Date         string `json:"date"`
	Emoji        string `json:"emoji"`
	IsAnnually   bool   `json:"is_annually"`
	DaysUntil    int    `json:"days_until"`
	CreatedAt    string `json:"created_at"`
}

func (h *Handler) Register(mux *http.ServeMux, wrap func(http.Handler) http.Handler) {
	route := func(pattern string, fn http.HandlerFunc) { mux.Handle(pattern, wrap(fn)) }
	route("GET /api/v1/dates/{$}", h.list)
	route("POST /api/v1/dates/{$}", h.create)
	route("GET /api/v1/dates/upcoming/{$}", h.upcoming)
	route("GET /api/v1/dates/{id}/{$}", h.detail)
	route("PATCH /api/v1/dates/{id}/{$}", h.update(true))
	route("PUT /api/v1/dates/{id}/{$}", h.update(false))
	route("DELETE /api/v1/dates/{id}/{$}", h.delete)
}

func (h *Handler) scan(row interface{ Scan(...any) error }) (importantDate, error) {
	var (
		d         importantDate
		date      time.Time
		createdAt time.Time
	)
	if err := row.Scan(&d.ID, &d.Relationship, &d.Title, &date, &d.Emoji, &d.IsAnnually, &createdAt); err != nil {
		return importantDate{}, err
	}
	d.Date = httpx.Date(date)
	d.DaysUntil = DaysUntil(date, httpx.Today(h.Loc), d.IsAnnually)
	d.CreatedAt = httpx.DateTime(createdAt, h.Loc)
	return d, nil
}

func (h *Handler) query(ctx context.Context, query string, args ...any) ([]importantDate, error) {
	rows, err := h.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []importantDate{}
	for rows.Next() {
		d, err := h.scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

var ordering = map[string]string{
	"id": "id", "relationship": "relationship_id", "title": "title", "date": "date",
	"emoji": "emoji", "is_annually": "is_annually", "created_at": "created_at",
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	page, ok := httpx.PageNumber(r)
	if !ok {
		httpx.Error(w, http.StatusNotFound, "Неправильная страница.")
		return
	}
	user := auth.FromContext(r.Context())
	if !user.HasPair() {
		httpx.WritePage(w, r, page, 0, []importantDate{})
		return
	}
	cond, args := "relationship_id = $1", []any{user.RelationshipID}
	if v := r.URL.Query().Get("is_annually"); v != "" {
		b, ok := parseBoolParam(v)
		if !ok {
			httpx.FieldErrors{"is_annually": {"Выберите корректный вариант."}}.Write(w)
			return
		}
		args = append(args, b)
		cond += " AND is_annually = $2"
	}

	var total int
	if err := h.DB.QueryRowContext(r.Context(), "SELECT count(*) FROM dates_importantdate WHERE "+cond, args...).Scan(&total); err != nil {
		httpx.ServerError(w, r, err)
		return
	}
	order := httpx.OrderBy(r, ordering, "date")
	results, err := h.query(r.Context(), fmt.Sprintf(
		"SELECT %s FROM dates_importantdate WHERE %s ORDER BY %s, id LIMIT %d OFFSET %d",
		columns, cond, order, httpx.PageSize, (page-1)*httpx.PageSize), args...)
	if err != nil {
		httpx.ServerError(w, r, err)
		return
	}
	httpx.WritePage(w, r, page, total, results)
}

// parseBoolParam — как BooleanFilter в django-filter.
func parseBoolParam(v string) (bool, bool) {
	switch v {
	case "true", "True", "1":
		return true, true
	case "false", "False", "0":
		return false, true
	}
	return false, false
}

func (h *Handler) upcoming(w http.ResponseWriter, r *http.Request) {
	user := auth.FromContext(r.Context())
	results := []importantDate{}
	if user.HasPair() {
		all, err := h.query(r.Context(),
			"SELECT "+columns+" FROM dates_importantdate WHERE relationship_id = $1 ORDER BY date, id", user.RelationshipID)
		if err != nil {
			httpx.ServerError(w, r, err)
			return
		}
		sort.SliceStable(all, func(i, j int) bool { return all[i].DaysUntil < all[j].DaysUntil })
		results = all[:min(len(all), upcomingMax)]
	}
	httpx.WritePage(w, r, 1, len(results), results)
}

type fields struct {
	title, emoji       string
	date               time.Time
	annually           bool
	hasTitle, hasEmoji bool
	hasDate, hasAnnual bool
}

func readFields(w http.ResponseWriter, r *http.Request, partial bool) (fields, bool) {
	p, err := httpx.DecodePayload(r)
	if err != nil {
		httpx.BadRequest(w, err)
		return fields{}, false
	}
	var f fields
	f.title, f.hasTitle = p.String("title", !partial, false, 200)
	f.date, f.hasDate = p.Date("date", !partial)
	f.emoji, f.hasEmoji = p.String("emoji", false, false, 10)
	f.annually, f.hasAnnual = p.Bool("is_annually", false)
	if len(p.Errors) > 0 {
		p.Errors.Write(w)
		return fields{}, false
	}
	return f, true
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	user := auth.FromContext(r.Context())
	f, ok := readFields(w, r, false)
	if !ok {
		return
	}
	if !user.HasPair() {
		// Django-версия здесь падала с 500 (relationship NOT NULL)
		httpx.Error(w, http.StatusBadRequest, "Сначала создайте пару с партнёром")
		return
	}
	if !f.hasEmoji {
		f.emoji = defaultEmoji
	}
	if !f.hasAnnual {
		f.annually = true
	}
	d, err := h.scan(h.DB.QueryRowContext(r.Context(), `
		INSERT INTO dates_importantdate (id, relationship_id, title, date, emoji, is_annually, created_by_id, created_at)
		VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, now())
		RETURNING `+columns, user.RelationshipID, f.title, f.date, f.emoji, f.annually, user.ID))
	if err != nil {
		httpx.ServerError(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, d)
}

func (h *Handler) get(ctx context.Context, user auth.User, id string) (importantDate, error) {
	if !user.HasPair() || !httpx.IsUUID(id) {
		return importantDate{}, sql.ErrNoRows
	}
	return h.scan(h.DB.QueryRowContext(ctx,
		"SELECT "+columns+" FROM dates_importantdate WHERE id = $1 AND relationship_id = $2", id, user.RelationshipID))
}

func (h *Handler) write(w http.ResponseWriter, r *http.Request, d importantDate, err error) {
	switch {
	case errors.Is(err, sql.ErrNoRows):
		httpx.NotFound(w)
	case err != nil:
		httpx.ServerError(w, r, err)
	default:
		httpx.JSON(w, http.StatusOK, d)
	}
}

func (h *Handler) detail(w http.ResponseWriter, r *http.Request) {
	d, err := h.get(r.Context(), auth.FromContext(r.Context()), r.PathValue("id"))
	h.write(w, r, d, err)
}

func (h *Handler) update(partial bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.FromContext(r.Context())
		id := r.PathValue("id")
		if _, err := h.get(r.Context(), user, id); err != nil {
			h.write(w, r, importantDate{}, err)
			return
		}
		f, ok := readFields(w, r, partial)
		if !ok {
			return
		}
		d, err := h.scan(h.DB.QueryRowContext(r.Context(), `
			UPDATE dates_importantdate SET
				title = CASE WHEN $3 THEN $4 ELSE title END,
				date = CASE WHEN $5 THEN $6::date ELSE date END,
				emoji = CASE WHEN $7 THEN $8 ELSE emoji END,
				is_annually = CASE WHEN $9 THEN $10 ELSE is_annually END
			WHERE id = $1 AND relationship_id = $2
			RETURNING `+columns,
			id, user.RelationshipID, f.hasTitle, f.title, f.hasDate, f.date, f.hasEmoji, f.emoji, f.hasAnnual, f.annually))
		h.write(w, r, d, err)
	}
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	user := auth.FromContext(r.Context())
	id := r.PathValue("id")
	if !user.HasPair() || !httpx.IsUUID(id) {
		httpx.NotFound(w)
		return
	}
	res, err := h.DB.ExecContext(r.Context(),
		"DELETE FROM dates_importantdate WHERE id = $1 AND relationship_id = $2", id, user.RelationshipID)
	if err != nil {
		httpx.ServerError(w, r, err)
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		httpx.NotFound(w)
		return
	}
	httpx.NoContent(w)
}
