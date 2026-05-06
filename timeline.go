package lore

import "time"

// HeatmapDay is one cell in the activity heatmap: a calendar day
// and the number of sessions whose Timestamp lands on that day.
type HeatmapDay struct {
	Date  time.Time // midnight local time of the day
	Count int
}

// Heatmap is an 8-week × 7-day activity grid. Cells[weekday][week] is
// indexed with weekday 0 = Monday and weekday 6 = Sunday; week 0 is the
// oldest week and week 7 is the most recent (the week containing the
// "now" used to build the heatmap).
type Heatmap struct {
	Cells [heatmapRows][heatmapCols]HeatmapDay
}

const (
	heatmapRows = 7 // Mon..Sun
	heatmapCols = 8 // 8 weeks; rightmost column is the current week
)

// mondayOf returns midnight of the Monday of the week containing t.
func mondayOf(t time.Time) time.Time {
	t = startOfDay(t)
	// time.Weekday: Sunday=0, Monday=1, ..., Saturday=6.
	// Convert to "days since Monday" so Monday=0, Sunday=6.
	offset := int(t.Weekday()) - 1
	if offset < 0 {
		offset = 6 // Sunday
	}
	return t.AddDate(0, 0, -offset)
}

// buildHeatmap returns a 7x8 activity grid ending in the week of `now`.
// Sessions whose Timestamp falls within the grid window are aggregated
// by day; sessions outside are ignored.
func buildHeatmap(sessions []Session, now time.Time) Heatmap {
	monday := mondayOf(now)
	earliest := monday.AddDate(0, 0, -(heatmapCols-1)*7)

	// Index counts by yyyy-mm-dd for cheap lookup.
	counts := make(map[time.Time]int, len(sessions))
	for _, s := range sessions {
		d := startOfDay(s.Timestamp)
		if d.Before(earliest) || d.After(monday.AddDate(0, 0, 6)) {
			continue
		}
		counts[d]++
	}

	var hm Heatmap
	for col := 0; col < heatmapCols; col++ {
		colMonday := earliest.AddDate(0, 0, col*7)
		for row := 0; row < heatmapRows; row++ {
			d := colMonday.AddDate(0, 0, row)
			hm.Cells[row][col] = HeatmapDay{Date: d, Count: counts[d]}
		}
	}
	return hm
}

// heatmapBucket maps a session count to an intensity level 0..3:
// 0 = empty, 1 = light (1-2 sessions), 2 = medium (3-5), 3 = bright (6+).
func heatmapBucket(count int) int {
	switch {
	case count <= 0:
		return 0
	case count <= 2:
		return 1
	case count <= 5:
		return 2
	default:
		return 3
	}
}

// countOn returns the count for date d (truncated to day) or zero.
func (h Heatmap) countOn(d time.Time) int {
	d = startOfDay(d)
	for row := 0; row < heatmapRows; row++ {
		for col := 0; col < heatmapCols; col++ {
			if h.Cells[row][col].Date.Equal(d) {
				return h.Cells[row][col].Count
			}
		}
	}
	return 0
}

// cellOf returns the (row, col, ok) location of date d in the grid.
func (h Heatmap) cellOf(d time.Time) (int, int, bool) {
	d = startOfDay(d)
	for row := 0; row < heatmapRows; row++ {
		for col := 0; col < heatmapCols; col++ {
			if h.Cells[row][col].Date.Equal(d) {
				return row, col, true
			}
		}
	}
	return 0, 0, false
}

// earliestDay returns the Monday of the leftmost column in the grid.
func (h Heatmap) earliestDay() time.Time {
	return h.Cells[0][0].Date
}

// latestDay returns the Sunday of the rightmost column in the grid.
func (h Heatmap) latestDay() time.Time {
	return h.Cells[6][heatmapCols-1].Date
}
