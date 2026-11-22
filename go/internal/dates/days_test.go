package dates

import (
	"testing"
	"time"
)

func d(s string) time.Time {
	t, err := time.Parse(time.DateOnly, s)
	if err != nil {
		panic(err)
	}
	return t
}

func TestDaysUntil(t *testing.T) {
	tests := []struct {
		name     string
		date     string
		today    string
		annually bool
		want     int
	}{
		{"one-off in future", "2026-12-31", "2026-11-22", false, 39},
		{"one-off in past is negative", "2026-11-20", "2026-11-22", false, -2},
		{"annual later this year", "2020-12-01", "2026-11-22", true, 9},
		{"annual today", "2020-11-22", "2026-11-22", true, 0},
		{"annual already passed rolls to next year", "2020-01-10", "2026-11-22", true, 49},
		{"feb 29 in non-leap year uses feb 28", "2024-02-29", "2027-02-01", true, 27},
		{"feb 29 in leap year", "2024-02-29", "2028-02-01", true, 28},
		{"feb 29 after it passed rolls to feb 28 next year", "2024-02-29", "2026-03-01", true, 364},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DaysUntil(d(tt.date), d(tt.today), tt.annually); got != tt.want {
				t.Fatalf("DaysUntil(%s, %s, %v) = %d, want %d", tt.date, tt.today, tt.annually, got, tt.want)
			}
		})
	}
}
