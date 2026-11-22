// Package moods — /api/v1/moods/: личное настроение пользователя.
// Порт apps/moods из Django; видимость — только свои записи (user = request.user).
package moods

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/salievyt/our_love_backend/go/internal/auth"
	"github.com/salievyt/our_love_backend/go/internal/httpx"
)

type Handler struct {
	DB  *sql.DB
	Loc *time.Location
}

type entry struct {
	ID        string `json:"id"`
	User      string `json:"user"`
	Mood      int    `json:"mood"`
	Note      string `json:"note"`
	Date      string `json:"date"`
	CreatedAt string `json:"created_at"`
}

const columns = "id, user_id, mood, note, date, created_at"

func (h *Handler) Register(mux *http.ServeMux, wrap func(http.Handler) http.Handler) {
	route := func(pattern string, fn http.HandlerFunc) { mux.Handle(pattern, wrap(fn)) }
	route("GET /api/v1/moods/{$}", h.list)
	route("POST /api/v1/moods/{$}", h.create)
	route("GET /api/v1/moods/today/{$}", h.today)
	route("PATCH /api/v1/moods/today/{$}", h.updateToday(true))
	route("PUT /api/v1/moods/today/{$}", h.updateToday(false))
	route("GET /api/v1/moods/stats/{$}", h.stats)
	route("GET /api/v1/moods/{id}/{$}", h.detail)
	route("PATCH /api/v1/moods/{id}/{$}", h.update(true))
	route("PUT /api/v1/moods/{id}/{$}", h.update(false))
	route("DELETE /api/v1/moods/{id}/{$}", h.delete)
}

func (h *Handler) scan(row interface{ Scan(...any) error }) (entry, error) {
	var (
		e         entry
		date      time.Time
		createdAt time.Time
	)
	if err := row.Scan(&e.ID, &e.User, &e.Mood, &e.Note, &date, &createdAt); err != nil {
		return entry{}, err
	}
	e.Date = httpx.Date(date)
	e.CreatedAt = httpx.DateTime(createdAt, h.Loc)
	return e, nil
}

var ordering = map[string]string{
	"id": "id", "user": "user_id", "mood": "mood", "note": "note", "date": "date", "created_at": "created_at",
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	page, ok := httpx.PageNumber(r)
	if !ok {
		httpx.Error(w, http.StatusNotFound, "Неправильная страница.")
		return
	}
	user := auth.FromContext(r.Context())
	where, args := []string{"user_id = $1"}, []any{user.ID}

	q := r.URL.Query()
	if v := q.Get("date"); v != "" {
		d, err := httpx.ParseDate(v)
		if err != nil {
			httpx.FieldErrors{"date": {"Введите правильную дату."}}.Write(w)
			return
		}
		args = append(args, d)
		where = append(where, fmt.Sprintf("date = $%d", len(args)))
	}
	if v := q.Get("mood"); v != "" {
		args = append(args, v)
		if !validMoodString(v) {
			httpx.FieldErrors{"mood": {fmt.Sprintf("Выберите корректный вариант. %s нет среди допустимых значений.", v)}}.Write(w)
			return
		}
		where = append(where, fmt.Sprintf("mood = $%d", len(args)))
	}
	cond := strings.Join(where, " AND ")

	var total int
	if err := h.DB.QueryRowContext(r.Context(), "SELECT count(*) FROM moods_moodentry WHERE "+cond, args...).Scan(&total); err != nil {
		httpx.ServerError(w, r, err)
		return
	}
	order := httpx.OrderBy(r, ordering, "date DESC")
	rows, err := h.DB.QueryContext(r.Context(), fmt.Sprintf(
		"SELECT %s FROM moods_moodentry WHERE %s ORDER BY %s, id LIMIT %d OFFSET %d",
		columns, cond, order, httpx.PageSize, (page-1)*httpx.PageSize), args...)
	if err != nil {
		httpx.ServerError(w, r, err)
		return
	}
	defer rows.Close()
	results := []entry{}
	for rows.Next() {
		e, err := h.scan(rows)
		if err != nil {
			httpx.ServerError(w, r, err)
			return
		}
		results = append(results, e)
	}
	if err := rows.Err(); err != nil {
		httpx.ServerError(w, r, err)
		return
	}
	httpx.WritePage(w, r, page, total, results)
}

