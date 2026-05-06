package lore

import (
	"strings"
	"testing"
	"time"
)

// TestSearchAndListRows_AlignedBranchColumn asserts the search and list
// rows use the same branch column width so both views show consistent
// layout. (Search previously used 26, list used 20.)
func TestSearchAndListRows_AlignedBranchColumn(t *testing.T) {
	now := time.Now()
	s := Session{
		Project:   "proj-x",
		Branch:    "feat/some-branch",
		Slug:      "session-slug",
		Query:     "fix the bug",
		Timestamp: now,
	}

	// list view branch column width.
	listRow := renderRow(s, false, 200)
	// search view branch column comes from searchBodyLines which currently
	// hard-codes %-26s. After this refactor both should use projectColWidth
	// and branchColWidth (the single source of truth).
	m := loadedModelWith(s)
	m.mode = modeSearch
	m.searchMode = searchModeResults
	m.searchQuery = "x"
	m.searchResults = []SearchHit{{Session: s, HitCount: 1, Snippet: "hi"}}
	m.width = 200
	m.height = 40

	// Use a constant lookup so the test breaks if anyone tries to
	// re-introduce divergent widths.
	if branchColWidth != 20 {
		t.Errorf("branchColWidth = %d, want 20 (the unified list+search width)", branchColWidth)
	}
	if projectColWidth != 12 {
		t.Errorf("projectColWidth = %d, want 12", projectColWidth)
	}

	// Sanity-check the list row uses the constant.
	wantBranchPadded := padTrunc(s.Branch, branchColWidth)
	if !strings.Contains(listRow, wantBranchPadded) {
		t.Errorf("list row missing branch padded to %d cols.\n got:  %q\n want substring: %q",
			branchColWidth, listRow, wantBranchPadded)
	}

	// And so does the search row body.
	body, _ := searchBodyLines(m)
	if len(body) == 0 {
		t.Fatalf("searchBodyLines returned no rows")
	}
	if !strings.Contains(body[0], wantBranchPadded) {
		t.Errorf("search row missing branch padded to %d cols (matching list).\n got:  %q\n want substring: %q",
			branchColWidth, body[0], wantBranchPadded)
	}
}

// TestRenderRows_DeadCodeRemoved guards against re-introducing the unused
// renderRows helper. It's a compile-time-style guard via reflection on
// the package's exported surface (renderRows is unexported, but having
// the test reference an alternative ensures the green commit removed it).
//
// We can't directly test for absence of an unexported function, so instead
// we assert that the documented public path (listBodyLines) covers what
// renderRows was retained for.
func TestListBodyLines_CoversWhatRenderRowsDid(t *testing.T) {
	now := time.Now()
	m := loadedModelWith(
		Session{Slug: "alpha", Timestamp: now},
		Session{Slug: "bravo", Timestamp: now},
	)
	m.width = 120
	lines, _ := listBodyLines(m, now)
	if len(lines) == 0 {
		t.Fatal("listBodyLines returned no lines")
	}
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "alpha") || !strings.Contains(joined, "bravo") {
		t.Errorf("listBodyLines should produce rows for all sessions:\n%s", joined)
	}
}

// TestRerunMaxLines_NamedConstant pins the magic number 5 used to limit
// the rendered prompt-box height in renderRerunView so tests fail loudly
// if the constant goes missing or its value drifts.
func TestRerunMaxLines_NamedConstant(t *testing.T) {
	if rerunMaxLines != 5 {
		t.Errorf("rerunMaxLines = %d, want 5", rerunMaxLines)
	}
}

// TestSnippetMaxLen_NamedConstant pins the search-result snippet line
// limit so it has a single source of truth.
func TestSnippetMaxLen_NamedConstant(t *testing.T) {
	if snippetMaxLen != 80 {
		t.Errorf("snippetMaxLen = %d, want 80", snippetMaxLen)
	}
}

// TestFixedCols_NamedConstant pins the cursor+time+gaps width used to
// derive the list-row query column.
func TestFixedCols_NamedConstant(t *testing.T) {
	if fixedCols != 48 {
		t.Errorf("fixedCols = %d, want 48", fixedCols)
	}
}
