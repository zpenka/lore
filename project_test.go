package lore

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestGroupByBranch_SingleBranch(t *testing.T) {
	sessions := []Session{
		{ID: "a", CWD: "/proj", Project: "proj", Branch: "main", Slug: "do-x", Timestamp: time.Now().Add(-1 * time.Hour)},
		{ID: "b", CWD: "/proj", Project: "proj", Branch: "main", Slug: "do-y", Timestamp: time.Now().Add(-2 * time.Hour)},
	}

	grouped := groupByBranch(sessions)
	if len(grouped) != 1 {
		t.Errorf("len(grouped) = %d, want 1", len(grouped))
	}
	if len(grouped[0].Sessions) != 2 {
		t.Errorf("grouped[0].Sessions len = %d, want 2", len(grouped[0].Sessions))
	}
	if grouped[0].Branch != "main" {
		t.Errorf("grouped[0].Branch = %q, want 'main'", grouped[0].Branch)
	}
}

func TestGroupByBranch_MultipleBranches(t *testing.T) {
	now := time.Now()
	sessions := []Session{
		{ID: "a", CWD: "/proj", Project: "proj", Branch: "main", Slug: "main-1", Timestamp: now.Add(-1 * time.Hour)},
		{ID: "b", CWD: "/proj", Project: "proj", Branch: "feat", Slug: "feat-1", Timestamp: now.Add(-2 * time.Hour)},
		{ID: "c", CWD: "/proj", Project: "proj", Branch: "main", Slug: "main-2", Timestamp: now.Add(-3 * time.Hour)},
		{ID: "d", CWD: "/proj", Project: "proj", Branch: "feat", Slug: "feat-2", Timestamp: now.Add(-4 * time.Hour)},
	}

	grouped := groupByBranch(sessions)
	if len(grouped) != 2 {
		t.Errorf("len(grouped) = %d, want 2", len(grouped))
	}

	// Verify branch names exist
	branchNames := make(map[string]bool)
	for _, g := range grouped {
		branchNames[g.Branch] = true
	}
	if !branchNames["main"] || !branchNames["feat"] {
		t.Errorf("branches = %v, want main and feat", branchNames)
	}
}

func TestGroupByBranch_SortedByLatestSession(t *testing.T) {
	now := time.Now()
	sessions := []Session{
		{ID: "a", CWD: "/proj", Project: "proj", Branch: "old-branch", Slug: "old-1", Timestamp: now.Add(-10 * time.Hour)},
		{ID: "b", CWD: "/proj", Project: "proj", Branch: "recent-branch", Slug: "recent-1", Timestamp: now.Add(-1 * time.Hour)},
	}

	grouped := groupByBranch(sessions)
	if len(grouped) != 2 {
		t.Errorf("len(grouped) = %d, want 2", len(grouped))
	}
	// Most recent branch group should come first
	if grouped[0].Branch != "recent-branch" {
		t.Errorf("grouped[0].Branch = %q, want 'recent-branch'", grouped[0].Branch)
	}
}

func TestGroupByBranch_WithinBranchSortedByTime(t *testing.T) {
	now := time.Now()
	sessions := []Session{
		{ID: "a", CWD: "/proj", Project: "proj", Branch: "main", Slug: "oldest", Timestamp: now.Add(-10 * time.Hour)},
		{ID: "b", CWD: "/proj", Project: "proj", Branch: "main", Slug: "newest", Timestamp: now.Add(-1 * time.Hour)},
		{ID: "c", CWD: "/proj", Project: "proj", Branch: "main", Slug: "middle", Timestamp: now.Add(-5 * time.Hour)},
	}

	grouped := groupByBranch(sessions)
	if len(grouped) != 1 {
		t.Errorf("len(grouped) = %d, want 1", len(grouped))
	}
	// Within the branch, sessions should be sorted newest first
	if grouped[0].Sessions[0].Slug != "newest" {
		t.Errorf("grouped[0].Sessions[0].Slug = %q, want 'newest'", grouped[0].Sessions[0].Slug)
	}
	if grouped[0].Sessions[1].Slug != "middle" {
		t.Errorf("grouped[0].Sessions[1].Slug = %q, want 'middle'", grouped[0].Sessions[1].Slug)
	}
	if grouped[0].Sessions[2].Slug != "oldest" {
		t.Errorf("grouped[0].Sessions[2].Slug = %q, want 'oldest'", grouped[0].Sessions[2].Slug)
	}
}

