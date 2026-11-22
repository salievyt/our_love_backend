// Package places — /api/v1/places/: места на карте отношений.
// Порт apps/places из Django. Видимость — как pair_q(user, "created_by"):
// места, созданные пользователем или его партнёром.
package places

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"mime/multipart"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/salievyt/our_love_backend/go/internal/auth"
	"github.com/salievyt/our_love_backend/go/internal/httpx"
	"github.com/salievyt/our_love_backend/go/internal/media"
)

const (
	defaultEmoji = "❤️"
	uploadPrefix = "places" // UploadTo("places")
	columns      = "id, title, description, emoji, latitude, longitude, photo, created_at"
	msgNullChar  = "Данные содержат запрещённый символ: ноль-байт"
)

// visible — условие видимости для пользователя в плейсхолдере ph. Повторяет
// pair_q(user, "created_by"): автор — сам пользователь или его партнёр по любой
// из двух сторон пары (relationship_as_partner1 / relationship_as_partner2).
func visible(ph string) string {
	return fmt.Sprintf(`(created_by_id = %[1]s OR created_by_id IN (
		SELECT partner1_id FROM accounts_relationship WHERE partner2_id = %[1]s
		UNION ALL
		SELECT partner2_id FROM accounts_relationship WHERE partner1_id = %[1]s))`, ph)
}

type Handler struct {
	DB     *sql.DB
	Loc    *time.Location
	Signer *media.Signer
	Store  media.Storage
}

// place — поля PlaceSerializer. photo, как и photo_url, — абсолютная
// подписанная ссылка (ImageField в DRF отдаёт URL) или null.
type place struct {
	ID          string      `json:"id"`
	Title       string      `json:"title"`
	Description string      `json:"description"`
	Emoji       string      `json:"emoji"`
	Latitude    httpx.Float `json:"latitude"`
	Longitude   httpx.Float `json:"longitude"`
	Photo       *string     `json:"photo"`
	PhotoURL    *string     `json:"photo_url"`
	CreatedAt   string      `json:"created_at"`
}

func (h *Handler) Register(mux *http.ServeMux, wrap func(http.Handler) http.Handler) {
	route := func(pattern string, fn http.HandlerFunc) { mux.Handle(pattern, wrap(fn)) }
	route("GET /api/v1/places/{$}", h.list)
	route("POST /api/v1/places/{$}", h.create)
	route("GET /api/v1/places/{id}/{$}", h.detail)
	route("PATCH /api/v1/places/{id}/{$}", h.update(true))
	route("PUT /api/v1/places/{id}/{$}", h.update(false))
	route("DELETE /api/v1/places/{id}/{$}", h.delete)
}

func (h *Handler) scan(r *http.Request, row interface{ Scan(...any) error }) (place, error) {
	var (
		p         place
		photo     sql.NullString
		createdAt time.Time
	)
	if err := row.Scan(&p.ID, &p.Title, &p.Description, &p.Emoji, &p.Latitude, &p.Longitude, &photo, &createdAt); err != nil {
		return place{}, err
	}
	p.Photo = h.Signer.URL(r, photo.String) // NULL и "" — файла нет
	p.PhotoURL = h.Signer.URL(r, photo.String)
	p.CreatedAt = httpx.DateTime(createdAt, h.Loc)
	return p, nil
}

