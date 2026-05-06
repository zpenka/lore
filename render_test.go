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

func TestView_RowsShowQueryPreview(t *testing.T) {
	now := time.Now()
	m := loadedModelWith(
		Session{Project: "grit", Branch: "main", Query: "fix the login bug", Timestamp: now},
		Session{Project: "dotfiles", Branch: "fix/zsh", Query: "review this MR", Timestamp: now},
	)
	m.width = 120
	out := m.View()
	if !strings.Contains(out, "fix the login bug") {
		t.Errorf("list view should show query preview 'fix the login bug':\n%s", out)
	}
	if !strings.Contains(out, "review this MR") {
		t.Errorf("list view should show query preview 'review this MR':\n%s", out)
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

func TestPlural_Singular(t *testing.T) {
	if plural(1) != "" {
		t.Errorf("plural(1) = %q, want empty", plural(1))
	}
}

func TestPlural_Plural(t *testing.T) {
	if plural(0) != "s" {
		t.Errorf("plural(0) = %q, want 's'", plural(0))
	}
	if plural(2) != "s" {
		t.Errorf("plural(2) = %q, want 's'", plural(2))
	}
}

func TestPadTrunc_NoTruncation(t *testing.T) {
	result := padTrunc("hi", 10)
	if result != "hi        " {
		t.Errorf("padTrunc('hi', 10) = %q, want 'hi        '", result)
	}
}

func TestPadTrunc_Truncate(t *testing.T) {
	result := padTrunc("very-long-branch-name", 10)
	// When truncating to 10, we take max-1=9 chars plus "…"
	if result != "very-long…" {
		t.Errorf("padTrunc with truncate = %q, want 'very-long…'", result)
	}
}

func TestPadTrunc_TruncateWithMax1(t *testing.T) {
	result := padTrunc("hi", 1)
	if result != "h" {
		t.Errorf("padTrunc('hi', 1) = %q, want 'h'", result)
	}
}

func TestRenderDivider_SmallWidth(t *testing.T) {
	result := renderDivider(2)
	if result != strings.Repeat("─", 80) {
		t.Errorf("renderDivider(2) should default to 80, got %d chars", len(result))
	}
}

func TestRenderDivider_NormalWidth(t *testing.T) {
	result := renderDivider(40)
	if result != strings.Repeat("─", 40) {
		t.Errorf("renderDivider(40) = %d chars, want 40", len(result))
	}
}

func TestRenderFooter_DefaultFooter(t *testing.T) {
	m := loadedModelWith(
		Session{Project: "grit", Slug: "s1", Timestamp: time.Now()},
	)
	out := renderFooter(m)
	if !strings.Contains(out, "j/k move") {
		t.Errorf("default footer missing 'j/k move':\n%s", out)
	}
	if !strings.Contains(out, "p filter project") {
		t.Errorf("default footer missing 'p filter project':\n%s", out)
	}
	if !strings.Contains(out, "b filter branch") {
		t.Errorf("default footer missing 'b filter branch':\n%s", out)
	}
}

func TestRenderFooter_ProjectFilterEntry(t *testing.T) {
	m := loadedModelWith(
		Session{Project: "grit", Slug: "s1", Timestamp: time.Now()},
	)
	m.filterMode = filterModeProject
	m.filterText = "gr"
	out := renderFooter(m)
	if !strings.Contains(out, "project filter: gr_") {
		t.Errorf("project filter entry footer wrong:\n%s", out)
	}
	if !strings.Contains(out, "[enter] apply") {
		t.Errorf("project filter entry footer missing '[enter] apply':\n%s", out)
	}
	if !strings.Contains(out, "[esc] cancel") {
		t.Errorf("project filter entry footer missing '[esc] cancel':\n%s", out)
	}
}

func TestRenderFooter_BranchFilterEntry(t *testing.T) {
	m := loadedModelWith(
		Session{Branch: "main", Slug: "s1", Timestamp: time.Now()},
	)
	m.filterMode = filterModeBranch
	m.filterText = "ma"
	out := renderFooter(m)
	if !strings.Contains(out, "branch filter: ma_") {
		t.Errorf("branch filter entry footer wrong:\n%s", out)
	}
	if !strings.Contains(out, "[enter] apply") {
		t.Errorf("branch filter entry footer missing '[enter] apply':\n%s", out)
	}
	if !strings.Contains(out, "[esc] cancel") {
		t.Errorf("branch filter entry footer missing '[esc] cancel':\n%s", out)
	}
}

func TestRenderFooter_ProjectFilterApplied(t *testing.T) {
	m := loadedModelWith(
		Session{Project: "grit", Slug: "s1", Timestamp: time.Now()},
		Session{Project: "dotfiles", Slug: "s2", Timestamp: time.Now()},
	)
	m.filterText = "gr"
	m.appliedFilterMode = filterModeProject
	out := renderFooter(m)
	if !strings.Contains(out, "filtered by project: gr") {
		t.Errorf("project filter applied footer wrong:\n%s", out)
	}
	if !strings.Contains(out, "esc clear") {
		t.Errorf("project filter applied footer missing 'esc clear':\n%s", out)
	}
}

func TestRenderFooter_BranchFilterApplied(t *testing.T) {
	m := loadedModelWith(
		Session{Branch: "main", Slug: "s1", Timestamp: time.Now()},
		Session{Branch: "fix", Slug: "s2", Timestamp: time.Now()},
	)
	m.filterText = "fi"
	m.appliedFilterMode = filterModeBranch
	out := renderFooter(m)
	if !strings.Contains(out, "filtered by branch: fi") {
		t.Errorf("branch filter applied footer wrong:\n%s", out)
	}
	if !strings.Contains(out, "esc clear") {
		t.Errorf("branch filter applied footer missing 'esc clear':\n%s", out)
	}
}

// Test detail view header turn position indicator

func TestDetailView_HeaderShowsTurnPosition_FirstTurn(t *testing.T) {
	m := newModel("/d")
	m.mode = modeDetail
	m.detailSession = Session{Slug: "test-session", Project: "p", Branch: "b", Timestamp: timeFromString("2026-05-01T14:30:00Z")}
	m.turns = []turn{
		{kind: "user", body: "first message"},
		{kind: "asst", body: "first response"},
		{kind: "user", body: "second message"},
	}
	m.cursorDetail = 0
	m.width = 100
	m.height = 40

	out := m.View()
	if !strings.Contains(out, "turn 1/3") {
		t.Errorf("header should show 'turn 1/3' when cursor is at 0: %s", out)
	}
}

func TestDetailView_HeaderShowsTurnPosition_LastTurn(t *testing.T) {
	m := newModel("/d")
	m.mode = modeDetail
	m.detailSession = Session{Slug: "test-session", Project: "p", Branch: "b", Timestamp: timeFromString("2026-05-01T14:30:00Z")}
	m.turns = []turn{
		{kind: "user", body: "first message"},
		{kind: "asst", body: "first response"},
		{kind: "user", body: "second message"},
	}
	m.cursorDetail = 2
	m.width = 100
	m.height = 40

	out := m.View()
	if !strings.Contains(out, "turn 3/3") {
		t.Errorf("header should show 'turn 3/3' when cursor is at 2: %s", out)
	}
}

func TestDetailView_HeaderNoTurnIndicator_WhenEmpty(t *testing.T) {
	m := newModel("/d")
	m.mode = modeDetail
	m.detailSession = Session{Slug: "test-session", Project: "p", Branch: "b", Timestamp: timeFromString("2026-05-01T14:30:00Z")}
	m.turns = nil
	m.cursorDetail = 0
	m.width = 100
	m.height = 40

	out := m.View()
	if strings.Contains(out, "turn ") {
		t.Errorf("header should not show turn indicator when there are no turns: %s", out)
	}
}

// Test detail view rendering

func TestDetailView_RenderExpandedTool(t *testing.T) {
	m := newModel("/d")
	m.mode = modeDetail
	m.detailSession = Session{Slug: "test", Project: "p", Branch: "b", Timestamp: timeFromString("2026-05-01T14:30:00Z")}
	m.turns = []turn{
		{kind: "tool", body: "Read file.go", input: map[string]interface{}{"file_path": "/path/to/file"}},
	}
	m.expandedTurns = map[int]bool{0: true}
	m.cursorDetail = 0
	m.width = 100
	m.height = 40

	out := m.View()
	if !strings.Contains(out, "file_path") {
		t.Errorf("expanded tool not rendered correctly: %s", out)
	}
}

func TestDetailView_DetailFooter_NoThinkingToggle(t *testing.T) {
	m := newModel("/d")
	m.mode = modeDetail
	m.detailSession = Session{Slug: "test", Project: "p", Branch: "b", Timestamp: timeFromString("2026-05-01T14:30:00Z")}
	m.turns = []turn{{kind: "user", body: "msg"}}
	m.justCopied = false
	m.width = 100
	m.height = 40

	out := m.View()
	if strings.Contains(out, "thinking") {
		t.Errorf("footer should not mention thinking (content is redacted in session files): %s", out)
	}
	if !strings.Contains(out, "expand") {
		t.Errorf("footer should mention 'expand': %s", out)
	}
	if !strings.Contains(out, "copy") {
		t.Errorf("footer should mention 'copy': %s", out)
	}
}

func TestDetailView_DetailFooter_WithCopied(t *testing.T) {
	m := newModel("/d")
	m.mode = modeDetail
	m.detailSession = Session{Slug: "test", Project: "p", Branch: "b", Timestamp: timeFromString("2026-05-01T14:30:00Z")}
	m.turns = []turn{{kind: "user", body: "msg"}}
	m.justCopied = true
	m.width = 100
	m.height = 40

	out := m.View()
	if !strings.Contains(out, "copied") {
		t.Errorf("footer should show 'copied' status: %s", out)
	}
}

func TestDetailView_RenderExpandedToolInputMultipleFields(t *testing.T) {
	m := newModel("/d")
	m.mode = modeDetail
	m.detailSession = Session{Slug: "test", Project: "p", Branch: "b", Timestamp: timeFromString("2026-05-01T14:30:00Z")}
	m.turns = []turn{
		{
			kind: "tool",
			body: "Bash command",
			input: map[string]interface{}{
				"command": "ls -la",
				"timeout": "30s",
			},
		},
	}
	m.expandedTurns = map[int]bool{0: true}
	m.cursorDetail = 0
	m.width = 100
	m.height = 40

	out := m.View()
	if !strings.Contains(out, "command") {
		t.Errorf("expanded tool should show command field: %s", out)
	}
	if !strings.Contains(out, "timeout") {
		t.Errorf("expanded tool should show timeout field: %s", out)
	}
}

func TestRender_EditToolDiff(t *testing.T) {
	m := newModel("/d")
	m.mode = modeDetail
	m.detailSession = Session{Slug: "test", Project: "p", Branch: "b", Timestamp: timeFromString("2026-05-01T14:30:00Z")}
	m.turns = []turn{
		{
			kind: "tool",
			body: "Edit /x.go",
			input: map[string]interface{}{
				"file_path":  "/x.go",
				"old_string": "foo\nbar",
				"new_string": "foo\nbaz",
			},
		},
	}
	m.expandedTurns = map[int]bool{0: true}
	m.cursorDetail = 0
	m.width = 100
	m.height = 40

	out := m.View()
	if !strings.Contains(out, "/x.go") {
		t.Errorf("Edit tool diff should show file path: %s", out)
	}
	if !strings.Contains(out, "- bar") {
		t.Errorf("Edit tool diff should show removed line with '- ' prefix: %s", out)
	}
	if !strings.Contains(out, "+ baz") {
		t.Errorf("Edit tool diff should show added line with '+ ' prefix: %s", out)
	}
	// Verify old and new context lines are present
	if !strings.Contains(out, "- foo") {
		t.Errorf("Edit tool diff should show old context line '- foo': %s", out)
	}
	if !strings.Contains(out, "+ foo") {
		t.Errorf("Edit tool diff should show new context line '+ foo': %s", out)
	}
}

func TestRender_WriteToolAddOnly(t *testing.T) {
	m := newModel("/d")
	m.mode = modeDetail
	m.detailSession = Session{Slug: "test", Project: "p", Branch: "b", Timestamp: timeFromString("2026-05-01T14:30:00Z")}
	m.turns = []turn{
		{
			kind: "tool",
			body: "Write /y.go",
			input: map[string]interface{}{
				"file_path": "/y.go",
				"content":   "hello\nworld",
			},
		},
	}
	m.expandedTurns = map[int]bool{0: true}
	m.cursorDetail = 0
	m.width = 100
	m.height = 40

	out := m.View()
	if !strings.Contains(out, "/y.go") {
		t.Errorf("Write tool should show file path: %s", out)
	}
	if !strings.Contains(out, "+ hello") {
		t.Errorf("Write tool should show content line with '+ ' prefix: %s", out)
	}
	if !strings.Contains(out, "+ world") {
		t.Errorf("Write tool should show content line with '+ ' prefix: %s", out)
	}
	if strings.Contains(out, "- ") {
		t.Errorf("Write tool should NOT contain '- ' remove lines: %s", out)
	}
}

func TestRender_BashToolUnchanged(t *testing.T) {
	m := newModel("/d")
	m.mode = modeDetail
	m.detailSession = Session{Slug: "test", Project: "p", Branch: "b", Timestamp: timeFromString("2026-05-01T14:30:00Z")}
	m.turns = []turn{
		{
			kind: "tool",
			body: "Bash ls -la",
			input: map[string]interface{}{
				"command": "ls -la",
			},
		},
	}
	m.expandedTurns = map[int]bool{0: true}
	m.cursorDetail = 0
	m.width = 100
	m.height = 40

	out := m.View()
	if !strings.Contains(out, "command") {
		t.Errorf("Bash tool should show 'command' key: %s", out)
	}
	if !strings.Contains(out, "ls -la") {
		t.Errorf("Bash tool should show command value: %s", out)
	}
	// Verify this uses key: value format, not diff format
	if !strings.Contains(out, "command:") {
		t.Errorf("Bash tool should use 'key:' format, not diff format: %s", out)
	}
}

// helpers

func containsFold(haystack, needle string) bool {
	return strings.Contains(strings.ToLower(haystack), strings.ToLower(needle))
}

// Search view tests

func TestSearchView_EntryMode(t *testing.T) {
	m := newModel("/d")
	m.mode = modeSearch
	m.searchMode = searchModeEntry
	m.searchQuery = "hello"
	m.width = 100
	m.height = 40

	out := m.View()
	if !strings.Contains(out, "hello") {
		t.Errorf("search entry should show query: %s", out)
	}
	if !strings.Contains(out, "[enter]") || !strings.Contains(out, "[esc]") {
		t.Errorf("search entry should show instructions: %s", out)
	}
}

func TestSearchView_ResultsMode_Empty(t *testing.T) {
	m := newModel("/d")
	m.mode = modeSearch
	m.searchMode = searchModeResults
	m.searchQuery = "xyz"
	m.searchResults = nil
	m.width = 100
	m.height = 40

	out := m.View()
	if !strings.Contains(out, "no matches") {
		t.Errorf("empty results should show '(no matches)': %s", out)
	}
}

func TestSearchView_ResultsMode_WithHits(t *testing.T) {
	m := newModel("/d")
	m.mode = modeSearch
	m.searchMode = searchModeResults
	m.searchQuery = "token"
	m.searchResults = []SearchHit{
		{
			Session:  Session{ID: "1", Slug: "s1", Project: "grit", Branch: "main", Timestamp: timeFromString("2026-05-01T14:30:00Z")},
			HitCount: 3,
			Snippet:  "refresh the token and rotate",
		},
		{
			Session:  Session{ID: "2", Slug: "s2", Project: "api", Branch: "feat/auth", Timestamp: timeFromString("2026-05-01T13:00:00Z")},
			HitCount: 1,
			Snippet:  "handle token expiry",
		},
	}
	m.width = 100
	m.height = 40

	out := m.View()
	if !strings.Contains(out, "token") {
		t.Errorf("results should show query: %s", out)
	}
	if !strings.Contains(out, "s1") || !strings.Contains(out, "s2") {
		t.Errorf("results should show session slugs: %s", out)
	}
	if !strings.Contains(out, "refresh the token") {
		t.Errorf("results should show snippet: %s", out)
	}
	if !strings.Contains(out, "hits") {
		t.Errorf("results should show 'hits' label: %s", out)
	}
}

func TestSearchView_ResultsMode_CursorHighlight(t *testing.T) {
	m := newModel("/d")
	m.mode = modeSearch
	m.searchMode = searchModeResults
	m.searchQuery = "test"
	m.searchResults = []SearchHit{
		{
			Session:  Session{ID: "1", Slug: "s1", Project: "p", Branch: "b", Timestamp: timeFromString("2026-05-01T14:00:00Z")},
			HitCount: 1,
			Snippet:  "test snippet 1",
		},
		{
			Session:  Session{ID: "2", Slug: "s2", Project: "p", Branch: "b", Timestamp: timeFromString("2026-05-01T13:00:00Z")},
			HitCount: 1,
			Snippet:  "test snippet 2",
		},
	}
	m.searchCursor = 1
	m.width = 100
	m.height = 40

	out := m.View()
	// Both results should be rendered
	if !strings.Contains(out, "s1") || !strings.Contains(out, "s2") {
		t.Errorf("both results should be rendered: %s", out)
	}
}

func TestSearchView_HeaderShowsHitCount(t *testing.T) {
	m := newModel("/d")
	m.mode = modeSearch
	m.searchMode = searchModeResults
	m.searchQuery = "auth"
	m.searchResults = []SearchHit{
		{Session: Session{ID: "1", Slug: "s1", Project: "p", Branch: "b", Timestamp: timeFromString("2026-05-01T14:00:00Z")}, HitCount: 5, Snippet: "auth"},
		{Session: Session{ID: "2", Slug: "s2", Project: "p", Branch: "b", Timestamp: timeFromString("2026-05-01T13:00:00Z")}, HitCount: 2, Snippet: "auth"},
	}
	m.width = 100
	m.height = 40

	out := m.View()
	if !strings.Contains(out, "7") { // 5 + 2 hits
		t.Errorf("header should show total hits (7): %s", out)
	}
}

func TestSearchView_FooterSearchMode_Navigation(t *testing.T) {
	m := newModel("/d")
	m.mode = modeSearch
	m.searchMode = searchModeResults
	m.searchQuery = "test"
	m.searchResults = []SearchHit{
		{Session: Session{ID: "1", Slug: "s1", Project: "p", Branch: "b", Timestamp: timeFromString("2026-05-01T14:00:00Z")}, HitCount: 1, Snippet: "test"},
	}
	m.width = 100
	m.height = 40

	out := m.View()
	if !strings.Contains(out, "j/k") || !strings.Contains(out, "enter") {
		t.Errorf("footer should show navigation hints: %s", out)
	}
}

// Re-run view tests

func TestRerunView_Header_IncludesSourceSlug(t *testing.T) {
	m := loadedModel("a")
	m.mode = modeRerun
	m.detailSession = Session{ID: "a", Slug: "my-session", Project: "proj", Branch: "main", CWD: "/home/test"}
	m.rerunPrompt = "hello world"
	m.rerunCWD = "/home/test"
	m.width = 100
	m.height = 40

	out := m.View()
	if !strings.Contains(out, "re-run") {
		t.Errorf("header should say 're-run': %s", out)
	}
	if !strings.Contains(out, "my-session") {
		t.Errorf("header should show source slug: %s", out)
	}
}

func TestRerunView_Body_ShowsPrompt(t *testing.T) {
	m := loadedModel("a")
	m.mode = modeRerun
	m.detailSession = Session{ID: "a", Slug: "test-slug", Project: "proj", Branch: "main", CWD: "/home/test"}
	m.rerunPrompt = "hello world"
	m.rerunCWD = "/home/test"
	m.width = 100
	m.height = 40

	out := m.View()
	if !strings.Contains(out, "hello world") {
		t.Errorf("body should show prompt: %s", out)
	}
	if !strings.Contains(out, "/home/test") {
		t.Errorf("body should show cwd: %s", out)
	}
	if !strings.Contains(out, "prompt:") {
		t.Errorf("body should have 'prompt:' label: %s", out)
	}
	if !strings.Contains(out, "cwd:") {
		t.Errorf("body should have 'cwd:' label: %s", out)
	}
}

func TestRerunView_Body_RendersBoxAroundPrompt(t *testing.T) {
	m := loadedModel("a")
	m.mode = modeRerun
	m.detailSession = Session{ID: "a", Slug: "test-slug", Project: "proj", Branch: "main", CWD: "/home/test"}
	m.rerunPrompt = "hello world"
	m.rerunCWD = "/home/test"
	m.width = 100
	m.height = 40

	out := m.View()
	if !strings.Contains(out, "┌") || !strings.Contains(out, "─") || !strings.Contains(out, "┐") {
		t.Errorf("body should have box top border: %s", out)
	}
	if !strings.Contains(out, "└") || !strings.Contains(out, "┘") {
		t.Errorf("body should have box bottom border: %s", out)
	}
	if !strings.Contains(out, "│") {
		t.Errorf("body should have box side borders: %s", out)
	}
}

func TestRerunView_Footer_ShowsRunAndCancel(t *testing.T) {
	m := loadedModel("a")
	m.mode = modeRerun
	m.detailSession = Session{ID: "a", Slug: "test-slug", Project: "proj", Branch: "main", CWD: "/home/test"}
	m.rerunPrompt = "hello world"
	m.rerunCWD = "/home/test"
	m.width = 100
	m.height = 40

	out := m.View()
	if !strings.Contains(out, "enter") || !strings.Contains(out, "run") {
		t.Errorf("footer should show 'enter run': %s", out)
	}
	if !strings.Contains(out, "esc") || !strings.Contains(out, "cancel") {
		t.Errorf("footer should show 'esc cancel': %s", out)
	}
}

// ----- stats mode view tests -----

func TestStatsView_Header(t *testing.T) {
	m := loadedModelWith(
		Session{Project: "lore", Branch: "main", Slug: "s1", Timestamp: time.Now()},
		Session{Project: "lore", Branch: "feat", Slug: "s2", Timestamp: time.Now()},
	)
	m.mode = modeStats
	m.statsData = []statsRow{
		{Session: m.sessions[0], Stats: SessionStats{Model: "claude-opus-4-6", InputTokens: 1000, OutputTokens: 500}},
		{Session: m.sessions[1], Stats: SessionStats{Model: "claude-sonnet-4-6", InputTokens: 2000, OutputTokens: 300}},
	}
	m.width = 100
	m.height = 40

	out := m.View()
	if !strings.Contains(out, "usage stats") {
		t.Errorf("stats header missing 'usage stats':\n%s", out)
	}
	if !strings.Contains(out, "2") {
		t.Errorf("stats header missing session count:\n%s", out)
	}
}

func TestStatsView_ShowsSessionData(t *testing.T) {
	m := loadedModelWith(
		Session{Project: "lore", Branch: "main", Slug: "s1", Timestamp: time.Now()},
	)
	m.mode = modeStats
	m.statsData = []statsRow{
		{Session: m.sessions[0], Stats: SessionStats{Model: "claude-opus-4-6", InputTokens: 5000, OutputTokens: 2000}},
	}
	m.statsCursor = 0
	m.width = 100
	m.height = 40

	out := m.View()
	if !strings.Contains(out, "lore") {
		t.Errorf("stats view missing project name:\n%s", out)
	}
	if !strings.Contains(out, "main") {
		t.Errorf("stats view missing branch:\n%s", out)
	}
	if !strings.Contains(out, "opus") {
		t.Errorf("stats view missing model name:\n%s", out)
	}
}

func TestStatsView_Footer(t *testing.T) {
	m := loadedModelWith(
		Session{Project: "lore", Branch: "main", Slug: "s1", Timestamp: time.Now()},
	)
	m.mode = modeStats
	m.statsData = []statsRow{}
	m.width = 100
	m.height = 40

	out := m.View()
	if !strings.Contains(out, "j/k") {
		t.Errorf("stats footer missing 'j/k':\n%s", out)
	}
	if !strings.Contains(out, "esc") {
		t.Errorf("stats footer missing 'esc':\n%s", out)
	}
}

// ----- unified footer tests (1A) -----

// All sub-view footers must show "q/esc/h/← back" for the back-nav hint.
// The list footer shows "q quit" instead.

func TestListFooter_HasQuitHint(t *testing.T) {
	m := loadedModelWith(
		Session{Project: "p", Slug: "s1", Timestamp: time.Now()},
	)
	m.width = 200
	m.height = 40
	out := renderListFooter(m)
	if !strings.Contains(out, "q quit") {
		t.Errorf("list footer should show 'q quit':\n%s", out)
	}
	if strings.Contains(out, "q/esc/h/← back") {
		t.Errorf("list footer should not show 'back' hint (it's the root view):\n%s", out)
	}
}

func TestDetailFooter_HasBackHint(t *testing.T) {
	m := loadedModel("a")
	m.mode = modeDetail
	m.detailSession = Session{Slug: "x", Project: "p", Branch: "b", Timestamp: time.Now()}
	m.turns = []turn{{kind: "user", body: "hi"}}
	m.expandedTurns = make(map[int]bool)
	m.width = 200
	m.height = 40
	out := renderDetailFooter(m)
	if !strings.Contains(out, "q/esc/h/← back") {
		t.Errorf("detail footer should show 'q/esc/h/← back':\n%s", out)
	}
}

func TestSearchFooter_HasBackHint(t *testing.T) {
	m := newModel("/d")
	m.mode = modeSearch
	m.searchMode = searchModeResults
	m.searchQuery = "x"
	m.searchResults = []SearchHit{
		{Session: Session{ID: "1", Slug: "s1", Project: "p", Branch: "b", Timestamp: time.Now()}, HitCount: 1, Snippet: "x"},
	}
	m.width = 200
	m.height = 40
	out := renderSearchFooter(m)
	if !strings.Contains(out, "q/esc/h/← back") {
		t.Errorf("search results footer should show 'q/esc/h/← back':\n%s", out)
	}
}

func TestProjectFooter_HasBackHint(t *testing.T) {
	m := newModel("/d")
	m.mode = modeProject
	m.projectCWD = "/x/p"
	m.width = 200
	m.height = 40
	out := renderProjectFooter(m)
	if !strings.Contains(out, "q/esc/h/← back") {
		t.Errorf("project footer should show 'q/esc/h/← back':\n%s", out)
	}
}

func TestRerunFooter_HasBackHint(t *testing.T) {
	m := newModel("/d")
	m.mode = modeRerun
	m.detailSession = Session{Slug: "x"}
	m.rerunPrompt = "hi"
	m.rerunCWD = "/x"
	m.width = 200
	m.height = 40
	out := renderRerunFooter(m)
	if !strings.Contains(out, "q/esc/h/← back") {
		t.Errorf("rerun footer should show 'q/esc/h/← back':\n%s", out)
	}
	if !strings.Contains(out, "enter run") {
		t.Errorf("rerun footer should still show 'enter run':\n%s", out)
	}
}

func TestStatsFooter_HasBackHintAndPaging(t *testing.T) {
	m := loadedModelWith(
		Session{Project: "p", Slug: "s1", Timestamp: time.Now()},
	)
	m.mode = modeStats
	m.statsData = []statsRow{}
	m.width = 200
	m.height = 40
	out := renderStatsFooter(m)
	if !strings.Contains(out, "q/esc/h/← back") {
		t.Errorf("stats footer should show 'q/esc/h/← back':\n%s", out)
	}
	if !strings.Contains(out, "d/u page") {
		t.Errorf("stats footer should show 'd/u page' (d/u scrolling is supported):\n%s", out)
	}
}

// Flash messages must take precedence over hints for every footer.

func TestAllFooters_FlashTakesPrecedence(t *testing.T) {
	cases := []struct {
		name string
		make func() string
	}{
		{
			name: "list",
			make: func() string {
				m := loadedModelWith(Session{Project: "p", Slug: "s1", Timestamp: time.Now()})
				m.width = 200
				m.height = 40
				m.flashMsg = "FLASH-LIST"
				return renderListFooter(m)
			},
		},
		{
			name: "detail",
			make: func() string {
				m := loadedModel("a")
				m.mode = modeDetail
				m.detailSession = Session{Slug: "x", Project: "p", Branch: "b", Timestamp: time.Now()}
				m.turns = []turn{{kind: "user", body: "hi"}}
				m.expandedTurns = make(map[int]bool)
				m.width = 200
				m.height = 40
				m.flashMsg = "FLASH-DETAIL"
				return renderDetailFooter(m)
			},
		},
		{
			name: "search",
			make: func() string {
				m := newModel("/d")
				m.mode = modeSearch
				m.searchMode = searchModeResults
				m.searchQuery = "x"
				m.width = 200
				m.height = 40
				m.flashMsg = "FLASH-SEARCH"
				return renderSearchFooter(m)
			},
		},
		{
			name: "project",
			make: func() string {
				m := newModel("/d")
				m.mode = modeProject
				m.projectCWD = "/x/p"
				m.width = 200
				m.height = 40
				m.flashMsg = "FLASH-PROJECT"
				return renderProjectFooter(m)
			},
		},
		{
			name: "rerun",
			make: func() string {
				m := newModel("/d")
				m.mode = modeRerun
				m.detailSession = Session{Slug: "x"}
				m.rerunPrompt = "hi"
				m.rerunCWD = "/x"
				m.width = 200
				m.height = 40
				m.flashMsg = "FLASH-RERUN"
				return renderRerunFooter(m)
			},
		},
		{
			name: "stats",
			make: func() string {
				m := loadedModelWith(Session{Project: "p", Slug: "s1", Timestamp: time.Now()})
				m.mode = modeStats
				m.width = 200
				m.height = 40
				m.flashMsg = "FLASH-STATS"
				return renderStatsFooter(m)
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out := c.make()
			want := "FLASH-" + strings.ToUpper(c.name)
			if !strings.Contains(out, want) {
				t.Errorf("%s footer should show flash %q when set:\n%s", c.name, want, out)
			}
			// When flash is set, the static hints should not be rendered.
			if strings.Contains(out, "q/esc/h/← back") || strings.Contains(out, "q quit") {
				t.Errorf("%s footer should suppress hints while flash is set:\n%s", c.name, out)
			}
		})
	}
}

// Search entry mode footer should show its prompt + apply/cancel hints.
func TestSearchFooter_EntryMode(t *testing.T) {
	m := newModel("/d")
	m.mode = modeSearch
	m.searchMode = searchModeEntry
	m.searchQuery = "abc"
	m.width = 200
	m.height = 40
	out := renderSearchFooter(m)
	if !strings.Contains(out, "abc") {
		t.Errorf("search entry footer should include current query:\n%s", out)
	}
}
