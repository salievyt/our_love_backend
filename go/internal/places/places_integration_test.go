//go:build integration

package places_test

import (
	"bytes"
	"image"
	"image/png"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/salievyt/our_love_backend/go/internal/media"
	"github.com/salievyt/our_love_backend/go/internal/testdb"
)

func pngBytes(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 4, 4))); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func ids(t *testing.T, res []any) map[string]bool {
	t.Helper()
	out := map[string]bool{}
	for _, item := range res {
		out[item.(map[string]any)["id"].(string)] = true
	}
	return out
}

func TestPlaces(t *testing.T) {
	e := testdb.New(t)
	alice, bob := e.CreateUser("alice"), e.CreateUser("bob")
	carol, dave := e.CreateUser("carol"), e.CreateUser("dave")
	eve := e.CreateUser("eve")
	e.CreatePair(alice, bob)
	e.CreatePair(carol, dave)

	park := e.Expect(e.Do("POST", "/api/v1/places/", alice, map[string]any{
		"title": "Парк Горького", "description": "Первое свидание", "latitude": 55.7298, "longitude": 37.6011,
	}), 201).JSON(t)
	parkID := park["id"].(string)
	if park["emoji"] != "❤️" || park["photo"] != nil || park["photo_url"] != nil || park["latitude"] != 55.7298 {
		t.Fatalf("created = %v", park)
	}
	cafe := e.Expect(e.DoMultipart("POST", "/api/v1/places/", bob, map[string]string{
		"title": "Кафе у моря", "emoji": "☕", "latitude": "43.58", "longitude": "39.72",
	}, nil), 201).JSON(t)
	cafeID := cafe["id"].(string)
	theirs := e.Expect(e.Do("POST", "/api/v1/places/", carol, map[string]any{
		"title": "Чужой парк", "latitude": 1, "longitude": 2,
	}), 201).JSON(t)["id"].(string)
	alone := e.Expect(e.Do("POST", "/api/v1/places/", eve, map[string]any{
		"title": "Только моё", "latitude": 0, "longitude": 0,
	}), 201).JSON(t)["id"].(string)

	t.Run("pair sees places of both partners", func(t *testing.T) {
		for _, u := range []string{alice, bob} {
			got := ids(t, e.Expect(e.Do("GET", "/api/v1/places/", u, nil), 200).Results(t))
			if len(got) != 2 || !got[parkID] || !got[cafeID] {
				t.Fatalf("pair sees %v", got)
			}
		}
		r := e.Expect(e.Do("GET", "/api/v1/places/"+cafeID+"/", alice, nil), 200).JSON(t)
		if r["title"] != "Кафе у моря" || r["emoji"] != "☕" {
			t.Fatalf("detail = %v", r)
		}
	})

	t.Run("other pair sees nothing of ours", func(t *testing.T) {
		got := ids(t, e.Expect(e.Do("GET", "/api/v1/places/", carol, nil), 200).Results(t))
		if len(got) != 1 || !got[theirs] {
			t.Fatalf("carol sees %v", got)
		}
		for _, u := range []string{carol, dave, eve} {
			e.Expect(e.Do("GET", "/api/v1/places/"+parkID+"/", u, nil), 404)
			e.Expect(e.Do("PATCH", "/api/v1/places/"+parkID+"/", u, map[string]any{"title": "x"}), 404)
			e.Expect(e.Do("DELETE", "/api/v1/places/"+parkID+"/", u, nil), 404)
		}
	})

	t.Run("user without pair sees only own places", func(t *testing.T) {
		got := ids(t, e.Expect(e.Do("GET", "/api/v1/places/", eve, nil), 200).Results(t))
		if len(got) != 1 || !got[alone] {
			t.Fatalf("eve sees %v", got)
		}
		e.Expect(e.Do("GET", "/api/v1/places/"+alone+"/", alice, nil), 404)
	})

	t.Run("search, emoji filter and ordering", func(t *testing.T) {
		cases := []struct {
			query string
			want  []string
		}{
			{"search=парк", []string{parkID}},      // регистр не важен
			{"search=СВИДАНИЕ", []string{parkID}},  // ищет и в description
			{"search=кафе моря", []string{cafeID}}, // все слова, в любом поле
			{"search=кафе свидание", nil},          // слова из разных мест — не найдено
			{"search=%", nil},                      // % ищется буквально
			{"emoji=☕", []string{cafeID}},
			{"ordering=title", []string{cafeID, parkID}},
			{"ordering=-title", []string{parkID, cafeID}},
			{"ordering=password", []string{cafeID, parkID}}, // неизвестное поле игнорируется, по умолчанию -created_at
		}
		for _, c := range cases {
			k, v, _ := strings.Cut(c.query, "=")
			res := e.Expect(e.Do("GET", "/api/v1/places/?"+k+"="+url.QueryEscape(v), alice, nil), 200).Results(t)
			var got []string
			for _, item := range res {
				got = append(got, item.(map[string]any)["id"].(string))
			}
			if strings.Join(got, ",") != strings.Join(c.want, ",") {
				t.Errorf("%s: got %v, want %v", c.query, got, c.want)
			}
		}
	})

	t.Run("validation", func(t *testing.T) {
		r := e.Expect(e.Do("POST", "/api/v1/places/", alice, map[string]any{"title": "", "latitude": "north"}), 400).JSON(t)
		if r["title"] == nil || r["latitude"] == nil || r["longitude"] == nil {
			t.Fatalf("errors = %v", r)
		}
		r = e.Expect(e.DoMultipart("POST", "/api/v1/places/", alice, map[string]string{
			"title": "x", "latitude": "1", "longitude": "2",
		}, map[string]testdb.File{"photo": {Name: "photo.jpg", Data: []byte("<html></html>")}}), 400).JSON(t)
		if r["photo"] == nil {
			t.Fatalf("errors = %v", r)
		}
	})

	t.Run("photo upload, signed link and clearing", func(t *testing.T) {
		data := pngBytes(t)
		r := e.Expect(e.DoMultipart("POST", "/api/v1/places/", alice, map[string]string{
			"title": "С фото", "latitude": "1.5", "longitude": "2",
		}, map[string]testdb.File{"photo": {Name: "photo.PNG", Data: data}}), 201).JSON(t)
		id := r["id"].(string)
		link, _ := r["photo_url"].(string)
		if link == "" || r["photo"] != link || r["longitude"] != 2.0 {
			t.Fatalf("created = %v", r)
		}
		u, err := url.Parse(link)
		if err != nil || u.Scheme != "http" || u.Host != "example.com" || !strings.HasPrefix(u.Path, "/media/places/") {
			t.Fatalf("link = %s", link)
		}
		name := strings.TrimPrefix(u.Path, "/media/")
		if !strings.HasSuffix(name, ".png") {
			t.Fatalf("name = %s", name)
		}
		// ссылку должен принять Django: та же подпись, тот же ключ
		if !media.NewSigner(testdb.Secret).Verify(name, u.Query().Get("exp"), u.Query().Get("sig"), time.Now()) {
			t.Fatalf("signature rejected: %s", link)
		}
		var stored string
		if err := e.DB.QueryRow("SELECT photo FROM places_place WHERE id = $1", id).Scan(&stored); err != nil || stored != name {
			t.Fatalf("db photo = %q, %v", stored, err)
		}
		saved, err := os.ReadFile(filepath.Join(e.MediaRoot, name))
		if err != nil || !bytes.Equal(saved, data) {
			t.Fatalf("file on disk: %v", err)
		}

		// партнёр видит фото; PUT без photo не трогает файл
		r = e.Expect(e.Do("PUT", "/api/v1/places/"+id+"/", bob, map[string]any{
			"title": "С фото!", "latitude": 3, "longitude": 4,
		}), 200).JSON(t)
		if r["photo_url"] == nil || r["title"] != "С фото!" || r["latitude"] != 3.0 {
			t.Fatalf("put = %v", r)
		}
		// пустое значение в форме очищает фото (DRF: "" → None)
		r = e.Expect(e.DoMultipart("PATCH", "/api/v1/places/"+id+"/", alice, map[string]string{"photo": ""}, nil), 200).JSON(t)
		if r["photo"] != nil || r["photo_url"] != nil || r["title"] != "С фото!" {
			t.Fatalf("cleared = %v", r)
		}
	})

	t.Run("update and delete inside the pair", func(t *testing.T) {
		r := e.Expect(e.Do("PATCH", "/api/v1/places/"+parkID+"/", bob, map[string]any{"emoji": "🌳"}), 200).JSON(t)
		if r["emoji"] != "🌳" || r["title"] != "Парк Горького" {
			t.Fatalf("patch = %v", r)
		}
		e.Expect(e.Do("PUT", "/api/v1/places/"+parkID+"/", bob, map[string]any{"title": "x"}), 400)
		e.Expect(e.Do("DELETE", "/api/v1/places/"+parkID+"/", bob, nil), 204)
		e.Expect(e.Do("GET", "/api/v1/places/"+parkID+"/", alice, nil), 404)
		e.Expect(e.Do("GET", "/api/v1/places/not-a-uuid/", alice, nil), 404)
	})
}
