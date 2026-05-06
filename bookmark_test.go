package lore

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// ----- storage layer -----

func TestLoadBookmarks_MissingFileReturnsEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bookmarks.json")
	got, err := loadBookmarks(path)
	if err != nil {
		t.Fatalf("missing file should not be an error, got: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("missing file should return empty map, got: %v", got)
	}
}

func TestLoadBookmarks_RoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "bookmarks.json")
	want := map[string]bool{"sess-a": true, "sess-b": true}
	if err := saveBookmarks(path, want); err != nil {
		t.Fatalf("saveBookmarks: %v", err)
	}
	got, err := loadBookmarks(path)
	if err != nil {
		t.Fatalf("loadBookmarks: %v", err)
	}
	if len(got) != 2 || !got["sess-a"] || !got["sess-b"] {
		t.Errorf("round-trip = %v, want %v", got, want)
	}
}

func TestLoadBookmarks_MalformedJSONReturnsError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bookmarks.json")
	if err := os.WriteFile(path, []byte("not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := loadBookmarks(path); err == nil {
		t.Error("malformed JSON should return an error")
	}
}

func TestSaveBookmarks_OmitsFalseValues(t *testing.T) {
	// Only "true" entries should be persisted; toggling a bookmark off should
	// remove it rather than store an explicit false.
	path := filepath.Join(t.TempDir(), "bookmarks.json")
	if err := saveBookmarks(path, map[string]bool{"a": true, "b": false}); err != nil {
		t.Fatalf("saveBookmarks: %v", err)
	}
	got, err := loadBookmarks(path)
	if err != nil {
		t.Fatalf("loadBookmarks: %v", err)
	}
	if !got["a"] {
		t.Errorf("bookmark 'a' should be loaded as true, got %v", got)
	}
	if got["b"] {
		t.Errorf("bookmark 'b' was stored false; should not be persisted, got %v", got)
	}
}

func TestToggleBookmark_AddsWhenAbsent(t *testing.T) {
	bookmarks := map[string]bool{}
	on := toggleBookmark(bookmarks, "sess-a")
	if !on {
		t.Errorf("toggleBookmark on absent key should return true (now bookmarked)")
	}
	if !bookmarks["sess-a"] {
		t.Errorf("bookmarks map should now contain sess-a, got %v", bookmarks)
	}
}

func TestToggleBookmark_RemovesWhenPresent(t *testing.T) {
	bookmarks := map[string]bool{"sess-a": true}
	on := toggleBookmark(bookmarks, "sess-a")
	if on {
		t.Errorf("toggleBookmark on present key should return false (now unbookmarked)")
	}
	if bookmarks["sess-a"] {
		t.Errorf("bookmarks map should no longer contain sess-a, got %v", bookmarks)
	}
}

// ----- model integration -----

// loadedBookmarkModel wires a temp bookmarks path into a model already loaded
// with the given sessions, so tests can exercise m/M key handling without
// touching the real cache dir.
func loadedBookmarkModel(t *testing.T, sessions ...Session) model {
	t.Helper()
	m := newModel("/d")
	m.sessions = sessions
	m.visibleSessions = sessions
	m.loading = false
	m.bookmarksPath = filepath.Join(t.TempDir(), "bookmarks.json")
	m.bookmarks = map[string]bool{}
	return m
}

func TestModel_PressM_TogglesBookmarkOnSelectedSession(t *testing.T) {
	m := loadedBookmarkModel(t,
		Session{ID: "a", Slug: "session-a", Timestamp: time.Now()},
		Session{ID: "b", Slug: "session-b", Timestamp: time.Now()},
	)
	m.cursor = 1

	next, _ := m.Update(keyMsg("m"))
	m = next.(model)

	if !m.bookmarks["b"] {
		t.Errorf("after pressing m on cursor=1, bookmarks should contain 'b': %v", m.bookmarks)
	}
	if !strings.Contains(strings.ToLower(m.flashMsg), "bookmark") {
		t.Errorf("flash should mention 'bookmark', got: %q", m.flashMsg)
	}

	// Pressing again toggles off.
	next, _ = m.Update(keyMsg("m"))
	m = next.(model)
	if m.bookmarks["b"] {
		t.Errorf("after second m, bookmark 'b' should be cleared: %v", m.bookmarks)
	}
}

func TestModel_PressM_PersistsToDisk(t *testing.T) {
	m := loadedBookmarkModel(t,
		Session{ID: "a", Slug: "session-a", Timestamp: time.Now()},
	)
	next, _ := m.Update(keyMsg("m"))
	m = next.(model)

	got, err := loadBookmarks(m.bookmarksPath)
	if err != nil {
		t.Fatalf("loadBookmarks: %v", err)
	}
	if !got["a"] {
		t.Errorf("expected 'a' to be persisted on disk, got: %v", got)
	}
}

