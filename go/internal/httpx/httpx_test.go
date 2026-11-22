package httpx

import (
	"bytes"
	"mime/multipart"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestDateTimeMatchesDRF(t *testing.T) {
	msk, _ := time.LoadLocation("Europe/Moscow")
	tests := []struct {
		in   time.Time
		loc  *time.Location
		want string
	}{
		{time.Date(2026, 11, 22, 9, 32, 44, 458155000, time.UTC), msk, "2026-11-22T12:32:44.458155+03:00"},
		{time.Date(2026, 11, 22, 9, 32, 44, 0, time.UTC), msk, "2026-11-22T12:32:44+03:00"},
		{time.Date(2026, 11, 22, 9, 32, 44, 100000000, time.UTC), time.UTC, "2026-11-22T09:32:44.100000Z"},
	}
	for _, tt := range tests {
		if got := DateTime(tt.in, tt.loc); got != tt.want {
			t.Errorf("DateTime(%v) = %s, want %s", tt.in, got, tt.want)
		}
	}
}

func TestPayloadJSONAndForm(t *testing.T) {
	r := httptest.NewRequest("POST", "/", strings.NewReader(`{"mood": 4, "note": "  hi ", "date": "2026-11-22", "flag": "true"}`))
	r.Header.Set("Content-Type", "application/json")
	p, err := DecodePayload(r)
	if err != nil {
		t.Fatal(err)
	}
	if n, ok := p.Int("mood", true); !ok || n != 4 {
		t.Fatalf("mood = %d %v", n, ok)
	}
	if s, _ := p.String("note", false, true, 0); s != "hi" {
		t.Fatalf("note = %q", s)
	}
	if b, ok := p.Bool("flag", true); !ok || !b {
		t.Fatal("flag")
	}
	p.Date("missing", true)
	p.String("note", false, true, 1)
	if len(p.Errors["missing"]) != 1 || len(p.Errors["note"]) != 1 {
		t.Fatalf("errors = %v", p.Errors)
	}

	form := httptest.NewRequest("POST", "/", strings.NewReader("mood=5&date=bad"))
	form.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	p, err = DecodePayload(form)
	if err != nil {
		t.Fatal(err)
	}
	if n, ok := p.Int("mood", true); !ok || n != 5 {
		t.Fatalf("form mood = %d", n)
	}
	if _, ok := p.Date("date", true); ok || len(p.Errors["date"]) != 1 {
		t.Fatal("bad date accepted")
	}

	for _, body := range []string{"[1,2]", "{bad"} {
		r := httptest.NewRequest("POST", "/", strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		if _, err := DecodePayload(r); err == nil {
			t.Fatalf("body %q accepted", body)
		}
	}
}

func TestPayloadFile(t *testing.T) {
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	mw.WriteField("title", "Парк")
	mw.WriteField("cleared", "")
	fw, _ := mw.CreateFormFile("photo", "photo.jpg")
	fw.Write([]byte("data"))
	mw.Close()
	r := httptest.NewRequest("POST", "/", &body)
	r.Header.Set("Content-Type", mw.FormDataContentType())
	p, err := DecodePayload(r)
	if err != nil {
		t.Fatal(err)
	}
	if fh, ok := p.File("photo"); !ok || fh == nil || fh.Filename != "photo.jpg" || !p.Has("photo") {
		t.Fatalf("photo = %v %v", fh, ok)
	}
	if fh, ok := p.File("cleared"); !ok || fh != nil {
		t.Fatal(`"" in a form must clear the file`)
	}
	if _, ok := p.File("missing"); ok {
		t.Fatal("missing file reported as present")
	}
	if fh, ok := p.File("title"); ok || fh != nil || len(p.Errors["title"]) != 1 {
		t.Fatalf("text value accepted as file: %v", p.Errors)
	}

	r = httptest.NewRequest("PATCH", "/", strings.NewReader(`{"photo": null, "other": "x"}`))
	r.Header.Set("Content-Type", "application/json")
	if p, err = DecodePayload(r); err != nil {
		t.Fatal(err)
	}
	if fh, ok := p.File("photo"); !ok || fh != nil {
		t.Fatal("null must clear the file")
	}
	if _, ok := p.File("other"); ok || len(p.Errors["other"]) != 1 {
		t.Fatal("string accepted as file")
	}
}

func TestPaginationLinks(t *testing.T) {
	r := httptest.NewRequest("GET", "http://api/api/v1/moods/?page=2&mood=3", nil)
	r.Header.Set("X-Forwarded-Proto", "https")
	r.Header.Set("X-Forwarded-Host", "dev.example.com")
	if got := pageURL(r, 3); got != "https://dev.example.com/api/v1/moods/?mood=3&page=3" {
		t.Fatalf("next = %s", got)
	}
	if got := pageURL(r, 1); got != "https://dev.example.com/api/v1/moods/?mood=3" {
		t.Fatalf("previous = %s", got)
	}
}

func TestOrderBy(t *testing.T) {
	allowed := map[string]string{"date": "date", "mood": "mood"}
	r := httptest.NewRequest("GET", "/?ordering=-mood,password,date", nil)
	if got := OrderBy(r, allowed, "x"); got != "mood DESC, date" {
		t.Fatalf("got %q", got)
	}
	if got := OrderBy(httptest.NewRequest("GET", "/", nil), allowed, "date DESC"); got != "date DESC" {
		t.Fatalf("default %q", got)
	}
}
