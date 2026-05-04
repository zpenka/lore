package lore

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// TestModel_FuzzyFilterEntry_PressF tests that pressing 'f' enters fuzzy filter mode.
func TestModel_FuzzyFilterEntry_PressF(t *testing.T) {
	m := loadedModel("a", "b")
	next, _ := m.Update(keyMsg("f"))
	nm := next.(model)
	if nm.filterMode != filterModeFuzzy {
		t.Errorf("after 'f': filterMode = %d, want %d (filterModeFuzzy)", nm.filterMode, filterModeFuzzy)
	}
	if nm.filterText != "" {
		t.Errorf("after 'f': filterText = %q, want ''", nm.filterText)
	}
}

// TestModel_FuzzyFilterEntry_AppendRune tests that typing in fuzzy mode appends to filterText.
func TestModel_FuzzyFilterEntry_AppendRune(t *testing.T) {
	m := loadedModel("a", "b")
	next, _ := m.Update(keyMsg("f"))
	m = next.(model)

	next, _ = m.Update(keyMsg("g"))
	m = next.(model)
	if m.filterText != "g" {
		t.Errorf("after 'f' then 'g': filterText = %q, want 'g'", m.filterText)
	}

	next, _ = m.Update(keyMsg("r"))
	m = next.(model)
	if m.filterText != "gr" {
		t.Errorf("after 'gr': filterText = %q, want 'gr'", m.filterText)
	}
}

// TestModel_FuzzyFilterEntry_Backspace tests that backspace removes the last rune.
func TestModel_FuzzyFilterEntry_Backspace(t *testing.T) {
	m := loadedModel("a", "b")
	next, _ := m.Update(keyMsg("f"))
	m = next.(model)
	next, _ = m.Update(keyMsg("h"))
	m = next.(model)
	next, _ = m.Update(keyMsg("i"))
	m = next.(model)

	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	m = next.(model)
	if m.filterText != "h" {
		t.Errorf("after backspace: filterText = %q, want 'h'", m.filterText)
	}
}

// TestModel_FuzzyFilterEntry_Enter_AppliesFilter tests that enter applies fuzzy filter to visibleSessions.
func TestModel_FuzzyFilterEntry_Enter_AppliesFilter(t *testing.T) {
	m := loadedModelWith(
		Session{Project: "grit", Branch: "main", Slug: "grit-session", Timestamp: time.Now()},
		Session{Project: "dotfiles", Branch: "main", Slug: "dot-session", Timestamp: time.Now()},
		Session{Project: "api", Branch: "feature", Slug: "api-work", Timestamp: time.Now()},
	)

	// Type "grit" in fuzzy mode - should match "grit" project/slug
	next, _ := m.Update(keyMsg("f"))
	m = next.(model)
	next, _ = m.Update(keyMsg("g"))
	m = next.(model)
	next, _ = m.Update(keyMsg("r"))
	m = next.(model)
	next, _ = m.Update(keyMsg("i"))
	m = next.(model)
	next, _ = m.Update(keyMsg("t"))
	m = next.(model)

	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(model)

	if m.filterMode != filterModeNone {
		t.Errorf("after enter: filterMode = %d, want %d (none)", m.filterMode, filterModeNone)
	}
	if m.appliedFilterMode != filterModeFuzzy {
		t.Errorf("after enter: appliedFilterMode = %d, want %d (filterModeFuzzy)", m.appliedFilterMode, filterModeFuzzy)
	}
	// "grit" should match grit project and slug — expect at least 1, not all 3
	if len(m.visibleSessions) == 0 {
		t.Error("after fuzzy filter 'grit': visibleSessions should not be empty")
	}
	if len(m.visibleSessions) >= 3 {
		t.Errorf("after fuzzy filter 'grit': visibleSessions len = %d, want < 3 (filtered)", len(m.visibleSessions))
	}
}

