package lore

import (
	"testing"
	"time"
)

func TestTimeBucket(t *testing.T) {
	now := time.Date(2026, 5, 1, 14, 30, 0, 0, time.UTC) // a Friday

	cases := []struct {
		name string
		when time.Time
		want string
	}{
		{"earlier today", time.Date(2026, 5, 1, 9, 0, 0, 0, time.UTC), "today"},
		{"yesterday afternoon", time.Date(2026, 4, 30, 17, 0, 0, 0, time.UTC), "yesterday"},
		{"earlier this week", time.Date(2026, 4, 28, 10, 0, 0, 0, time.UTC), "this week"},
		{"last week", time.Date(2026, 4, 22, 10, 0, 0, 0, time.UTC), "last week"},
		{"this month", time.Date(2026, 4, 5, 10, 0, 0, 0, time.UTC), "this month"},
		{"older", time.Date(2025, 12, 1, 10, 0, 0, 0, time.UTC), "older"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := timeBucket(c.when, now); got != c.want {
				t.Errorf("timeBucket(%v) = %q, want %q", c.when, got, c.want)
			}
		})
	}
}
