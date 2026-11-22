package migrate

import (
	"strings"
	"testing"
)

func TestEmbeddedMigrations(t *testing.T) {
	ms, err := List()
	if err != nil {
		t.Fatal(err)
	}
	if len(ms) == 0 || ms[0].Version != baseline {
		t.Fatalf("first migration = %v, want %s", ms, baseline)
	}
	for i := 1; i < len(ms); i++ {
		if ms[i-1].Version >= ms[i].Version {
			t.Fatalf("not sorted: %s, %s", ms[i-1].Version, ms[i].Version)
		}
	}
	sql := ms[0].SQL
	for _, table := range []string{"accounts_user", "accounts_relationship", "moods_moodentry", "token_blacklist_outstandingtoken"} {
		if !strings.Contains(sql, "CREATE TABLE public."+table+" ") {
			t.Errorf("baseline has no %s", table)
		}
	}
	// search_path = '' из pg_dump сломал бы все запросы на этом соединении пула
	for _, bad := range []string{"set_config", "\\restrict", "SET search_path", "public.django_", "public.auth_"} {
		if strings.Contains(sql, bad) {
			t.Errorf("baseline contains %q", bad)
		}
	}
}
