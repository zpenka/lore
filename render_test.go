package lore

import (
	"strings"
	"testing"
	"time"
)

func TestView_Loading(t *testing.T) {
	m := newModel("/d") // loading=true
	out := m.View()
	if !containsFold(out, "loading") {
		t.Errorf("loading view missing 'loading' marker:\n%s", out)
	}
}

func TestView_Error(t *testing.T) {
	m := newModel("/d")
	next, _ := m.Update(sessionsLoadedMsg{err: errFake("disk on fire")})
	out := next.(model).View()
	if !strings.Contains(out, "disk on fire") {
		t.Errorf("error view missing error text:\n%s", out)
	}
}

func TestView_Empty(t *testing.T) {
	m := newModel("/some/projects")
	next, _ := m.Update(sessionsLoadedMsg{sessions: nil})
	out := next.(model).View()
	if !containsFold(out, "no sessions") {
		t.Errorf("empty view missing 'no sessions' marker:\n%s", out)
	}
	if !strings.Contains(out, "/some/projects") {
		t.Errorf("empty view missing projects dir:\n%s", out)
	}
}

func TestView_HeaderShowsCounts(t *testing.T) {
	m := loadedModelWith(
		Session{CWD: "/x/grit", Project: "grit", Branch: "main", Slug: "do-x", Timestamp: time.Now()},
		Session{CWD: "/x/grit", Project: "grit", Branch: "main", Slug: "do-y", Timestamp: time.Now()},
		Session{CWD: "/x/dotfiles", Project: "dotfiles", Branch: "main", Slug: "fix-z", Timestamp: time.Now()},
	)
	out := m.View()
	if !strings.Contains(out, "lore") {
		t.Errorf("header missing tool name 'lore':\n%s", out)
	}
	if !strings.Contains(out, "3") || !strings.Contains(out, "2") {
		t.Errorf("header should mention 3 sessions and 2 projects:\n%s", out)
	}
}

func TestView_RowsShowSessionFields(t *testing.T) {
	now := time.Now()
	m := loadedModelWith(
		Session{Project: "grit", Branch: "main", Slug: "session-one", Timestamp: now},
		Session{Project: "dotfiles", Branch: "fix/zsh", Slug: "session-two", Timestamp: now},
	)
	out := m.View()
	for _, want := range []string{"grit", "main", "session-one", "dotfiles", "fix/zsh", "session-two"} {
		if !strings.Contains(out, want) {
			t.Errorf("rows missing %q:\n%s", want, out)
		}
	}
}

func TestView_CursorMarkerOnSelectedRow(t *testing.T) {
	now := time.Now()
	m := loadedModelWith(
		Session{Slug: "alpha", Timestamp: now},
		Session{Slug: "bravo", Timestamp: now},
	)
	m.cursor = 1
	out := m.View()

	var alphaLine, bravoLine string
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "alpha") {
			alphaLine = line
		}
		if strings.Contains(line, "bravo") {
			bravoLine = line
		}
	}
	if alphaLine == "" || bravoLine == "" {
		t.Fatalf("alpha/bravo not in output:\n%s", out)
	}
	if !strings.Contains(bravoLine, "►") {
		t.Errorf("selected (bravo) line missing cursor marker:\n%q", bravoLine)
	}
	if strings.Contains(alphaLine, "►") {
		t.Errorf("unselected (alpha) line should not have cursor marker:\n%q", alphaLine)
	}
}

func TestView_GroupsByTimeBucket(t *testing.T) {
	now := time.Now()
	m := loadedModelWith(
		Session{Slug: "today-one", Timestamp: now.Add(-1 * time.Hour)},
		Session{Slug: "yesterday-one", Timestamp: now.AddDate(0, 0, -1).Add(-1 * time.Hour)},
	)
	out := m.View()
	if !strings.Contains(out, "today") {
		t.Errorf("missing 'today' bucket header:\n%s", out)
	}
	if !strings.Contains(out, "yesterday") {
		t.Errorf("missing 'yesterday' bucket header:\n%s", out)
	}
}

// helpers

func loadedModelWith(ss ...Session) model {
	m := newModel("/d")
	m.loading = false
	m.sessions = ss
	m.width = 100
	return m
}

func containsFold(haystack, needle string) bool {
	return strings.Contains(strings.ToLower(haystack), strings.ToLower(needle))
}