// TestModel_FuzzyFilterEntry_Escape_Cancels tests that esc cancels filter entry and restores full list.
func TestModel_FuzzyFilterEntry_Escape_Cancels(t *testing.T) {
	m := loadedModelWith(
		Session{Project: "grit", Branch: "main", Slug: "s1", Timestamp: time.Now()},
		Session{Project: "dotfiles", Branch: "main", Slug: "s2", Timestamp: time.Now()},
	)

	next, _ := m.Update(keyMsg("f"))
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

// TestModel_FuzzyFilterApplied_EscClears tests that esc in normal list mode clears the applied fuzzy filter.
func TestModel_FuzzyFilterApplied_EscClears(t *testing.T) {
	m := loadedModelWith(
		Session{Project: "grit", Branch: "main", Slug: "grit-session", Timestamp: time.Now()},
		Session{Project: "dotfiles", Branch: "main", Slug: "dot-session", Timestamp: time.Now()},
	)

	// Apply fuzzy filter
	next, _ := m.Update(keyMsg("f"))
	m = next.(model)
	next, _ = m.Update(keyMsg("g"))
	m = next.(model)
	next, _ = m.Update(keyMsg("r"))
	m = next.(model)
	next, _ = m.Update(keyMsg("i"))
	m = next.(model)
	next, _ = m.Update(keyMsg("t"))
	m = next.(model)
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(model)

	if m.appliedFilterMode != filterModeFuzzy {
		t.Fatalf("expected fuzzy filter to be applied, got appliedFilterMode = %d", m.appliedFilterMode)
	}

	// Esc in normal list mode should clear
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = next.(model)

	if m.filterText != "" {
		t.Errorf("after esc: filterText = %q, want ''", m.filterText)
	}
	if m.appliedFilterMode != filterModeNone {
		t.Errorf("after esc: appliedFilterMode = %d, want %d (none)", m.appliedFilterMode, filterModeNone)
	}
	if len(m.visibleSessions) != 2 {
		t.Errorf("after esc: len(visibleSessions) = %d, want 2 (full list restored)", len(m.visibleSessions))
	}
}

// TestModel_FuzzyFilter_MatchesAcrossSlugProjectBranch tests that fuzzy filter matches on slug, project, and branch.
func TestModel_FuzzyFilter_MatchesAcrossSlugProjectBranch(t *testing.T) {
	m := loadedModelWith(
		Session{Project: "myapp", Branch: "main", Slug: "some-work", Timestamp: time.Now()},
		Session{Project: "other", Branch: "feature-foo", Slug: "foo-session", Timestamp: time.Now()},
		Session{Project: "third", Branch: "bugfix", Slug: "foo-work", Timestamp: time.Now()},
	)

	// "foo" should match sessions 2 and 3 (branch and slug), not session 1
	next, _ := m.Update(keyMsg("f"))
	m = next.(model)
	next, _ = m.Update(keyMsg("f"))
	m = next.(model)
	next, _ = m.Update(keyMsg("o"))
	m = next.(model)
	next, _ = m.Update(keyMsg("o"))
	m = next.(model)
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(model)

	if len(m.visibleSessions) == 0 {
		t.Error("fuzzy filter 'foo': should match sessions with 'foo' in slug or branch")
	}
	// Should not include "myapp/main/some-work" — none of its fields contain 'foo'
	for _, s := range m.visibleSessions {
		if s.Slug == "some-work" && s.Project == "myapp" {
			t.Errorf("fuzzy filter 'foo': should not match myapp/main/some-work")
		}
	}
}

// TestModel_FuzzyFilter_EmptyQuery_ShowsAll tests that an empty fuzzy filter shows all sessions.
func TestModel_FuzzyFilter_EmptyQuery_ShowsAll(t *testing.T) {
	m := loadedModelWith(
		Session{Project: "grit", Branch: "main", Slug: "s1", Timestamp: time.Now()},
		Session{Project: "dotfiles", Branch: "main", Slug: "s2", Timestamp: time.Now()},
	)

	// Enter fuzzy mode and immediately hit enter without typing
	next, _ := m.Update(keyMsg("f"))
	m = next.(model)
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(model)

	if len(m.visibleSessions) != 2 {
		t.Errorf("empty fuzzy filter: len(visibleSessions) = %d, want 2 (all sessions)", len(m.visibleSessions))
	}
}

// TestModel_FuzzyFilter_FooterShowsEntryPrompt tests that the footer shows "fuzzy filter:" prompt while typing.
func TestModel_FuzzyFilter_FooterShowsEntryPrompt(t *testing.T) {
	m := loadedModelWith(
		Session{Project: "grit", Branch: "main", Slug: "s1", Timestamp: time.Now()},
	)
	m.width = 100

	next, _ := m.Update(keyMsg("f"))
	m = next.(model)
	next, _ = m.Update(keyMsg("g"))
	m = next.(model)

	footer := renderFooter(m)
	// Footer should contain "fuzzy filter" indicator and the typed text "g"
	if !strings.Contains(footer, "fuzzy") {
		t.Errorf("footer in fuzzy entry mode should contain 'fuzzy', got: %q", footer)
	}
}

// TestModel_FuzzyFilter_FooterShowsApplied tests footer shows fuzzy filter when applied.
func TestModel_FuzzyFilter_FooterShowsApplied(t *testing.T) {
	m := loadedModelWith(
		Session{Project: "grit", Branch: "main", Slug: "grit-session", Timestamp: time.Now()},
		Session{Project: "dotfiles", Branch: "main", Slug: "dot-session", Timestamp: time.Now()},
	)
	m.width = 100

	// Apply fuzzy filter
	next, _ := m.Update(keyMsg("f"))
	m = next.(model)
	next, _ = m.Update(keyMsg("g"))
	m = next.(model)
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(model)

	footer := renderFooter(m)
	// Footer should contain "fuzzy" indicating the applied fuzzy filter
	if !strings.Contains(footer, "fuzzy") {
		t.Errorf("footer with applied fuzzy filter should contain 'fuzzy', got: %q", footer)
	}
}

// TestModel_FuzzyFilter_HelpOverlayListMode tests that the help overlay for list mode documents 'f'.
func TestModel_FuzzyFilter_HelpOverlayListMode(t *testing.T) {
	m := loadedModel("a")
	m.mode = modeList

	helpText := renderHelpOverlay(m)
	// Help overlay for list mode should explain the 'f' fuzzy filter key
	if !strings.Contains(helpText, "fuzzy") {
		t.Errorf("help overlay for list mode should document fuzzy filter 'f' key (mention 'fuzzy'), got: %q", helpText)
	}
}
