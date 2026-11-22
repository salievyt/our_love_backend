//go:build integration

package dates_test

import (
	"testing"

	"github.com/salievyt/our_love_backend/go/internal/testdb"
)

func TestDates(t *testing.T) {
	e := testdb.New(t)
	alice, bob := e.CreateUser("alice"), e.CreateUser("bob")
	carol, dave := e.CreateUser("carol"), e.CreateUser("dave")
	eve := e.CreateUser("eve")
	pair := e.CreatePair(alice, bob)
	e.CreatePair(carol, dave)

	created := e.Expect(e.Do("POST", "/api/v1/dates/", alice, map[string]any{"title": "Годовщина", "date": "2024-02-29"}), 201).JSON(t)
	id := created["id"].(string)
	if created["relationship"] != pair || created["emoji"] != "❤️" || created["is_annually"] != true {
		t.Fatalf("created = %v", created)
	}

	t.Run("partner sees the date", func(t *testing.T) {
		r := e.Expect(e.Do("GET", "/api/v1/dates/"+id+"/", bob, nil), 200).JSON(t)
		if r["title"] != "Годовщина" {
			t.Fatalf("detail = %v", r)
		}
	})

	t.Run("other pair and users without pair see nothing", func(t *testing.T) {
		for _, u := range []string{carol, eve} {
			if n := len(e.Expect(e.Do("GET", "/api/v1/dates/", u, nil), 200).Results(t)); n != 0 {
				t.Fatalf("stranger sees %d dates", n)
			}
			e.Expect(e.Do("GET", "/api/v1/dates/"+id+"/", u, nil), 404)
			e.Expect(e.Do("PATCH", "/api/v1/dates/"+id+"/", u, map[string]any{"title": "x"}), 404)
			e.Expect(e.Do("DELETE", "/api/v1/dates/"+id+"/", u, nil), 404)
		}
	})

	t.Run("create without pair is 400, not 500", func(t *testing.T) {
		e.Expect(e.Do("POST", "/api/v1/dates/", eve, map[string]any{"title": "x", "date": "2026-01-01"}), 400)
	})

	t.Run("validation", func(t *testing.T) {
		r := e.Expect(e.Do("POST", "/api/v1/dates/", alice, map[string]any{"title": "", "emoji": "12345678901"}), 400).JSON(t)
		if r["title"] == nil || r["date"] == nil || r["emoji"] == nil {
			t.Fatalf("errors = %v", r)
		}
	})

	t.Run("upcoming sorts by days until and returns at most five", func(t *testing.T) {
		for _, title := range []string{"a", "b", "c", "d", "e"} {
			e.Expect(e.Do("POST", "/api/v1/dates/", bob, map[string]any{"title": title, "date": "2020-01-01", "is_annually": false}), 201)
		}
		res := e.Expect(e.Do("GET", "/api/v1/dates/upcoming/", alice, nil), 200).Results(t)
		if len(res) != 5 {
			t.Fatalf("upcoming = %d", len(res))
		}
		prev := -1 << 31
		for _, item := range res {
			d := int(item.(map[string]any)["days_until"].(float64))
			if d < prev {
				t.Fatalf("not sorted: %v", res)
			}
			prev = d
		}
		if n := len(e.Expect(e.Do("GET", "/api/v1/dates/?is_annually=true", alice, nil), 200).Results(t)); n != 1 {
			t.Fatalf("is_annually filter = %d", n)
		}
	})

	t.Run("update and delete inside the pair", func(t *testing.T) {
		r := e.Expect(e.Do("PATCH", "/api/v1/dates/"+id+"/", bob, map[string]any{"emoji": "💍"}), 200).JSON(t)
		if r["emoji"] != "💍" || r["title"] != "Годовщина" {
			t.Fatalf("patch = %v", r)
		}
		e.Expect(e.Do("DELETE", "/api/v1/dates/"+id+"/", bob, nil), 204)
		e.Expect(e.Do("GET", "/api/v1/dates/"+id+"/", alice, nil), 404)
	})
}
