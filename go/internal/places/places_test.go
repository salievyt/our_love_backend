package places

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/salievyt/our_love_backend/go/internal/auth"
	"github.com/salievyt/our_love_backend/go/internal/media"
)

// Эталон — rest_framework.filters.search_smart_split (DRF 3.18).
func TestSearchTermsMatchDRF(t *testing.T) {
	tests := []struct {
		in   string
		want []string
	}{
		{"парк", []string{"парк"}},
		{"  кафе   у моря ", []string{"кафе", "у", "моря"}},
		{"a,b, c", []string{"a", "b", "c"}},
		{`"Red Square" moscow`, []string{"Red Square", "moscow"}},
		{`'it\'s' x`, []string{"it's", "x"}},
		{`a"b c"d`, []string{`a"b c"d`}},
		{",,", nil},
		{`"unclosed phrase`, []string{`"unclosed`, "phrase"}},
		{`"a,b",c`, []string{`"a`, `b"`, "c"}},
		{`%_\`, []string{`%_\`}},
		{"", nil},
	}
	for _, tt := range tests {
		got := SearchTerms(tt.in)
		if strings.Join(got, "\x1f") != strings.Join(tt.want, "\x1f") || len(got) != len(tt.want) {
			t.Errorf("SearchTerms(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

// Невалидный запрос отклоняется до базы (DB = nil) и ничего не пишет в MEDIA_ROOT.
func TestCreateValidationTouchesNothing(t *testing.T) {
	root := t.TempDir()
	h := &Handler{Signer: media.NewSigner("k"), Store: media.Storage{Root: root}}

	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	mw.WriteField("latitude", "north")
	mw.WriteField("emoji", "12345678901")
	fw, _ := mw.CreateFormFile("photo", "photo.jpg")
	fw.Write([]byte("<html>not an image</html>"))
	mw.Close()
	r := httptest.NewRequest("POST", "/api/v1/places/", &body)
	r.Header.Set("Content-Type", mw.FormDataContentType())
	r = r.WithContext(auth.WithUser(r.Context(), auth.User{ID: "test-user"})) // до базы id не доходит
	w := httptest.NewRecorder()
	h.create(w, r)

	if w.Code != 400 {
		t.Fatalf("status = %d: %s", w.Code, w.Body)
	}
	var errs map[string][]string
	if err := json.Unmarshal(w.Body.Bytes(), &errs); err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"title":     "Обязательное поле.",
		"latitude":  "Требуется численное значение.",
		"longitude": "Обязательное поле.",
		"emoji":     "Убедитесь, что это значение содержит не более 10 символов.",
		"photo":     "Загрузите правильное изображение. Файл, который вы загрузили, поврежден или не является изображением.",
	}
	for field, msg := range want {
		if len(errs[field]) != 1 || errs[field][0] != msg {
			t.Errorf("%s = %q, want %q", field, errs[field], msg)
		}
	}
	if entries, _ := os.ReadDir(root); len(entries) != 0 {
		t.Fatalf("MEDIA_ROOT is not empty: %v", entries)
	}
}

func TestEscapeLike(t *testing.T) {
	if got := escapeLike(`50%_off\`); got != `50\%\_off\\` {
		t.Fatalf("escapeLike = %s", got)
	}
}
