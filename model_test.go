package lore

import (
	"testing"
	"time"

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

func TestModel_Init(t *testing.T) {
	m := newModel("/some/dir")
	cmd := m.Init()
	if cmd == nil {
		t.Fatal("Init returned nil cmd")
	}
	msg := cmd()
	if _, ok := msg.(sessionsLoadedMsg); !ok {
		t.Fatalf("Init cmd produced %T, want sessionsLoadedMsg", msg)
	}
}

func TestLoadSessionsCmd(t *testing.T) {
	cmd := loadSessionsCmd("/nonexistent/dir/for/testing")
	if cmd == nil {
		t.Fatal("loadSessionsCmd returned nil")
	}
	msg := cmd()
	result, ok := msg.(sessionsLoadedMsg)
	if !ok {
		t.Fatalf("loadSessionsCmd produced %T, want sessionsLoadedMsg", msg)
	}
	// loadSessionsCmd returns a sessionsLoadedMsg with no error and empty slice for nonexistent dir
	// (WalkDir doesn't error, it just walks zero files)
	if result.err != nil {
		t.Logf("Note: got error %v (acceptable for nonexistent dir)", result.err)
	}
	// sessions should be nil or empty
	if result.sessions != nil && len(result.sessions) > 0 {
		t.Errorf("sessions len = %d, want 0", len(result.sessions))
	}
}

// helpers

type errFake string

func (e errFake) Error() string { return string(e) }

func TestModel_ProjectFilterEntry_PressP(t *testing.T) {
	m := loadedModel("a", "b")
	next, _ := m.Update(keyMsg("p"))
	nm := next.(model)
	if nm.filterMode != filterModeProject {
		t.Errorf("after 'p': filterMode = %d, want %d", nm.filterMode, filterModeProject)
	}
	if nm.filterText != "" {
		t.Errorf("after 'p': filterText = %q, want ''", nm.filterText)
	}
}

func TestModel_BranchFilterEntry_PressB(t *testing.T) {
	m := loadedModel("a", "b")
	next, _ := m.Update(keyMsg("b"))
	nm := next.(model)
	if nm.filterMode != filterModeBranch {
		t.Errorf("after 'b': filterMode = %d, want %d", nm.filterMode, filterModeBranch)
	}
	if nm.filterText != "" {
		t.Errorf("after 'b': filterText = %q, want ''", nm.filterText)
	}
}

func TestModel_FilterEntry_AppendRune(t *testing.T) {
	m := loadedModel("a", "b")
	next, _ := m.Update(keyMsg("p"))
	m = next.(model)

	next, _ = m.Update(keyMsg("g"))
	m = next.(model)
	if m.filterText != "g" {
		t.Errorf("after 'p' then 'g': filterText = %q, want 'g'", m.filterText)
	}

	next, _ = m.Update(keyMsg("r"))
	m = next.(model)
	if m.filterText != "gr" {
		t.Errorf("after 'gr': filterText = %q, want 'gr'", m.filterText)
	}
}

func TestModel_FilterEntry_Backspace(t *testing.T) {
	m := loadedModel("a", "b")
	next, _ := m.Update(keyMsg("p"))
	m = next.(model)
	next, _ = m.Update(keyMsg("h"))
	m = next.(model)
	next, _ = m.Update(keyMsg("e"))
	m = next.(model)
	if m.filterText != "he" {
		t.Errorf("filterText = %q, want 'he'", m.filterText)
	}

	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	m = next.(model)
	if m.filterText != "h" {
		t.Errorf("after backspace: filterText = %q, want 'h'", m.filterText)
	}

	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	m = next.(model)
	if m.filterText != "" {
		t.Errorf("after second backspace: filterText = %q, want ''", m.filterText)
	}
}

func TestModel_FilterEntry_BackspaceWhenEmpty_NoOp(t *testing.T) {
	m := loadedModel("a", "b")
	next, _ := m.Update(keyMsg("p"))
	m = next.(model)

	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	m = next.(model)
	if m.filterText != "" {
		t.Errorf("after backspace on empty: filterText = %q, want ''", m.filterText)
	}
	if m.filterMode != filterModeProject {
		t.Errorf("still in filterMode = %d, want %d", m.filterMode, filterModeProject)
	}
}

func TestModel_FilterEntry_Enter_AppliesFilter(t *testing.T) {
	m := loadedModelWith(
		Session{Project: "grit", Branch: "main", Slug: "s1", Timestamp: time.Now()},
		Session{Project: "dotfiles", Branch: "main", Slug: "s2", Timestamp: time.Now()},
		Session{Project: "grit", Branch: "fix", Slug: "s3", Timestamp: time.Now()},
	)

	next, _ := m.Update(keyMsg("p"))
	m = next.(model)
	next, _ = m.Update(keyMsg("g"))
	m = next.(model)
	next, _ = m.Update(keyMsg("r"))
	m = next.(model)
	if m.filterText != "gr" {
		t.Fatalf("filterText = %q, want 'gr'", m.filterText)
	}

	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(model)
	if m.filterMode != filterModeNone {
		t.Errorf("after enter: filterMode = %d, want %d (none)", m.filterMode, filterModeNone)
	}
	if m.filterText != "gr" {
		t.Errorf("after enter: filterText = %q, want 'gr'", m.filterText)
	}
	if len(m.visibleSessions) != 2 {
		t.Errorf("after enter: len(visibleSessions) = %d, want 2", len(m.visibleSessions))
	}
}

func TestModel_FilterEntry_Escape_ClearsFilter(t *testing.T) {
	m := loadedModel("a", "b")
	next, _ := m.Update(keyMsg("p"))
	m = next.(model)
	next, _ = m.Update(keyMsg("g"))
	m = next.(model)

	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = next.(model)
	if m.filterMode != filterModeNone {
		t.Errorf("after esc: filterMode = %d, want %d (none)", m.filterMode, filterModeNone)
	}
	if m.filterText != "" {
		t.Errorf("after esc: filterText = %q, want ''", m.filterText)
	}
	if len(m.visibleSessions) != 2 {
		t.Errorf("after esc: len(visibleSessions) = %d, want 2 (full list restored)", len(m.visibleSessions))
	}
}

func TestModel_FilterApplied_EscapeClears(t *testing.T) {
	m := loadedModelWith(
		Session{Project: "grit", Branch: "main", Slug: "s1", Timestamp: time.Now()},
		Session{Project: "dotfiles", Branch: "main", Slug: "s2", Timestamp: time.Now()},
	)

	// Apply filter
	next, _ := m.Update(keyMsg("p"))
	m = next.(model)
	next, _ = m.Update(keyMsg("g"))
	m = next.(model)
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(model)

	if len(m.visibleSessions) != 1 {
		t.Fatalf("after filter: len(visibleSessions) = %d, want 1", len(m.visibleSessions))
	}

	// Escape while NOT in entry mode should clear filter
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = next.(model)
	if m.filterText != "" {
		t.Errorf("after esc: filterText = %q, want ''", m.filterText)
	}
	if len(m.visibleSessions) != 2 {
		t.Errorf("after esc: len(visibleSessions) = %d, want 2", len(m.visibleSessions))
	}
}

func TestModel_FilterEntry_PressP_ReEntersWithText(t *testing.T) {
	m := loadedModelWith(
		Session{Project: "grit", Branch: "main", Slug: "s1", Timestamp: time.Now()},
		Session{Project: "dotfiles", Branch: "main", Slug: "s2", Timestamp: time.Now()},
	)

	// Apply filter "g"
	next, _ := m.Update(keyMsg("p"))
	m = next.(model)
	next, _ = m.Update(keyMsg("g"))
	m = next.(model)
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(model)

	// Press p again — should re-enter with text "g"
	next, _ = m.Update(keyMsg("p"))
	m = next.(model)
	if m.filterMode != filterModeProject {
		t.Errorf("after second 'p': filterMode = %d, want %d", m.filterMode, filterModeProject)
	}
	if m.filterText != "g" {
		t.Errorf("after second 'p': filterText = %q, want 'g'", m.filterText)
	}
}

func TestModel_FilterEntry_JKey_AppendedNotNavigate(t *testing.T) {
	m := loadedModel("a", "b", "c")
	m.cursor = 0
	next, _ := m.Update(keyMsg("p"))
	m = next.(model)

	// Typing "j" should append, not move cursor
	next, _ = m.Update(keyMsg("j"))
	m = next.(model)
	if m.filterText != "j" {
		t.Errorf("after 'j' in filter entry: filterText = %q, want 'j'", m.filterText)
	}
	if m.cursor != 0 {
		t.Errorf("after 'j' in filter entry: cursor = %d, want 0 (unchanged)", m.cursor)
	}
}

func TestModel_CursorClamp_AfterFilter(t *testing.T) {
	m := loadedModelWith(
		Session{Project: "grit", Slug: "s1", Timestamp: time.Now()},
		Session{Project: "grit", Slug: "s2", Timestamp: time.Now()},
		Session{Project: "dotfiles", Slug: "s3", Timestamp: time.Now()},
	)
	m.cursor = 2

	// Apply filter that leaves 2 visible sessions
	next, _ := m.Update(keyMsg("p"))
	m = next.(model)
	next, _ = m.Update(keyMsg("g"))
	m = next.(model)
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(model)

	if len(m.visibleSessions) != 2 {
		t.Fatalf("len(visibleSessions) = %d, want 2", len(m.visibleSessions))
	}
	if m.cursor >= len(m.visibleSessions) {
		t.Errorf("after filter: cursor = %d, len(visibleSessions) = %d (cursor should be clamped)", m.cursor, len(m.visibleSessions))
	}
}

func TestModel_CursorZero_WhenAllFiltered(t *testing.T) {
	m := loadedModelWith(
		Session{Project: "grit", Slug: "s1", Timestamp: time.Now()},
		Session{Project: "dotfiles", Slug: "s2", Timestamp: time.Now()},
	)
	m.cursor = 1

	// Apply filter that hides everything
	next, _ := m.Update(keyMsg("p"))
	m = next.(model)
	next, _ = m.Update(keyMsg("x"))
	m = next.(model)
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(model)

	if len(m.visibleSessions) != 0 {
		t.Fatalf("len(visibleSessions) = %d, want 0", len(m.visibleSessions))
	}
	if m.cursor != 0 {
		t.Errorf("with empty visibleSessions: cursor = %d, want 0", m.cursor)
	}
}

func TestModel_BranchFilter_CaseInsensitive(t *testing.T) {
	m := loadedModelWith(
		Session{Branch: "Main", Slug: "s1", Timestamp: time.Now()},
		Session{Branch: "fix/Auth", Slug: "s2", Timestamp: time.Now()},
	)

	// Filter by "main" should match "Main"
	next, _ := m.Update(keyMsg("b"))
	m = next.(model)
	next, _ = m.Update(keyMsg("m"))
	m = next.(model)
	next, _ = m.Update(keyMsg("a"))
	m = next.(model)
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(model)

	if len(m.visibleSessions) != 1 {
		t.Errorf("after branch filter 'ma': len(visibleSessions) = %d, want 1", len(m.visibleSessions))
	}
}

func TestModel_NavigationOnVisibleList(t *testing.T) {
	m := loadedModelWith(
		Session{Project: "grit", Slug: "s1", Timestamp: time.Now()},
		Session{Project: "dotfiles", Slug: "s2", Timestamp: time.Now()},
		Session{Project: "grit", Slug: "s3", Timestamp: time.Now()},
	)

	// Apply filter
	next, _ := m.Update(keyMsg("p"))
	m = next.(model)
	next, _ = m.Update(keyMsg("g"))
	m = next.(model)
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(model)

	if len(m.visibleSessions) != 2 {
		t.Fatalf("len(visibleSessions) = %d, want 2", len(m.visibleSessions))
	}

	// j should navigate within visible list
	next, _ = m.Update(keyMsg("j"))
	m = next.(model)
	if m.cursor != 1 {
		t.Errorf("after 'j': cursor = %d, want 1", m.cursor)
	}

	next, _ = m.Update(keyMsg("j"))
	m = next.(model)
	if m.cursor != 1 {
		t.Errorf("after second 'j': cursor = %d, want 1 (bounded by visible)", m.cursor)
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
	m.visibleSessions = m.sessions
	return m
}

func loadedModelWith(ss ...Session) model {
	m := newModel("/d")
	m.loading = false
	m.sessions = ss
	m.visibleSessions = ss
	m.width = 100
	return m
}
