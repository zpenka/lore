package lore

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// --- honest footers ---

func TestListFooter_AdvertisesAllBoundKeys(t *testing.T) {
	m := loadedModelWith(
		Session{Slug: "a", Project: "p", Branch: "b", CWD: "/p", Timestamp: timeFromString("2026-05-01T10:00:00Z")},
	)
	m.height = 25

	out := m.View()
	for _, want := range []string{"enter", "/", "P", "p ", "b ", "g/G", "S "} {
		if !strings.Contains(out, want) {
			t.Errorf("list footer missing %q — keys are bound but not advertised:\n%s", want, out)
		}
	}
}

func TestDetailFooter_AdvertisesAllKeys(t *testing.T) {
	m := loadedModel("a")
	m.mode = modeDetail
	m.detailSession = Session{Slug: "x", Project: "p", Branch: "b", Timestamp: timeFromString("2026-05-01T10:00:00Z")}
	m.turns = []turn{{kind: "user", body: "hello"}}
	m.expandedTurns = make(map[int]bool)
	m.height = 25

	out := m.View()
	for _, want := range []string{"j/k", "space", "y ", "r ", "g/G", "esc"} {
		if !strings.Contains(out, want) {
			t.Errorf("detail footer missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "thinking") {
		t.Errorf("detail footer should not advertise thinking toggle (content is redacted in session files):\n%s", out)
	}
}

// --- no-op flash messages for keys that did nothing ---

func TestDetail_RKey_OnNonUserTurn_SetsFlash(t *testing.T) {
	m := loadedModel("a")
	m.mode = modeDetail
	m.detailSession = Session{Slug: "x"}
	m.turns = []turn{
		{kind: "asst", body: "hi"},
		{kind: "tool", body: "Bash 'ls'"},
	}
	m.expandedTurns = make(map[int]bool)
	m.cursorDetail = 0 // on asst turn

	next, _ := m.Update(keyMsg("r"))
	nm := next.(model)

	if nm.flashMsg == "" {
		t.Error("'r' on non-user turn: flashMsg empty, want a no-op explanation")
	}
	if nm.mode != modeDetail {
		t.Errorf("'r' on non-user turn shouldn't transition modes; mode = %d, want modeDetail", nm.mode)
	}
}

func TestDetail_SpaceKey_OnNonToolTurn_SetsFlash(t *testing.T) {
	m := loadedModel("a")
	m.mode = modeDetail
	m.detailSession = Session{Slug: "x"}
	m.turns = []turn{
		{kind: "user", body: "hi"},
	}
	m.expandedTurns = make(map[int]bool)
	m.cursorDetail = 0

	next, _ := m.Update(keyMsg(" "))
	nm := next.(model)

	if nm.flashMsg == "" {
		t.Error("space on non-tool turn: flashMsg empty, want a no-op explanation")
	}
}

func TestDetail_YKey_NoPriorUser_SetsFlash(t *testing.T) {
	m := loadedModel("a")
	m.mode = modeDetail
	m.detailSession = Session{Slug: "x"}
	m.turns = []turn{
		{kind: "asst", body: "I started without a user prompt"},
	}
	m.expandedTurns = make(map[int]bool)
	m.cursorDetail = 0
	m.clipboardFn = func(string) error { return nil }

	next, _ := m.Update(keyMsg("y"))
	nm := next.(model)

	if nm.flashMsg == "" {
		t.Error("'y' with no user turn at-or-before cursor: flashMsg empty, want a no-op explanation")
	}
}

func TestFlashMsg_ClearsOnNextKey(t *testing.T) {
	m := loadedModel("a")
	m.mode = modeDetail
	m.detailSession = Session{Slug: "x"}
	m.turns = []turn{{kind: "user", body: "hello"}}
	m.expandedTurns = make(map[int]bool)
	m.flashMsg = "stale message"

	next, _ := m.Update(keyMsg("j"))
	nm := next.(model)

	if nm.flashMsg != "" {
		t.Errorf("flashMsg should clear on any keypress; got %q", nm.flashMsg)
	}
}

// --- g / G in detail and project modes (currently unbound) ---

func TestDetail_GKey_JumpsToTop(t *testing.T) {
	m := loadedModel("a")
	m.mode = modeDetail
	m.detailSession = Session{Slug: "x"}
	for i := 0; i < 10; i++ {
		m.turns = append(m.turns, turn{kind: "user", body: "t"})
	}
	m.expandedTurns = make(map[int]bool)
	m.cursorDetail = 5

	next, _ := m.Update(keyMsg("g"))
	nm := next.(model)
	if nm.cursorDetail != 0 {
		t.Errorf("'g' in detail: cursorDetail = %d, want 0", nm.cursorDetail)
	}
}

func TestDetail_BigGKey_JumpsToBottom(t *testing.T) {
	m := loadedModel("a")
	m.mode = modeDetail
	m.detailSession = Session{Slug: "x"}
	for i := 0; i < 10; i++ {
		m.turns = append(m.turns, turn{kind: "user", body: "t"})
	}
	m.expandedTurns = make(map[int]bool)
	m.cursorDetail = 0

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("G")})
	nm := next.(model)
	if nm.cursorDetail != 9 {
		t.Errorf("'G' in detail: cursorDetail = %d, want 9", nm.cursorDetail)
	}
}
