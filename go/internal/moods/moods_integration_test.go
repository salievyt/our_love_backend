//go:build integration

package moods_test

import (
	"testing"

	"github.com/salievyt/our_love_backend/go/internal/testdb"
)

func TestMoods(t *testing.T) {
	e := testdb.New(t)
	alice, bob := e.CreateUser("alice"), e.CreateUser("bob")
	e.CreatePair(alice, bob)

	t.Run("requires auth", func(t *testing.T) {
		e.Expect(e.Do("GET", "/api/v1/moods/", "", nil), 401)
	})

	created := e.Expect(e.Do("POST", "/api/v1/moods/", alice, map[string]any{"mood": 5, "note": "хорошо", "date": "2026-01-10"}), 201).JSON(t)
	id := created["id"].(string)
	if created["user"] != alice || created["mood"].(float64) != 5 || created["date"] != "2026-01-10" {
		t.Fatalf("created = %v", created)
	}

	t.Run("validation", func(t *testing.T) {
		r := e.Expect(e.Do("POST", "/api/v1/moods/", alice, map[string]any{"mood": 9}), 400).JSON(t)
		if r["mood"] == nil || r["date"] == nil {
			t.Fatalf("errors = %v", r)
		}
		e.Expect(e.Do("POST", "/api/v1/moods/", alice, map[string]any{"mood": 3, "date": "2026-01-10"}), 400)
	})

	t.Run("moods are personal even inside a pair", func(t *testing.T) {
		if n := len(e.Expect(e.Do("GET", "/api/v1/moods/", bob, nil), 200).Results(t)); n != 0 {
			t.Fatalf("partner sees %d moods", n)
		}
		e.Expect(e.Do("GET", "/api/v1/moods/"+id+"/", bob, nil), 404)
		e.Expect(e.Do("PATCH", "/api/v1/moods/"+id+"/", bob, map[string]any{"mood": 1}), 404)
		e.Expect(e.Do("DELETE", "/api/v1/moods/"+id+"/", bob, nil), 404)
	})

	t.Run("list, filter, ordering", func(t *testing.T) {
		e.Expect(e.Do("POST", "/api/v1/moods/", alice, map[string]any{"mood": 2, "date": "2026-01-11"}), 201)
		res := e.Expect(e.Do("GET", "/api/v1/moods/", alice, nil), 200).Results(t)
		if len(res) != 2 || res[0].(map[string]any)["date"] != "2026-01-11" {
			t.Fatalf("default ordering -date: %v", res)
		}
		res = e.Expect(e.Do("GET", "/api/v1/moods/?ordering=date", alice, nil), 200).Results(t)
		if res[0].(map[string]any)["date"] != "2026-01-10" {
			t.Fatalf("ordering=date: %v", res)
		}
		if n := len(e.Expect(e.Do("GET", "/api/v1/moods/?mood=5", alice, nil), 200).Results(t)); n != 1 {
			t.Fatalf("mood filter: %d", n)
		}
		if n := len(e.Expect(e.Do("GET", "/api/v1/moods/?date=2026-01-11", alice, nil), 200).Results(t)); n != 1 {
			t.Fatalf("date filter: %d", n)
		}
	})

	t.Run("update and delete", func(t *testing.T) {
		r := e.Expect(e.Do("PATCH", "/api/v1/moods/"+id+"/", alice, map[string]any{"note": "ещё лучше"}), 200).JSON(t)
		if r["note"] != "ещё лучше" || r["mood"].(float64) != 5 {
			t.Fatalf("patch = %v", r)
		}
		e.Expect(e.Do("PUT", "/api/v1/moods/"+id+"/", alice, map[string]any{"mood": 4}), 400) // PUT требует date
		e.Expect(e.Do("PATCH", "/api/v1/moods/"+id+"/", alice, map[string]any{"date": "2026-01-11"}), 400)
		e.Expect(e.Do("GET", "/api/v1/moods/not-a-uuid/", alice, nil), 404)
	})

	t.Run("today is get-or-create with mood 3", func(t *testing.T) {
		first := e.Expect(e.Do("GET", "/api/v1/moods/today/", bob, nil), 200).JSON(t)
		if first["mood"].(float64) != 3 {
			t.Fatalf("today = %v", first)
		}
		updated := e.Expect(e.Do("PATCH", "/api/v1/moods/today/", bob, map[string]any{"mood": 5, "note": "ура"}), 200).JSON(t)
		if updated["id"] != first["id"] || updated["mood"].(float64) != 5 {
			t.Fatalf("today patch = %v", updated)
		}
	})

	t.Run("stats", func(t *testing.T) {
		s := e.Expect(e.Do("GET", "/api/v1/moods/stats/", alice, nil), 200).JSON(t)
		if s["total"].(float64) != 2 || s["average"].(float64) != 3.5 || s["good_days"].(float64) != 1 || s["bad_days"].(float64) != 1 {
			t.Fatalf("stats = %v", s)
		}
		carol := e.CreateUser("carol")
		empty := e.Expect(e.Do("GET", "/api/v1/moods/stats/", carol, nil), 200).JSON(t)
		if empty["total"].(float64) != 0 || len(empty["by_mood"].(map[string]any)) != 0 {
			t.Fatalf("empty stats = %v", empty)
		}
	})

	e.Expect(e.Do("DELETE", "/api/v1/moods/"+id+"/", alice, nil), 204)
	e.Expect(e.Do("GET", "/api/v1/moods/"+id+"/", alice, nil), 404)
}
