package lore

import "testing"

func TestNav(t *testing.T) {
	cases := []struct {
		name     string
		key      string
		cursor   int
		count    int
		halfPage int
		want     int
	}{
		// empty list
		{"empty j", "j", 0, 0, 5, 0},
		{"empty k", "k", 0, 0, 5, 0},
		{"empty G", "G", 0, 0, 5, 0},
		{"empty g", "g", 0, 0, 5, 0},
		{"empty d", "d", 0, 0, 5, 0},
		{"empty u", "u", 0, 0, 5, 0},

		// one item
		{"one item j", "j", 0, 1, 1, 0},
		{"one item k", "k", 0, 1, 1, 0},

		// mid-list navigation
		{"mid j", "j", 3, 10, 3, 4},
		{"mid down", "down", 3, 10, 3, 4},
		{"mid k", "k", 3, 10, 3, 2},
		{"mid up", "up", 3, 10, 3, 2},

		// boundary clamping
		{"bottom j clamps", "j", 9, 10, 3, 9},
		{"top k clamps", "k", 0, 10, 3, 0},

		// g/G jumps
		{"g to top", "g", 5, 10, 3, 0},
		{"G to bottom", "G", 0, 10, 3, 9},

		// half-page d/u
		{"d mid", "d", 2, 10, 3, 5},
		{"d overshoots end", "d", 8, 10, 3, 9},
		{"u mid", "u", 7, 10, 3, 4},
		{"u underflows start", "u", 1, 10, 3, 0},

		// unknown key is a no-op
		{"unknown key", "x", 3, 10, 3, 3},
		{"enter no-op", "enter", 3, 10, 3, 3},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := nav(tc.key, tc.cursor, tc.count, tc.halfPage)
			if got != tc.want {
				t.Errorf("nav(%q, cursor=%d, count=%d, half=%d) = %d, want %d",
					tc.key, tc.cursor, tc.count, tc.halfPage, got, tc.want)
			}
		})
	}
}
