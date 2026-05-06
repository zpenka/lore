package lore

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// ----- buildHeatmap -----

// dayAt returns midnight (UTC) for the given Y-M-D.
func dayAt(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 12, 0, 0, 0, time.UTC)
}

func TestBuildHeatmap_GroupsSessionsByDay(t *testing.T) {
	now := dayAt(2026, 5, 15) // a Friday
	sessions := []Session{
		{Timestamp: dayAt(2026, 5, 15)},
		{Timestamp: dayAt(2026, 5, 15)},
		{Timestamp: dayAt(2026, 5, 15)},
		{Timestamp: dayAt(2026, 5, 14)},
	}
	hm := buildHeatmap(sessions, now)

	// The "today" cell should have count 3 (Friday, latest column).
	if c := hm.countOn(dayAt(2026, 5, 15)); c != 3 {
		t.Errorf("today count = %d, want 3", c)
	}
	if c := hm.countOn(dayAt(2026, 5, 14)); c != 1 {
		t.Errorf("yesterday count = %d, want 1", c)
	}
}

func TestBuildHeatmap_ExcludesSessionsOlderThan8Weeks(t *testing.T) {
	now := dayAt(2026, 5, 15)
	old := dayAt(2026, 1, 1) // > 8 weeks before May 15
	sessions := []Session{
		{Timestamp: old},
		{Timestamp: now},
	}
	hm := buildHeatmap(sessions, now)
	if c := hm.countOn(old); c != 0 {
		t.Errorf("session older than 8 weeks should not appear, got count=%d", c)
	}
	if c := hm.countOn(now); c != 1 {
		t.Errorf("today should be counted, got %d", c)
	}
}

func TestBuildHeatmap_DimensionsAre7x8(t *testing.T) {
	hm := buildHeatmap(nil, dayAt(2026, 5, 15))
	if got := len(hm.Cells); got != 7 {
		t.Errorf("Heatmap rows = %d, want 7 (Mon..Sun)", got)
	}
	if got := len(hm.Cells[0]); got != 8 {
		t.Errorf("Heatmap cols = %d, want 8 (8 weeks)", got)
	}
}

func TestBuildHeatmap_NewestWeekIsRightmost(t *testing.T) {
	now := dayAt(2026, 5, 15) // Friday
	hm := buildHeatmap(nil, now)
	// Rightmost column (index 7) should contain the week of `now`.
	rightCol := hm.Cells[0][7].Date
	if rightCol.Year() != 2026 || rightCol.Month() != time.May {
		t.Errorf("rightmost column should be week of 2026-05-15, got %v", rightCol)
	}
}

func TestBuildHeatmap_RowsMonThroughSun(t *testing.T) {
	now := dayAt(2026, 5, 15) // Friday
	hm := buildHeatmap(nil, now)
	// Row 0 should always be a Monday.
	for col := 0; col < 8; col++ {
		if hm.Cells[0][col].Date.Weekday() != time.Monday {
			t.Errorf("row 0 col %d should be Monday, got %v", col, hm.Cells[0][col].Date.Weekday())
		}
	}
	// Row 6 should always be a Sunday.
	for col := 0; col < 8; col++ {
		if hm.Cells[6][col].Date.Weekday() != time.Sunday {
			t.Errorf("row 6 col %d should be Sunday, got %v", col, hm.Cells[6][col].Date.Weekday())
		}
	}
}

// ----- heatmapBucket / styling -----

func TestHeatmapBucket_Levels(t *testing.T) {
	cases := []struct {
		count int
		want  int
	}{
		{0, 0},
		{1, 1}, {2, 1},
		{3, 2}, {5, 2},
		{6, 3}, {100, 3},
	}
	for _, c := range cases {
		if got := heatmapBucket(c.count); got != c.want {
			t.Errorf("heatmapBucket(%d) = %d, want %d", c.count, got, c.want)
		}
	}
}

// ----- model: T key -----

func TestModel_PressT_EntersTimelineMode(t *testing.T) {
	m := loadedModelWith(
		Session{Slug: "s1", Timestamp: time.Now()},
	)
	next, _ := m.Update(keyMsg("T"))
	m = next.(model)
	if m.mode != modeTimeline {
		t.Errorf("after T, mode = %d, want modeTimeline (%d)", m.mode, modeTimeline)
	}
}

// ----- model: navigation in timeline mode -----