func TestModel_PressCapitalM_FiltersToBookmarksOnly(t *testing.T) {
	m := loadedBookmarkModel(t,
		Session{ID: "a", Slug: "alpha", Timestamp: time.Now()},
		Session{ID: "b", Slug: "bravo", Timestamp: time.Now()},
		Session{ID: "c", Slug: "charlie", Timestamp: time.Now()},
	)
	m.bookmarks = map[string]bool{"a": true, "c": true}

	next, _ := m.Update(keyMsg("M"))
	m = next.(model)

	if len(m.visibleSessions) != 2 {
		t.Fatalf("after M with two bookmarks, visibleSessions = %d, want 2", len(m.visibleSessions))
	}
	for _, s := range m.visibleSessions {
		if s.ID != "a" && s.ID != "c" {
			t.Errorf("visible session %q should not appear (not bookmarked)", s.ID)
		}
	}

	// Pressing M again clears the filter.
	next, _ = m.Update(keyMsg("M"))
	m = next.(model)
	if len(m.visibleSessions) != 3 {
		t.Errorf("after second M, visibleSessions = %d, want 3 (filter cleared)", len(m.visibleSessions))
	}
}

func TestModel_PressCapitalM_NoBookmarks_ShowsFlash(t *testing.T) {
	m := loadedBookmarkModel(t,
		Session{ID: "a", Slug: "alpha", Timestamp: time.Now()},
	)
	next, _ := m.Update(keyMsg("M"))
	m = next.(model)
	if m.flashMsg == "" {
		t.Errorf("M with no bookmarks should set a flash message")
	}
}

func TestModel_DetailMode_PressM_TogglesBookmark(t *testing.T) {
	m := loadedBookmarkModel(t,
		Session{ID: "a", Slug: "session-a", Timestamp: time.Now()},
	)
	m.mode = modeDetail
	m.detailSession = m.sessions[0]
	m.turns = []turn{{kind: "user", body: "hi"}}
	m.expandedTurns = make(map[int]bool)

	next, _ := m.Update(keyMsg("m"))
	m = next.(model)

	if !m.bookmarks["a"] {
		t.Errorf("pressing m in detail should bookmark detailSession 'a': %v", m.bookmarks)
	}
}

// ----- rendering star marker -----

func TestRenderRow_BookmarkedShowsStar(t *testing.T) {
	s := Session{ID: "a", Slug: "alpha", Project: "p", Branch: "main", Timestamp: time.Now()}
	plain := renderRow(s, false, false, 200)
	star := renderRow(s, false, true, 200)
	if strings.Contains(plain, "★") {
		t.Errorf("non-bookmarked row should NOT contain ★: %q", plain)
	}
	if !strings.Contains(star, "★") {
		t.Errorf("bookmarked row should contain ★: %q", star)
	}
}

func TestSearchRow_BookmarkedShowsStar(t *testing.T) {
	m := newModel("/d")
	m.mode = modeSearch
	m.searchMode = searchModeResults
	m.searchQuery = "x"
	m.bookmarks = map[string]bool{"a": true}
	m.searchResults = []SearchHit{
		{Session: Session{ID: "a", Slug: "alpha", Project: "p", Branch: "main", Timestamp: time.Now()}, HitCount: 1, Snippet: "x"},
		{Session: Session{ID: "b", Slug: "bravo", Project: "p", Branch: "main", Timestamp: time.Now()}, HitCount: 1, Snippet: "x"},
	}
	body, _ := searchBodyLines(m)
	if len(body) < 2 {
		t.Fatalf("expected at least 2 body lines, got %d", len(body))
	}
	if !strings.Contains(body[0], "★") {
		t.Errorf("bookmarked search row should contain ★: %q", body[0])
	}
	if strings.Contains(body[2], "★") {
		t.Errorf("non-bookmarked search row should NOT contain ★: %q", body[2])
	}
}

func TestProjectRow_BookmarkedShowsStar(t *testing.T) {
	m := newModel("/d")
	m.mode = modeProject
	m.projectCWD = "/x/p"
	m.bookmarks = map[string]bool{"a": true}
	m.projectSessions = []Session{
		{ID: "a", Slug: "alpha", Project: "p", Branch: "main", Timestamp: time.Now()},
		{ID: "b", Slug: "bravo", Project: "p", Branch: "main", Timestamp: time.Now()},
	}
	body, _ := projectBodyLines(m, time.Now())
	joined := strings.Join(body, "\n")
	if !strings.Contains(joined, "★") {
		t.Errorf("project view should contain ★ for bookmarked session: %v", body)
	}
}

// keyMsg helper used for the bookmark tests; mirrors the helper used in
// model_test.go so we can dispatch single-rune keystrokes through Update.
var _ = tea.KeyMsg{}
