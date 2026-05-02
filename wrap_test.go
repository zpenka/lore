package lore

import (
	"strings"
	"testing"
)

func TestWrapText_NoWrapNeeded(t *testing.T) {
	got := wrapText("hello world", 20)
	if len(got) != 1 || got[0] != "hello world" {
		t.Errorf("got %q, want [\"hello world\"]", got)
	}
}

func TestWrapText_WrapsAtSpace(t *testing.T) {
	got := wrapText("hello world", 6)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2; got: %q", len(got), got)
	}
	if got[0] != "hello" || got[1] != "world" {
		t.Errorf("got %q, want [\"hello\" \"world\"]", got)
	}
}

func TestWrapText_PreservesNewlines(t *testing.T) {
	got := wrapText("a\n\nb", 20)
	if len(got) != 3 || got[0] != "a" || got[1] != "" || got[2] != "b" {
		t.Errorf("got %q, want [a, '', b]", got)
	}
}

func TestWrapText_HardCutsLongWord(t *testing.T) {
	got := wrapText("supercalifragilistic", 5)
	if len(got) < 4 {
		t.Errorf("expected at least 4 lines for hard-cut, got %d: %q", len(got), got)
	}
}

// --- detailBodyLines must yield one body line per visual row, so a 50-newline
// turn produces 50+ lines (not 1). Otherwise viewport scrolling is off. ---

func TestDetailBodyLines_MultilineTurnProducesMultipleLines(t *testing.T) {
	body := strings.Repeat("a long line\n", 50)
	m := loadedModel("a")
	m.mode = modeDetail
	m.detailSession = Session{Slug: "x"}
	m.turns = []turn{{kind: "user", body: body}}
	m.expandedTurns = make(map[int]bool)
	m.width = 80

	lines, _ := detailBodyLines(m)
	if len(lines) < 50 {
		t.Errorf("multi-line turn should produce >=50 body lines (one per visual row); got %d", len(lines))
	}
}

func TestDetailBodyLines_OverWidthTurnWraps(t *testing.T) {
	body := strings.Repeat("word ", 100) // ~500 chars on one logical line
	m := loadedModel("a")
	m.mode = modeDetail
	m.detailSession = Session{Slug: "x"}
	m.turns = []turn{{kind: "user", body: body}}
	m.expandedTurns = make(map[int]bool)
	m.width = 80 // 500 chars / 80 cols ≈ 7+ wrapped rows

	lines, _ := detailBodyLines(m)
	if len(lines) < 5 {
		t.Errorf("over-width turn should wrap to multiple body lines; got %d", len(lines))
	}
}

func TestDetailBodyLines_CursorLineOnFirstVisualRowOfSelectedTurn(t *testing.T) {
	// First turn produces 3 visual rows; second turn is selected.
	body3 := "line a\nline b\nline c"
	m := loadedModel("a")
	m.mode = modeDetail
	m.detailSession = Session{Slug: "x"}
	m.turns = []turn{
		{kind: "user", body: body3},
		{kind: "asst", body: "second"},
	}
	m.expandedTurns = make(map[int]bool)
	m.cursorDetail = 1
	m.width = 80

	lines, cursorLine := detailBodyLines(m)
	if cursorLine != 3 {
		t.Errorf("cursorLine = %d, want 3 (first row of second turn after 3 rows of first); body=%v", cursorLine, lines)
	}
}
