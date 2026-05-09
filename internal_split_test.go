package lore

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// TestModelDispatch_AllModes verifies that the model dispatches key events to
// each mode's handler without panicking. This is a regression net for the
// per-mode handler split in Task 1: if any function moves and breaks the
// call chain, this test surfaces the failure.
func TestModelDispatch_AllModes(t *testing.T) {
	modes := []struct {
		name   string
		mode   int
		setup  func(m model) model
		key    string
	}{
		{
			name: "list mode j",
			mode: modeList,
			key:  "j",
		},
		{
			name: "detail mode j",
			mode: modeDetail,
			setup: func(m model) model {
				m.turns = []turn{{kind: "user", body: "hello"}}
				return m
			},
			key: "j",
		},
		{
			name: "search entry mode",
			mode: modeSearch,
			setup: func(m model) model {
				m.searchMode = searchModeEntry
				return m
			},
			key: "a",
		},
		{
			name: "search results mode",
			mode: modeSearch,
			setup: func(m model) model {
				m.searchMode = searchModeResults
				m.searchResults = []SearchHit{{Session: Session{ID: "x"}}}
				return m
			},
			key: "j",
		},
		{
			name: "project mode j",
			mode: modeProject,
			setup: func(m model) model {
				m.projectSessions = []Session{{ID: "x"}}
				return m
			},
			key: "j",
		},
		{
			name: "rerun mode esc",
			mode: modeRerun,
			setup: func(m model) model {
				m.rerunFn = func(prompt, cwd string) tea.Cmd { return nil }
				return m
			},
			key: "esc",
		},
		{
			name: "stats mode j",
			mode: modeStats,
			setup: func(m model) model {
				m.statsData = []statsRow{{Session: Session{ID: "x"}}}
				return m
			},
			key: "j",
		},
		{
			name: "timeline mode h",
			mode: modeTimeline,
			key:  "h",
		},
	}

	for _, tc := range modes {
		t.Run(tc.name, func(t *testing.T) {
			m := newModel("/tmp")
			m.loading = false
			m.mode = tc.mode
			m.width = 120
			m.height = 40
			if tc.setup != nil {
				m = tc.setup(m)
			}

			defer func() {
				if r := recover(); r != nil {
					t.Errorf("mode %s panicked on key %q: %v", tc.name, tc.key, r)
				}
			}()

			updated, _ := m.Update(keyMsg(tc.key))
			if updated == nil {
				t.Errorf("mode %s: Update returned nil model", tc.name)
			}
		})
	}
}