func TestModel_PressCapitalP_EntersProjectMode(t *testing.T) {
	now := time.Now()
	m := loadedModelWith(
		Session{ID: "a", CWD: "/proj1", Project: "proj1", Branch: "main", Slug: "s1", Timestamp: now},
		Session{ID: "b", CWD: "/proj2", Project: "proj2", Branch: "main", Slug: "s2", Timestamp: now.Add(-1 * time.Hour)},
	)

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("P")})
	nm := next.(model)
	if nm.mode != modeProject {
		t.Errorf("after 'P': mode = %d, want modeProject (%d)", nm.mode, modeProject)
	}
	if nm.projectCWD != "/proj1" {
		t.Errorf("after 'P': projectCWD = %q, want '/proj1'", nm.projectCWD)
	}
}

func TestModel_ProjectMode_JKNavigation(t *testing.T) {
	now := time.Now()
	sessions := []Session{
		{ID: "a", CWD: "/p", Project: "p", Branch: "main", Slug: "s1", Timestamp: now},
		{ID: "b", CWD: "/p", Project: "p", Branch: "main", Slug: "s2", Timestamp: now.Add(-1 * time.Hour)},
	}
	m := loadedModelWith(sessions...)
	m.mode = modeProject
	m.projectCWD = "/p"
	m.projectSessions = sessions
	m.projectCursor = 0

	// Press j to move down
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	nm := next.(model)
	if nm.projectCursor != 1 {
		t.Errorf("after 'j': projectCursor = %d, want 1", nm.projectCursor)
	}

	// Press k to move up
	next, _ = nm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	nm = next.(model)
	if nm.projectCursor != 0 {
		t.Errorf("after 'k': projectCursor = %d, want 0", nm.projectCursor)
	}
}

func TestModel_ProjectMode_QExitsToList(t *testing.T) {
	now := time.Now()
	m := loadedModelWith(
		Session{ID: "a", CWD: "/p", Project: "p", Branch: "main", Slug: "s1", Timestamp: now},
	)
	m.mode = modeProject
	m.projectCWD = "/p"
	m.projectSessions = []Session{}
	m.projectCursor = 0

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	nm := next.(model)
	if nm.mode != modeList {
		t.Errorf("after 'q' in project mode: mode = %d, want modeList (%d)", nm.mode, modeList)
	}
}

func TestModel_ProjectMode_EscExitsToList(t *testing.T) {
	now := time.Now()
	m := loadedModelWith(
		Session{ID: "a", CWD: "/p", Project: "p", Branch: "main", Slug: "s1", Timestamp: now},
	)
	m.mode = modeProject
	m.projectCWD = "/p"
	m.projectSessions = []Session{}
	m.projectCursor = 0

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	nm := next.(model)
	if nm.mode != modeList {
		t.Errorf("after 'esc' in project mode: mode = %d, want modeList (%d)", nm.mode, modeList)
	}
}

func TestModel_ProjectMode_EnterOpensDetail(t *testing.T) {
	now := time.Now()
	sess := Session{ID: "a", CWD: "/p", Project: "p", Branch: "main", Slug: "s1", Path: "/p/s1.jsonl", Timestamp: now}
	m := loadedModelWith(sess)
	m.mode = modeProject
	m.projectCWD = "/p"
	m.projectSessions = []Session{sess}
	m.projectCursor = 0

	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	nm := next.(model)
	if !nm.detailLoading {
		t.Errorf("after enter: detailLoading = %v, want true", nm.detailLoading)
	}
	if nm.detailSession.ID != "a" {
		t.Errorf("after enter: detailSession.ID = %q, want 'a'", nm.detailSession.ID)
	}
	if cmd == nil {
		t.Errorf("after enter: cmd should not be nil (should load detail)")
	}
}

