package lore

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestModel_HelpInitiallyHidden(t *testing.T) {
	m := loadedModel("a", "b")
	if m.showHelp {
		t.Error("showHelp should be false initially")
	}
}

func TestModel_QuestionMarkOpensHelp(t *testing.T) {
	m := loadedModel("a", "b")
	m.mode = modeList
	next, _ := m.Update(keyMsg("?"))
	nm := next.(model)
	if !nm.showHelp {
		t.Error("showHelp should be true after pressing ?")
	}
}

func TestModel_AnyKeyDismissesHelp(t *testing.T) {
	m := loadedModel("a", "b", "c")
	m.mode = modeList
	m.showHelp = true
	m.cursor = 0

	// Press "j" while help is showing
	next, _ := m.Update(keyMsg("j"))
	nm := next.(model)

	// Help should be dismissed
	if nm.showHelp {
		t.Error("showHelp should be false after pressing j while help was open")
	}
	// Cursor should remain unchanged (key should not be processed)
	if nm.cursor != 0 {
		t.Errorf("cursor should stay at 0, but got %d", nm.cursor)
	}
}

func TestModel_AnyKeyDismissesHelp_DetailMode(t *testing.T) {
	m := newModel("/d")
	m.loading = false
	m.sessions = []Session{{ID: "a", Slug: "a"}}
	m.visibleSessions = m.sessions
	m.mode = modeDetail
	m.detailLoading = false
	m.turns = []turn{{kind: "user", body: "hello"}}
	m.showHelp = true
	m.cursorDetail = 0

	// Press "k" while help is showing
	next, _ := m.Update(keyMsg("k"))
	nm := next.(model)

	// Help should be dismissed
	if nm.showHelp {
		t.Error("showHelp should be false after pressing k while help was open")
	}
	// Cursor should remain unchanged
	if nm.cursorDetail != 0 {
		t.Errorf("cursorDetail should stay at 0, but got %d", nm.cursorDetail)
	}
}

func TestRender_HelpOverlayInListMode(t *testing.T) {
	m := loadedModel("a", "b")
	m.mode = modeList
	m.width = 80
	m.height = 24
	m.showHelp = true

	out := m.View()

	// Help overlay should contain list mode keybindings
	if !strings.Contains(out, "j") {
		t.Error("help output should contain 'j'")
	}
	if !strings.Contains(out, "List Mode Help") {
		t.Error("help output should contain 'List Mode Help' header")
	}
}

func TestRender_HelpOverlayInDetailMode(t *testing.T) {
	m := newModel("/d")
	m.loading = false
	m.sessions = []Session{{ID: "a", Slug: "a"}}
	m.visibleSessions = m.sessions
	m.mode = modeDetail
	m.detailLoading = false
	m.turns = []turn{{kind: "user", body: "hello"}}
	m.width = 80
	m.height = 24
	m.showHelp = true

	out := m.View()

	// Help overlay should contain detail mode keybindings
	if !strings.Contains(out, "space") {
		t.Error("help output should contain 'space'")
	}
	if !strings.Contains(out, "Detail Mode Help") {
		t.Error("help output should contain 'Detail Mode Help' header")
	}
}

func TestModel_HelpDoesntGetDismissedByCtrlC(t *testing.T) {
	// Special case: ctrl-c should quit even if help is showing
	m := loadedModel("a", "b")
	m.showHelp = true

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Fatal("ctrl+c should produce a command even with help showing")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Errorf("ctrl+c should produce tea.QuitMsg, got %T", cmd())
	}
}

func TestRender_HelpOverlayInSearchMode(t *testing.T) {
	m := loadedModel("a", "b")
	m.mode = modeSearch
	m.width = 80
	m.height = 24
	m.showHelp = true

	out := m.View()

	if !strings.Contains(out, "Search Mode Help") {
		t.Error("help output should contain 'Search Mode Help' header")
	}
}

func TestRender_HelpOverlayInProjectMode(t *testing.T) {
	m := loadedModel("a", "b")
	m.mode = modeProject
	m.projectSessions = m.visibleSessions
	m.width = 80
	m.height = 24
	m.showHelp = true

	out := m.View()

	if !strings.Contains(out, "Project Mode Help") {
		t.Error("help output should contain 'Project Mode Help' header")
	}
}

func TestRender_HelpOverlayInRerunMode(t *testing.T) {
	m := newModel("/d")
	m.loading = false
	m.sessions = []Session{{ID: "a", Slug: "a"}}
	m.visibleSessions = m.sessions
	m.mode = modeRerun
	m.rerunPrompt = "test"
	m.rerunCWD = "/home/test"
	m.width = 80
	m.height = 24
	m.showHelp = true

	out := m.View()

	if !strings.Contains(out, "Re-run Mode Help") {
		t.Error("help output should contain 'Re-run Mode Help' header")
	}
}
