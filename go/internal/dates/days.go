package dates

import "time"

// DaysUntil — сколько дней до даты; для ежегодных — до ближайшей годовщины.
// В отличие от Django-версии, 29 февраля в невисокосный год считается 28-м,
// а не роняет запрос с ValueError.
func DaysUntil(date, today time.Time, annually bool) int {
	if !annually {
		return days(today, date)
	}
	next := anniversary(date, today.Year())
	if next.Before(today) {
		next = anniversary(date, today.Year()+1)
	}
	return days(today, next)
}

func anniversary(date time.Time, year int) time.Time {
	day := date.Day()
	if date.Month() == time.February && day == 29 && !isLeap(year) {
		day = 28
	}
	return time.Date(year, date.Month(), day, 0, 0, 0, 0, time.UTC)
}

func isLeap(y int) bool { return y%4 == 0 && (y%100 != 0 || y%400 == 0) }

func days(from, to time.Time) int {
	from = time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, time.UTC)
	to = time.Date(to.Year(), to.Month(), to.Day(), 0, 0, 0, 0, time.UTC)
	return int(to.Sub(from).Hours() / 24)
}