func TestModel_ProjectMode_HExitsToList(t *testing.T) {
	now := time.Now()
	m := loadedModelWith(
		Session{ID: "a", CWD: "/p", Project: "p", Branch: "main", Slug: "s1", Timestamp: now},
	)
	m.mode = modeProject
	m.projectCWD = "/p"
	m.projectSessions = []Session{}
	m.projectCursor = 0

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("h")})
	nm := next.(model)
	if nm.mode != modeList {
		t.Errorf("after 'h' in project mode: mode = %d, want modeList (%d)", nm.mode, modeList)
	}
}

func TestModel_ProjectMode_LeftExitsToList(t *testing.T) {
	now := time.Now()
	m := loadedModelWith(
		Session{ID: "a", CWD: "/p", Project: "p", Branch: "main", Slug: "s1", Timestamp: now},
	)
	m.mode = modeProject
	m.projectCWD = "/p"
	m.projectSessions = []Session{}
	m.projectCursor = 0

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	nm := next.(model)
	if nm.mode != modeList {
		t.Errorf("after 'left' in project mode: mode = %d, want modeList (%d)", nm.mode, modeList)
	}
}

func TestRenderProjectView_Header(t *testing.T) {
	now := time.Now()
	m := loadedModelWith(
		Session{ID: "a", CWD: "/myproj", Project: "myproj", Branch: "main", Slug: "s1", Timestamp: now},
		Session{ID: "b", CWD: "/myproj", Project: "myproj", Branch: "feat", Slug: "s2", Timestamp: now.Add(-1 * time.Hour)},
	)
	m.mode = modeProject
	m.projectCWD = "/myproj"
	m.projectSessions = []Session{m.sessions[0], m.sessions[1]}
	m.projectCursor = 0
	m.width = 80

	out := renderProjectView(m, now)
	if !containsFold(out, "myproj") {
		t.Errorf("project view missing project name:\n%s", out)
	}
	if !containsFold(out, "/myproj") {
		t.Errorf("project view missing CWD:\n%s", out)
	}
	if !containsFold(out, "2") {
		t.Errorf("project view missing session count:\n%s", out)
	}
}

func TestRenderProjectView_ShowsQueryPreview(t *testing.T) {
	now := time.Now()
	m := loadedModelWith(
		Session{ID: "a", CWD: "/p", Project: "p", Branch: "main", Query: "fix the login flow", Timestamp: now},
		Session{ID: "b", CWD: "/p", Project: "p", Branch: "main", Query: "add unit tests", Timestamp: now.Add(-1 * time.Hour)},
	)
	m.mode = modeProject
	m.projectCWD = "/p"
	m.projectSessions = []Session{m.sessions[0], m.sessions[1]}
	m.projectCursor = 0
	m.width = 80

	out := renderProjectView(m, now)
	if !strings.Contains(out, "fix the login flow") {
		t.Errorf("project view should show query preview 'fix the login flow':\n%s", out)
	}
	if !strings.Contains(out, "add unit tests") {
		t.Errorf("project view should show query preview 'add unit tests':\n%s", out)
	}
}

func TestRenderProjectView_BranchGrouping(t *testing.T) {
	now := time.Now()
	m := loadedModelWith(
		Session{ID: "a", CWD: "/p", Project: "p", Branch: "main", Slug: "s1", Timestamp: now},
		Session{ID: "b", CWD: "/p", Project: "p", Branch: "feat", Slug: "s2", Timestamp: now.Add(-1 * time.Hour)},
	)
	m.mode = modeProject
	m.projectCWD = "/p"
	m.projectSessions = []Session{m.sessions[0], m.sessions[1]}
	m.projectCursor = 0
	m.width = 80

	out := renderProjectView(m, now)
	if !containsFold(out, "main") {
		t.Errorf("project view missing 'main' branch:\n%s", out)
	}
	if !containsFold(out, "feat") {
		t.Errorf("project view missing 'feat' branch:\n%s", out)
	}
}