func (h *Handler) query(r *http.Request, query string, args ...any) ([]place, error) {
	rows, err := h.DB.QueryContext(r.Context(), query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []place{}
	for rows.Next() {
		p, err := h.scan(r, rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// ordering — поля PlaceSerializer, по которым OrderingFilter разрешает сортировку
// (photo_url — метод, сортировать по нему нельзя).
var ordering = map[string]string{
	"id": "id", "title": "title", "description": "description", "emoji": "emoji",
	"latitude": "latitude", "longitude": "longitude", "photo": "photo", "created_at": "created_at",
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	page, ok := httpx.PageNumber(r)
	if !ok {
		httpx.Error(w, http.StatusNotFound, "Неправильная страница.")
		return
	}
	user := auth.FromContext(r.Context())
	q := r.URL.Query()
	where, args := []string{visible("$1")}, []any{user.ID}
	arg := func(v any) string {
		args = append(args, v)
		return fmt.Sprintf("$%d", len(args))
	}

	if v := q.Get("emoji"); v != "" {
		if strings.ContainsRune(v, 0) {
			httpx.FieldErrors{"emoji": {msgNullChar}}.Write(w)
			return
		}
		where = append(where, "emoji = "+arg(v))
	}
	search := q.Get("search")
	if strings.ContainsRune(search, 0) {
		httpx.JSON(w, http.StatusBadRequest, []string{msgNullChar}) // ValidationError списком, как в DRF
		return
	}
	// SearchFilter: каждое слово должно найтись хотя бы в одном из полей (icontains)
	for _, term := range SearchTerms(search) {
		ph := arg("%" + escapeLike(term) + "%")
		where = append(where, fmt.Sprintf("(title ILIKE %[1]s OR description ILIKE %[1]s)", ph))
	}
	cond := strings.Join(where, " AND ")

	var total int
	if err := h.DB.QueryRowContext(r.Context(), "SELECT count(*) FROM places_place WHERE "+cond, args...).Scan(&total); err != nil {
		httpx.ServerError(w, r, err)
		return
	}
	order := httpx.OrderBy(r, ordering, "created_at DESC")
	results, err := h.query(r, fmt.Sprintf(
		"SELECT %s FROM places_place WHERE %s ORDER BY %s, id LIMIT %d OFFSET %d",
		columns, cond, order, httpx.PageSize, (page-1)*httpx.PageSize), args...)
	if err != nil {
		httpx.ServerError(w, r, err)
		return
	}
	httpx.WritePage(w, r, page, total, results)
}

// escapeLike экранирует спецсимволы LIKE, как Django для icontains.
func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}

// smartSplit — django.utils.text.smart_split: пробелы делят строку,
// но не внутри кавычек.
var smartSplit = regexp.MustCompile(`[^\s'"]*(?:(?:"(?:[^"\\]|\\.)*"|'(?:[^'\\]|\\.)*')[^\s'"]*)+|\S+`)

// SearchTerms — rest_framework.filters.search_smart_split: слова через пробел
// или запятую, фразы в кавычках не делятся.
func SearchTerms(s string) []string {
	var terms []string
	for _, tok := range smartSplit.FindAllString(s, -1) {
		tok = strings.Trim(tok, ",")
		if len(tok) >= 2 && (tok[0] == '"' || tok[0] == '\'') && tok[len(tok)-1] == tok[0] {
			q := string(tok[0])
			terms = append(terms, strings.NewReplacer(`\`+q, q, `\\`, `\`).Replace(tok[1:len(tok)-1]))
			continue
		}
		for _, sub := range strings.Split(tok, ",") {
			if sub != "" {
				terms = append(terms, strings.TrimSpace(sub))
			}
		}
	}
	return terms
}

// fields — PlaceCreateSerializer; has* — поле есть в запросе.
type fields struct {
	title, description, emoji           string
	latitude, longitude                 float64
	photo                               *multipart.FileHeader // nil при hasPhoto — очистить
	hasTitle, hasDescription            bool
	hasEmoji, hasLatitude, hasLongitude bool
	hasPhoto                            bool
}

func readFields(w http.ResponseWriter, r *http.Request, partial bool) (fields, bool) {
	p, err := httpx.DecodePayload(r)
	if err != nil {
		httpx.BadRequest(w, err)
		return fields{}, false
	}
	var f fields
	f.title, f.hasTitle = p.String("title", !partial, false, 200)
	f.description, f.hasDescription = p.String("description", false, true, 0)
	f.emoji, f.hasEmoji = p.String("emoji", false, false, 10)
	f.latitude, f.hasLatitude = p.Float("latitude", !partial)
	f.longitude, f.hasLongitude = p.Float("longitude", !partial)
	f.photo, f.hasPhoto = p.File("photo")
	if f.photo != nil {
		msgs, err := media.ValidateFile(f.photo)
		if err != nil {
			httpx.ServerError(w, r, err)
			return fields{}, false
		}
		for _, m := range msgs {
			p.Errors.Add("photo", m)
		}
	}
	if len(p.Errors) > 0 {
		p.Errors.Write(w)
		return fields{}, false
	}
	return f, true
}

// savePhoto кладёт новый файл в MEDIA_ROOT; "" — файла нет или его очищают.
func (h *Handler) savePhoto(f fields) (string, error) {
	if f.photo == nil {
		return "", nil
	}
	return h.Store.Save(uploadPrefix, f.photo)
}

// discard удаляет только что сохранённый файл, если запись в базу не состоялась.
func (h *Handler) discard(r *http.Request, name string) {
	if name == "" {
		return
	}
	if err := h.Store.Remove(name); err != nil {
		slog.WarnContext(r.Context(), "orphan media file", "name", name, "err", err)
	}
}

func nullable(name string) sql.NullString { return sql.NullString{String: name, Valid: name != ""} }

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	user := auth.FromContext(r.Context())
	f, ok := readFields(w, r, false)
	if !ok {
		return
	}
	if !f.hasEmoji {
		f.emoji = defaultEmoji
	}
	photo, err := h.savePhoto(f)
	if err != nil {
		httpx.ServerError(w, r, err)
		return
	}
	p, err := h.scan(r, h.DB.QueryRowContext(r.Context(), `
		INSERT INTO places_place (id, title, description, emoji, latitude, longitude, photo, created_by_id, created_at)
		VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, now())
		RETURNING `+columns, f.title, f.description, f.emoji, f.latitude, f.longitude, nullable(photo), user.ID))
	if err != nil {
		h.discard(r, photo)
		httpx.ServerError(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, p)
}

func (h *Handler) get(r *http.Request, user auth.User, id string) (place, error) {
	if !httpx.IsUUID(id) {
		return place{}, sql.ErrNoRows
	}
	return h.scan(r, h.DB.QueryRowContext(r.Context(),
		"SELECT "+columns+" FROM places_place WHERE id = $1 AND "+visible("$2"), id, user.ID))
}

func (h *Handler) write(w http.ResponseWriter, r *http.Request, p place, err error) {
	switch {
	case errors.Is(err, sql.ErrNoRows):
		httpx.NotFound(w)
	case err != nil:
		httpx.ServerError(w, r, err)
	default:
		httpx.JSON(w, http.StatusOK, p)
	}
}

func (h *Handler) detail(w http.ResponseWriter, r *http.Request) {
	p, err := h.get(r, auth.FromContext(r.Context()), r.PathValue("id"))
	h.write(w, r, p, err)
}

func (h *Handler) update(partial bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.FromContext(r.Context())
		id := r.PathValue("id")
		// 404 раньше 400, как get_object() в DRF
		if _, err := h.get(r, user, id); err != nil {
			h.write(w, r, place{}, err)
			return
		}
		f, ok := readFields(w, r, partial)
		if !ok {
			return
		}
		photo, err := h.savePhoto(f)
		if err != nil {
			httpx.ServerError(w, r, err)
			return
		}
		// Старый файл не удаляем — Django тоже оставляет его на диске.
		p, err := h.scan(r, h.DB.QueryRowContext(r.Context(), `
			UPDATE places_place SET
				title = CASE WHEN $2 THEN $3 ELSE title END,
				description = CASE WHEN $4 THEN $5 ELSE description END,
				emoji = CASE WHEN $6 THEN $7 ELSE emoji END,
				latitude = CASE WHEN $8 THEN $9::double precision ELSE latitude END,
				longitude = CASE WHEN $10 THEN $11::double precision ELSE longitude END,
				photo = CASE WHEN $12 THEN $13::varchar ELSE photo END
			WHERE id = $1 AND `+visible("$14")+`
			RETURNING `+columns,
			id, f.hasTitle, f.title, f.hasDescription, f.description, f.hasEmoji, f.emoji,
			f.hasLatitude, f.latitude, f.hasLongitude, f.longitude, f.hasPhoto, nullable(photo), user.ID))
		if err != nil {
			h.discard(r, photo)
		}
		h.write(w, r, p, err)
	}
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	user := auth.FromContext(r.Context())
	id := r.PathValue("id")
	if !httpx.IsUUID(id) {
		httpx.NotFound(w)
		return
	}
	res, err := h.DB.ExecContext(r.Context(),
		"DELETE FROM places_place WHERE id = $1 AND "+visible("$2"), id, user.ID)
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
