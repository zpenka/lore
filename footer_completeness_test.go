package lore

// footer_completeness_test.go: every mode's footer must show ? help and the
// keys that are absent from the current footer strings but present in the
// help overlay (renderHelpOverlay).

import (
	"strings"
	"testing"
	"time"
)

// TestAllFooters_HaveHelpHint ensures every mode's steady-state footer shows
// "? help" so users can discover the help overlay.
func TestAllFooters_HaveHelpHint(t *testing.T) {
	cases := []struct {
		name string
		get  func() string
	}{
		{
			name: "list",
			get: func() string {
				m := loadedModelWith(Session{Project: "p", Slug: "s1", Timestamp: time.Now()})
				m.width = 220
				m.height = 40
				return renderListFooter(m)
			},
		},
		{
			name: "detail",
			get: func() string {
				m := loadedModel("a")
				m.mode = modeDetail
				m.detailSession = Session{Slug: "x", Project: "p", Branch: "b", Timestamp: time.Now()}
				m.turns = []turn{{kind: "user", body: "hi"}}
				m.expandedTurns = make(map[int]bool)
				m.width = 220
				m.height = 40
				return renderDetailFooter(m)
			},
		},
		{
			name: "search results",
			get: func() string {
				m := newModel("/d")
				m.mode = modeSearch
				m.searchMode = searchModeResults
				m.searchQuery = "x"
				m.width = 220
				m.height = 40
				return renderSearchFooter(m)
			},
		},
		{
			name: "project",
			get: func() string {
				m := newModel("/d")
				m.mode = modeProject
				m.projectCWD = "/x/p"
				m.width = 220
				m.height = 40
				return renderProjectFooter(m)
			},
		},
		{
			name: "rerun",
			get: func() string {
				m := newModel("/d")
				m.mode = modeRerun
				m.detailSession = Session{Slug: "x"}
				m.rerunPrompt = "hi"
				m.rerunCWD = "/x"
				m.width = 220
				m.height = 40
				return renderRerunFooter(m)
			},
		},
		{
			name: "stats",
			get: func() string {
				m := loadedModelWith(Session{Project: "p", Slug: "s1", Timestamp: time.Now()})
				m.mode = modeStats
				m.statsData = []statsRow{}
				m.width = 220
				m.height = 40
				return renderStatsFooter(m)
			},
		},
		{
			name: "timeline",
			get: func() string {
				m := newModel("/d")
				m.mode = modeTimeline
				m.width = 220
				m.height = 40
				return renderTimelineFooter(m)
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := tc.get()
			if !strings.Contains(out, "?") {
				t.Errorf("%s footer missing '?' help hint:\n%s", tc.name, out)
			}
		})
	}
}

// TestListFooter_HasBookmarkAndTimelineHints checks that m (bookmark toggle),
// M (bookmark-only filter), and T (timeline) are all shown in the default
// list footer. These are first-class features that users should be able to
// discover without pressing ?.
func TestListFooter_HasBookmarkAndTimelineHints(t *testing.T) {
	m := loadedModelWith(Session{Project: "p", Slug: "s1", Timestamp: time.Now()})
	m.width = 220
	m.height = 40
	out := renderListFooter(m)

	for _, want := range []string{"m bookmark", "M bookmarks", "T timeline"} {
		if !strings.Contains(out, want) {
			t.Errorf("list footer missing %q:\n%s", want, out)
		}
	}
}

// TestDetailFooter_HasBookmarkAndSearchHints checks that m (bookmark) and /
// (search) appear in the detail footer. Both are shown in the help overlay
// but were absent from the footer hint bar.
func TestDetailFooter_HasBookmarkAndSearchHints(t *testing.T) {
	m := loadedModel("a")
	m.mode = modeDetail
	m.detailSession = Session{Slug: "x", Project: "p", Branch: "b", Timestamp: time.Now()}
	m.turns = []turn{{kind: "user", body: "hi"}}
	m.expandedTurns = make(map[int]bool)
	m.width = 220
	m.height = 40
	out := renderDetailFooter(m)

	for _, want := range []string{"m bookmark", "/ search"} {
		if !strings.Contains(out, want) {
			t.Errorf("detail footer missing %q:\n%s", want, out)
		}
	}
}
