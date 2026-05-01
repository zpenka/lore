package lore

import "time"

// timeBucket returns a human-friendly recency label for t relative to now.
// Used to group rows in the session list.
func timeBucket(t, now time.Time) string {
	today := startOfDay(now)
	switch {
	case !t.Before(today):
		return "today"
	case !t.Before(today.AddDate(0, 0, -1)):
		return "yesterday"
	case !t.Before(today.AddDate(0, 0, -7)):
		return "this week"
	case !t.Before(today.AddDate(0, 0, -14)):
		return "last week"
	case !t.Before(today.AddDate(0, 0, -30)):
		return "this month"
	default:
		return "older"
	}
}

func startOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}