func validMoodString(v string) bool { return len(v) == 1 && v[0] >= '1' && v[0] <= '5' }

// fields — разобранные поля mood/note/date. partial: отсутствующие поля не трогаем.
type fields struct {
	mood    int
	note    string
	date    time.Time
	hasMood bool
	hasNote bool
	hasDate bool
}

func readFields(w http.ResponseWriter, r *http.Request, partial bool) (fields, bool) {
	p, err := httpx.DecodePayload(r)
	if err != nil {
		httpx.BadRequest(w, err)
		return fields{}, false
	}
	var f fields
	f.mood, f.hasMood = p.Int("mood", !partial)
	if f.hasMood && (f.mood < 1 || f.mood > 5) {
		p.Errors.Add("mood", fmt.Sprintf("Значения %d нет среди допустимых вариантов.", f.mood))
	}
	f.note, f.hasNote = p.String("note", false, true, 0)
	f.date, f.hasDate = p.Date("date", !partial)
	if len(p.Errors) > 0 {
		p.Errors.Write(w)
		return fields{}, false
	}
	return f, true
}

var errDuplicateDate = errors.New("duplicate date")

func (h *Handler) dateTaken(ctx context.Context, userID string, date time.Time, exceptID string) (bool, error) {
	var exists bool
	err := h.DB.QueryRowContext(ctx,
		"SELECT EXISTS (SELECT 1 FROM moods_moodentry WHERE user_id = $1 AND date = $2 AND id::text <> $3)",
		userID, date, exceptID).Scan(&exists)
	return exists, err
}

func writeDuplicate(w http.ResponseWriter) {
	httpx.FieldErrors{"date": {"Настроение на эту дату уже отмечено."}}.Write(w)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	f, ok := readFields(w, r, false)
	if !ok {
		return
	}
	user := auth.FromContext(r.Context())
	if taken, err := h.dateTaken(r.Context(), user.ID, f.date, ""); err != nil {
		httpx.ServerError(w, r, err)
		return
	} else if taken {
		writeDuplicate(w)
		return
	}
	e, err := h.scan(h.DB.QueryRowContext(r.Context(), `
		INSERT INTO moods_moodentry (id, user_id, mood, note, date, created_at)
		VALUES (gen_random_uuid(), $1, $2, $3, $4, now())
		RETURNING `+columns, user.ID, f.mood, f.note, f.date))
	switch {
	case httpx.IsUniqueViolation(err):
		writeDuplicate(w)
	case err != nil:
		httpx.ServerError(w, r, err)
	default:
		httpx.JSON(w, http.StatusCreated, e)
	}
}

func (h *Handler) get(ctx context.Context, userID, id string) (entry, error) {
	if !httpx.IsUUID(id) {
		return entry{}, sql.ErrNoRows
	}
	return h.scan(h.DB.QueryRowContext(ctx,
		"SELECT "+columns+" FROM moods_moodentry WHERE id = $1 AND user_id = $2", id, userID))
}

func (h *Handler) detail(w http.ResponseWriter, r *http.Request) {
	e, err := h.get(r.Context(), auth.FromContext(r.Context()).ID, r.PathValue("id"))
	h.writeEntry(w, r, e, err)
}

func (h *Handler) writeEntry(w http.ResponseWriter, r *http.Request, e entry, err error) {
	switch {
	case errors.Is(err, sql.ErrNoRows):
		httpx.NotFound(w)
	case errors.Is(err, errDuplicateDate), httpx.IsUniqueViolation(err):
		writeDuplicate(w)
	case err != nil:
		httpx.ServerError(w, r, err)
	default:
		httpx.JSON(w, http.StatusOK, e)
	}
}

