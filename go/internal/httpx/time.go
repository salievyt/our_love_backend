package httpx

import (
	"strings"
	"time"
)

// DateTime форматирует время как DRF: ISO 8601 в часовом поясе проекта,
// микросекунды только если они ненулевые, "+00:00" → "Z".
func DateTime(t time.Time, loc *time.Location) string {
	t = t.In(loc)
	layout := "2006-01-02T15:04:05-07:00"
	if t.Nanosecond()/1000 != 0 {
		layout = "2006-01-02T15:04:05.000000-07:00"
	}
	s := t.Format(layout)
	if strings.HasSuffix(s, "+00:00") {
		s = strings.TrimSuffix(s, "+00:00") + "Z"
	}
	return s
}

func Date(t time.Time) string { return t.Format(time.DateOnly) }

// Today — текущая дата в часовом поясе проекта (как django.utils.timezone.now().date()).
func Today(loc *time.Location) time.Time {
	y, m, d := time.Now().In(loc).Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}
