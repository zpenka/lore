package lore

import (
	"strings"
	"testing"
	"time"
)

// TestRenderDispatch_AllModes verifies that View() produces non-empty output
// for every mode without panicking. Acts as a regression net for the
// render.go per-mode split in Task 2.
func TestRenderDispatch_AllModes(t *testing.T) {
	cases := []struct {
		name  string
		mode  int
		setup func(m model) model
	}{
		{
			name: "list mode",
			mode: modeList,
			setup: func(m model) model {
				m.sessions = []Session{{ID: "a", Project: "p", Branch: "main", Timestamp: time.Now()}}
				m.visibleSessions = m.sessions
				return m
			},
		},
		{
			name: "detail mode",
			mode: modeDetail,
			setup: func(m model) model {
				m.detailSession = Session{ID: "a", Project: "p", Branch: "main"}
				m.turns = []turn{{kind: "user", body: "hello"}}
				return m
			},
		},
		{
			name: "search entry",
			mode: modeSearch,
			setup: func(m model) model {
				m.searchMode = searchModeEntry
				m.searchQuery = "foo"
				return m
			},
		},
		{
			name: "search results",
			mode: modeSearch,
			setup: func(m model) model {
				m.searchMode = searchModeResults
				m.searchResults = []SearchHit{{Session: Session{ID: "x", Project: "p", Branch: "b"}, Snippet: "hi"}}
				return m
			},
		},
		{
			name: "project mode",
			mode: modeProject,
			setup: func(m model) model {
				m.projectCWD = "/tmp/proj"
				m.projectSessions = []Session{{ID: "a", Project: "proj", Branch: "main", Timestamp: time.Now()}}
				return m
			},
		},
		{
			name: "rerun mode",
			mode: modeRerun,
			setup: func(m model) model {
				m.detailSession = Session{ID: "a", Slug: "test"}
				m.rerunPrompt = "do stuff"
				m.rerunCWD = "/tmp"
				return m
			},
		},
		{
			name: "stats mode",
			mode: modeStats,
			setup: func(m model) model {
				m.statsData = []statsRow{{Session: Session{ID: "a", Project: "p", Branch: "main"}}}
				return m
			},
		},
		{
			name: "timeline mode",
			mode: modeTimeline,
			setup: func(m model) model {
				m.timelineCursor = startOfDay(time.Now())
				return m
			},
		},
	}

	for _, tc := range cases {
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
					t.Errorf("render %s panicked: %v", tc.name, r)
				}
			}()

			out := m.View()
			if strings.TrimSpace(out) == "" {
				t.Errorf("render %s produced empty output", tc.name)
			}
		})
	}
}
