package lore

import (
	"strings"
	"testing"
)

func TestClampOffset_CursorAlreadyVisible(t *testing.T) {
	// cursor at line 5, current offset 0, height 20 → no scroll
	got := clampOffset(0, 5, 100, 20)
	if got != 0 {
		t.Errorf("offset = %d, want 0 (cursor already visible)", got)
	}
}

func TestClampOffset_CursorBelowWindow(t *testing.T) {
	// cursor at 50, offset 0, height 20 → scroll so cursor is the last visible
	got := clampOffset(0, 50, 100, 20)
	want := 31 // 50 - 20 + 1
	if got != want {
		t.Errorf("offset = %d, want %d", got, want)
	}
}

func TestClampOffset_CursorAboveWindow(t *testing.T) {
	// cursor at 5, offset 30, height 20 → scroll up so cursor is first visible
	got := clampOffset(30, 5, 100, 20)
	if got != 5 {
		t.Errorf("offset = %d, want 5", got)
	}
}

func TestClampOffset_TotalSmallerThanHeight(t *testing.T) {
	// 5 lines fit in window of 20 → offset always 0
	got := clampOffset(0, 4, 5, 20)
	if got != 0 {
		t.Errorf("offset = %d, want 0", got)
	}
}

func TestClampOffset_BoundedAtEnd(t *testing.T) {
	// cursor near end with small list shouldn't push offset past totalLines-height
	got := clampOffset(0, 99, 100, 20)
	want := 80 // 100 - 20
	if got != want {
		t.Errorf("offset = %d, want %d", got, want)
	}
}

func TestClampOffset_NegativeValues(t *testing.T) {
	// defensive: negative inputs should not produce negative offsets
	if got := clampOffset(-5, 10, 100, 20); got < 0 {
		t.Errorf("offset = %d, want >= 0", got)
	}
	if got := clampOffset(0, -1, 100, 20); got < 0 {
		t.Errorf("offset = %d, want >= 0", got)
	}
}

func TestSliceLines_ExactWindow(t *testing.T) {
	lines := []string{"a", "b", "c", "d", "e"}
	got := sliceLines(lines, 1, 3)
	want := []string{"b", "c", "d"}
	if len(got) != len(want) {
		t.Fatalf("len(got) = %d, want %d", len(got), len(want))
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("got[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestSliceLines_PadsToHeight(t *testing.T) {
	// fewer lines than height → pad with empty strings so footer stays anchored
	lines := []string{"a", "b"}
	got := sliceLines(lines, 0, 5)
	if len(got) != 5 {
		t.Errorf("len = %d, want 5 (padded)", len(got))
	}
	if got[0] != "a" || got[1] != "b" {
		t.Errorf("first two lines = %q, %q, want a, b", got[0], got[1])
	}
	for i := 2; i < 5; i++ {
		if got[i] != "" {
			t.Errorf("got[%d] = %q, want empty (pad)", i, got[i])
		}
	}
}

func TestSliceLines_OffsetPastEnd(t *testing.T) {
	lines := []string{"a", "b", "c"}
	got := sliceLines(lines, 10, 5)
	if len(got) != 5 {
		t.Errorf("len = %d, want 5", len(got))
	}
	for i := 0; i < 5; i++ {
		if got[i] != "" {
			t.Errorf("got[%d] = %q, want empty", i, got[i])
		}
	}
}

// --- view-level scrolling integration tests ---

func TestListView_ScrollsToKeepCursorVisible(t *testing.T) {
	// 100 sessions, cursor at 80, height 20 → cursor's slug must appear
	// in the rendered output, AND the rendered output must not include
	// every session (it's clipped to the viewport).
	m := newModel("/d")
	m.loading = false
	for i := 0; i < 100; i++ {
		s := Session{
			Slug:      "session-" + intToStr(i),
			Project:   "p",
			Branch:    "b",
			CWD:       "/p",
			Timestamp: timeFromString("2026-05-01T10:00:00Z"),
		}
		m.sessions = append(m.sessions, s)
	}
	m.visibleSessions = m.sessions
	m.cursor = 80
	m.width = 100
	m.height = 25 // 25 total - chrome ≈ 20 body lines

	out := m.View()

	if !strings.Contains(out, "session-80") {
		t.Errorf("cursor's session (80) missing from rendered view — viewport not scrolling")
	}
	if strings.Contains(out, "session-0") {
		t.Errorf("session-0 should be scrolled off, but appears in output")
	}
}

func TestDetailView_ScrollsToKeepCursorVisible(t *testing.T) {
	m := loadedModel("a")
	m.mode = modeDetail
	m.detailSession = Session{Slug: "long", Project: "p", Branch: "b", Timestamp: timeFromString("2026-05-01T10:00:00Z")}
	m.expandedTurns = make(map[int]bool)
	for i := 0; i < 100; i++ {
		m.turns = append(m.turns, turn{kind: "user", body: "turn-" + intToStr(i)})
	}
	m.cursorDetail = 80
	m.width = 100
	m.height = 25

	out := m.View()
	if !strings.Contains(out, "turn-80") {
		t.Errorf("cursor turn (80) missing from rendered view — detail viewport not scrolling")
	}
	if strings.Contains(out, "turn-0") {
		t.Errorf("turn-0 should be scrolled off, but appears in output")
	}
}

func intToStr(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	if neg {
		b = append([]byte{'-'}, b...)
	}
	return string(b)
}