func TestModel_TimelineMode_LeftRightMoveCursorByOneDay(t *testing.T) {
	m := loadedModelWith(
		Session{Slug: "s1", Timestamp: time.Now()},
	)
	next, _ := m.Update(keyMsg("T"))
	m = next.(model)
	start := m.timelineCursor

	next, _ = m.Update(keyMsg("h"))
	m = next.(model)
	if !m.timelineCursor.Equal(start.AddDate(0, 0, -1)) {
		t.Errorf("after h, cursor = %v, want one day earlier than %v", m.timelineCursor, start)
	}

	next, _ = m.Update(keyMsg("l"))
	m = next.(model)
	if !m.timelineCursor.Equal(start) {
		t.Errorf("after h then l, cursor = %v, want back to %v", m.timelineCursor, start)
	}
}

func TestModel_TimelineMode_RightCappedAtToday(t *testing.T) {
	m := loadedModelWith(
		Session{Slug: "s1", Timestamp: time.Now()},
	)
	next, _ := m.Update(keyMsg("T"))
	m = next.(model)
	start := m.timelineCursor
	next, _ = m.Update(keyMsg("l"))
	m = next.(model)
	if !m.timelineCursor.Equal(start) {
		t.Errorf("l from today should not advance past today; cursor moved from %v to %v", start, m.timelineCursor)
	}
}

func TestModel_TimelineMode_LeftCappedAt8WeeksAgo(t *testing.T) {
	m := loadedModelWith(
		Session{Slug: "s1", Timestamp: time.Now()},
	)
	next, _ := m.Update(keyMsg("T"))
	m = next.(model)
	for i := 0; i < 100; i++ {
		next, _ = m.Update(keyMsg("h"))
		m = next.(model)
	}
	// Cursor should not have gone further than the heatmap span (~56 days).
	earliest := time.Now().AddDate(0, 0, -8*7-1)
	if m.timelineCursor.Before(earliest) {
		t.Errorf("cursor went past heatmap span: %v < %v", m.timelineCursor, earliest)
	}
}

// ----- model: enter filters list to that date -----

func TestModel_TimelineMode_EnterFiltersListAndReturns(t *testing.T) {
	now := time.Now()
	m := loadedModelWith(
		Session{ID: "today1", Slug: "today-one", Timestamp: now},
		Session{ID: "today2", Slug: "today-two", Timestamp: now},
		Session{ID: "yest", Slug: "yest", Timestamp: now.AddDate(0, 0, -1)},
	)
	next, _ := m.Update(keyMsg("T"))
	m = next.(model)
	// Cursor starts at today; pressing enter should filter to today.
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(model)

	if m.mode != modeList {
		t.Errorf("after enter, should return to modeList, got mode=%d", m.mode)
	}
	if len(m.visibleSessions) != 2 {
		t.Errorf("after enter on today, visible sessions = %d, want 2", len(m.visibleSessions))
	}
}

func TestModel_TimelineMode_EscReturnsToList(t *testing.T) {
	m := loadedModelWith(
		Session{Slug: "s1", Timestamp: time.Now()},
	)
	next, _ := m.Update(keyMsg("T"))
	m = next.(model)
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = next.(model)
	if m.mode != modeList {
		t.Errorf("after esc in timeline, mode=%d, want modeList", m.mode)
	}
}

// ----- rendering -----

func TestRenderTimelineView_HasWeekdayLabels(t *testing.T) {
	m := loadedModelWith(
		Session{Slug: "s1", Timestamp: time.Now()},
	)
	next, _ := m.Update(keyMsg("T"))
	m = next.(model)
	m.width = 80
	m.height = 30
	out := m.View()
	// Weekday labels should be visible somewhere in the body.
	for _, want := range []string{"Mon", "Wed", "Fri"} {
		if !strings.Contains(out, want) {
			t.Errorf("timeline view missing weekday label %q:\n%s", want, out)
		}
	}
}

func TestRenderTimelineView_FooterShowsHighlightedDate(t *testing.T) {
	now := time.Now()
	m := loadedModelWith(
		Session{Slug: "s1", Timestamp: now},
	)
	next, _ := m.Update(keyMsg("T"))
	m = next.(model)
	m.width = 80
	m.height = 30
	out := m.View()
	want := now.Format("2006-01-02")
	if !strings.Contains(out, want) {
		t.Errorf("timeline footer should show highlighted date %q:\n%s", want, out)
	}
}