func (h *Handler) apply(ctx context.Context, userID, id string, f fields) (entry, error) {
	if f.hasDate {
		taken, err := h.dateTaken(ctx, userID, f.date, id)
		if err != nil {
			return entry{}, err
		}
		if taken {
			return entry{}, errDuplicateDate
		}
	}
	return h.scan(h.DB.QueryRowContext(ctx, `
		UPDATE moods_moodentry SET
			mood = CASE WHEN $3 THEN $4 ELSE mood END,
			note = CASE WHEN $5 THEN $6 ELSE note END,
			date = CASE WHEN $7 THEN $8::date ELSE date END
		WHERE id = $1 AND user_id = $2
		RETURNING `+columns,
		id, userID, f.hasMood, f.mood, f.hasNote, f.note, f.hasDate, f.date))
}

func (h *Handler) update(partial bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.FromContext(r.Context())
		id := r.PathValue("id")
		if _, err := h.get(r.Context(), user.ID, id); err != nil {
			h.writeEntry(w, r, entry{}, err)
			return
		}
		f, ok := readFields(w, r, partial)
		if !ok {
			return
		}
		e, err := h.apply(r.Context(), user.ID, id, f)
		h.writeEntry(w, r, e, err)
	}
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !httpx.IsUUID(id) {
		httpx.NotFound(w)
		return
	}
	res, err := h.DB.ExecContext(r.Context(),
		"DELETE FROM moods_moodentry WHERE id = $1 AND user_id = $2", id, auth.FromContext(r.Context()).ID)
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

// todayEntry — get_or_create(user, date=today, defaults={"mood": 3}).
func (h *Handler) todayEntry(ctx context.Context, userID string) (entry, error) {
	today := httpx.Today(h.Loc)
	if _, err := h.DB.ExecContext(ctx, `
		INSERT INTO moods_moodentry (id, user_id, mood, note, date, created_at)
		VALUES (gen_random_uuid(), $1, 3, '', $2, now())
		ON CONFLICT (user_id, date) DO NOTHING`, userID, today); err != nil {
		return entry{}, err
	}
	return h.scan(h.DB.QueryRowContext(ctx,
		"SELECT "+columns+" FROM moods_moodentry WHERE user_id = $1 AND date = $2", userID, today))
}

func (h *Handler) today(w http.ResponseWriter, r *http.Request) {
	e, err := h.todayEntry(r.Context(), auth.FromContext(r.Context()).ID)
	h.writeEntry(w, r, e, err)
}

func (h *Handler) updateToday(partial bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.FromContext(r.Context())
		current, err := h.todayEntry(r.Context(), user.ID)
		if err != nil {
			httpx.ServerError(w, r, err)
			return
		}
		f, ok := readFields(w, r, partial)
		if !ok {
			return
		}
		e, err := h.apply(r.Context(), user.ID, current.ID, f)
		h.writeEntry(w, r, e, err)
	}
}

func (h *Handler) stats(w http.ResponseWriter, r *http.Request) {
	userID := auth.FromContext(r.Context()).ID
	rows, err := h.DB.QueryContext(r.Context(),
		"SELECT mood, count(*) FROM moods_moodentry WHERE user_id = $1 GROUP BY mood", userID)
	if err != nil {
		httpx.ServerError(w, r, err)
		return
	}
	defer rows.Close()
	counts := map[int]int{}
	for rows.Next() {
		var mood, n int
		if err := rows.Scan(&mood, &n); err != nil {
			httpx.ServerError(w, r, err)
			return
		}
		counts[mood] = n
	}
	if err := rows.Err(); err != nil {
		httpx.ServerError(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, buildStats(counts))
}

type statsResponse struct {
	Total    int            `json:"total"`
	Average  float64        `json:"average"`
	GoodDays int            `json:"good_days"`
	BadDays  int            `json:"bad_days"`
	ByMood   map[string]int `json:"by_mood"`
}

func buildStats(counts map[int]int) statsResponse {
	s := statsResponse{ByMood: map[string]int{}}
	sum := 0
	for mood, n := range counts {
		s.Total += n
		sum += mood * n
		if mood >= 4 {
			s.GoodDays += n
		}
		if mood <= 2 {
			s.BadDays += n
		}
	}
	if s.Total == 0 {
		return s
	}
	for i := 1; i <= 5; i++ {
		s.ByMood[fmt.Sprint(i)] = counts[i]
	}
	s.Average = math.Round(float64(sum)/float64(s.Total)*10) / 10
	return s
}
