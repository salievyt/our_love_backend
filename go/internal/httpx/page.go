package httpx

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// PageSize совпадает с REST_FRAMEWORK["PAGE_SIZE"].
const PageSize = 20

// Page — ответ PageNumberPagination: {"count", "next", "previous", "results"}.
type Page struct {
	Count    int     `json:"count"`
	Next     *string `json:"next"`
	Previous *string `json:"previous"`
	Results  any     `json:"results"`
}

// PageNumber читает ?page=. ok=false — невалидная страница (DRF отвечает 404).
func PageNumber(r *http.Request) (int, bool) {
	raw := r.URL.Query().Get("page")
	if raw == "" || raw == "last" {
		return 1, raw == ""
	}
	n, err := strconv.Atoi(raw)
	return n, err == nil && n >= 1
}

// WritePage пишет страницу; total — общее число записей, page — номер страницы.
func WritePage(w http.ResponseWriter, r *http.Request, page, total int, results any) {
	if pages := (total + PageSize - 1) / PageSize; page > max(pages, 1) {
		Error(w, http.StatusNotFound, "Неправильная страница.")
		return
	}
	p := Page{Count: total, Results: results}
	if page*PageSize < total {
		next := pageURL(r, page+1)
		p.Next = &next
	}
	if page > 1 {
		prev := pageURL(r, page-1)
		p.Previous = &prev
	}
	JSON(w, http.StatusOK, p)
}

func pageURL(r *http.Request, page int) string {
	q := r.URL.Query()
	if page == 1 {
		q.Del("page")
	} else {
		q.Set("page", strconv.Itoa(page))
	}
	u := url.URL{Scheme: scheme(r), Host: host(r), Path: r.URL.Path, RawQuery: q.Encode()}
	return u.String()
}

// BaseURL — "схема://хост" запроса, как request.build_absolute_uri("/")[:-1] в Django.
func BaseURL(r *http.Request) string { return scheme(r) + "://" + host(r) }

// Как Django с USE_X_FORWARDED_HOST и SECURE_PROXY_SSL_HEADER: доверяем заголовкам nginx.
func scheme(r *http.Request) string {
	if p := r.Header.Get("X-Forwarded-Proto"); p == "https" {
		return "https"
	}
	if r.TLS != nil {
		return "https"
	}
	return "http"
}

func host(r *http.Request) string {
	if h := r.Header.Get("X-Forwarded-Host"); h != "" {
		return strings.TrimSpace(strings.Split(h, ",")[0])
	}
	return r.Host
}

// OrderBy разбирает ?ordering=a,-b по белому списку полей (как OrderingFilter).
// allowed: имя поля API → колонка SQL. Если ничего не подошло — def.
func OrderBy(r *http.Request, allowed map[string]string, def string) string {
	var parts []string
	for _, f := range strings.Split(r.URL.Query().Get("ordering"), ",") {
		f = strings.TrimSpace(f)
		desc := strings.HasPrefix(f, "-")
		col, ok := allowed[strings.TrimPrefix(f, "-")]
		if !ok {
			continue
		}
		if desc {
			col += " DESC"
		}
		parts = append(parts, col)
	}
	if len(parts) == 0 {
		return def
	}
	return strings.Join(parts, ", ")
}
