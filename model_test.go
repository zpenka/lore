package lore

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func keyMsg(s string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func TestModel_Initial(t *testing.T) {
	m := newModel("/some/dir")
	if !m.loading {
		t.Error("expected loading=true initially")
	}
	if len(m.sessions) != 0 {
		t.Errorf("sessions len = %d, want 0", len(m.sessions))
	}
	if m.cursor != 0 {
		t.Errorf("cursor = %d, want 0", m.cursor)
	}
}

func TestModel_SessionsLoaded(t *testing.T) {
	m := newModel("/d")
	ss := []Session{{ID: "a"}, {ID: "b"}}
	next, _ := m.Update(sessionsLoadedMsg{sessions: ss})
	nm := next.(model)
	if nm.loading {
		t.Error("expected loading=false after load")
	}
	if len(nm.sessions) != 2 {
		t.Errorf("len = %d, want 2", len(nm.sessions))
	}
}

func TestModel_SessionsLoaded_Error(t *testing.T) {
	m := newModel("/d")
	next, _ := m.Update(sessionsLoadedMsg{err: errFake("boom")})
	nm := next.(model)
	if nm.loading {
		t.Error("loading should be false after error")
	}
	if nm.err == nil {
		t.Error("err should be set")
	}
}

func TestModel_CursorDown_Bounded(t *testing.T) {
	m := loadedModel("a", "b", "c")

	for i, want := range []int{1, 2, 2} { // j j j → 1, 2, 2 (bounded)
		next, _ := m.Update(keyMsg("j"))
		m = next.(model)
		if m.cursor != want {
			t.Errorf("j step %d: cursor = %d, want %d", i+1, m.cursor, want)
		}
	}
}

func TestModel_CursorUp_Bounded(t *testing.T) {
	m := loadedModel("a", "b")
	m.cursor = 1

	for i, want := range []int{0, 0} { // k k → 0, 0 (bounded)
		next, _ := m.Update(keyMsg("k"))
		m = next.(model)
		if m.cursor != want {
			t.Errorf("k step %d: cursor = %d, want %d", i+1, m.cursor, want)
		}
	}
}

func TestModel_GotoTopBottom(t *testing.T) {
	m := loadedModel("a", "b", "c")

	next, _ := m.Update(keyMsg("G"))
	m = next.(model)
	if m.cursor != 2 {
		t.Errorf("after G: cursor = %d, want 2", m.cursor)
	}

	next, _ = m.Update(keyMsg("g"))
	m = next.(model)
	if m.cursor != 0 {
		t.Errorf("after g: cursor = %d, want 0", m.cursor)
	}
}

func TestModel_QuitKey(t *testing.T) {
	m := newModel("/d")
	_, cmd := m.Update(keyMsg("q"))
	if cmd == nil {
		t.Fatal("q produced nil cmd, want tea.Quit")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Errorf("q cmd produced %T, want tea.QuitMsg", cmd())
	}
}

func TestModel_CtrlC_Quits(t *testing.T) {
	m := newModel("/d")
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Fatal("ctrl+c produced nil cmd")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Errorf("ctrl+c cmd produced %T, want tea.QuitMsg", cmd())
	}
}

func TestModel_WindowSize(t *testing.T) {
	m := newModel("/d")
	next, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	nm := next.(model)
	if nm.width != 100 || nm.height != 40 {
		t.Errorf("width,height = %d,%d, want 100,40", nm.width, nm.height)
	}
}

func TestModel_NavigationIgnored_WhileLoading(t *testing.T) {
	m := newModel("/d") // loading=true, no sessions
	next, _ := m.Update(keyMsg("j"))
	nm := next.(model)
	if nm.cursor != 0 {
		t.Errorf("cursor moved while loading: got %d, want 0", nm.cursor)
	}
}

// helpers

type errFake string

func (e errFake) Error() string { return string(e) }

func loadedModel(ids ...string) model {
	m := newModel("/d")
	m.loading = false
	for _, id := range ids {
		m.sessions = append(m.sessions, Session{ID: id, Slug: id})
	}
	return m
}
