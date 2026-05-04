package lore

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// fixtureJSONL generates minimal JSONL content for testing.
func fixtureJSONL(sessionID, userText, assistantText string) string {
	return fmt.Sprintf(
		`{"type":"user","sessionId":"%s","timestamp":"2026-01-01T00:00:00Z","cwd":"/tmp","gitBranch":"main","slug":"test-session","message":{"role":"user","content":"%s"}}
{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"%s"}]}}
`, sessionID, userText, assistantText)
}

// writeIndexFixture writes JSONL content to a file and returns its path.
func writeIndexFixture(t *testing.T, dir, filename, content string) string {
	t.Helper()
	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("writeIndexFixture: %v", err)
	}
	return path
}

func TestOpenIndex_CreatesDB(t *testing.T) {
	tmpdir := t.TempDir()
	idx, err := OpenIndex(tmpdir)
	if err != nil {
		t.Fatalf("OpenIndex: %v", err)
	}
	defer idx.Close()

	dbPath := filepath.Join(tmpdir, "index.db")
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Errorf("DB file not created at %s", dbPath)
	}
}

func TestIndex_Sync_IndexesNewFile(t *testing.T) {
	tmpdir := t.TempDir()
	projectsDir := filepath.Join(tmpdir, "projects")
	if err := os.MkdirAll(projectsDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	writeIndexFixture(t, projectsDir, "session1.jsonl",
		fixtureJSONL("sess-001", "hello world", "goodbye world"))

	cacheDir := filepath.Join(tmpdir, "cache")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	idx, err := OpenIndex(cacheDir)
	if err != nil {
		t.Fatalf("OpenIndex: %v", err)
	}
	defer idx.Close()

	if err := idx.Sync(projectsDir); err != nil {
		t.Fatalf("Sync: %v", err)
	}

	// Verify the session was indexed by searching for a term
	hits, err := idx.Search("hello")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(hits) != 1 {
		t.Errorf("after Sync: got %d hits, want 1", len(hits))
	}
}

func TestIndex_Sync_SkipsUnchangedFile(t *testing.T) {
	tmpdir := t.TempDir()
	projectsDir := filepath.Join(tmpdir, "projects")
	if err := os.MkdirAll(projectsDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	path := writeIndexFixture(t, projectsDir, "session1.jsonl",
		fixtureJSONL("sess-001", "unique query term here", "response text"))

	cacheDir := filepath.Join(tmpdir, "cache")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	idx, err := OpenIndex(cacheDir)
	if err != nil {
		t.Fatalf("OpenIndex: %v", err)
	}
	defer idx.Close()

	// First sync — indexes the file
	if err := idx.Sync(projectsDir); err != nil {
		t.Fatalf("first Sync: %v", err)
	}

	// Record the mtime_ns stored after first sync
	var mtimeNS1 int64
	row := idx.db.QueryRow("SELECT mtime_ns FROM session_meta WHERE path = ?", path)
	if err := row.Scan(&mtimeNS1); err != nil {
		t.Fatalf("query session_meta: %v", err)
	}

	// Second sync — should detect file is unchanged and skip
	if err := idx.Sync(projectsDir); err != nil {
		t.Fatalf("second Sync: %v", err)
	}

	var mtimeNS2 int64
	row2 := idx.db.QueryRow("SELECT mtime_ns FROM session_meta WHERE path = ?", path)
	if err := row2.Scan(&mtimeNS2); err != nil {
		t.Fatalf("query session_meta after second sync: %v", err)
	}

	// The mtime should be identical — second sync did not re-index
	if mtimeNS1 != mtimeNS2 {
		t.Errorf("second sync changed mtime_ns: was %d, now %d", mtimeNS1, mtimeNS2)
	}
}

func TestIndex_Sync_ReindexesChangedFile(t *testing.T) {
	tmpdir := t.TempDir()
	projectsDir := filepath.Join(tmpdir, "projects")
	if err := os.MkdirAll(projectsDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	path := writeIndexFixture(t, projectsDir, "session1.jsonl",
		fixtureJSONL("sess-001", "original content", "original response"))

	cacheDir := filepath.Join(tmpdir, "cache")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	idx, err := OpenIndex(cacheDir)
	if err != nil {
		t.Fatalf("OpenIndex: %v", err)
	}
	defer idx.Close()

	// First sync
	if err := idx.Sync(projectsDir); err != nil {
		t.Fatalf("first Sync: %v", err)
	}

	// Verify "original" is found
	hits, err := idx.Search("original")
	if err != nil {
		t.Fatalf("Search before update: %v", err)
	}
	if len(hits) == 0 {
		t.Fatal("expected to find 'original' before update")
	}

	// Modify the file content with a different mtime
	time.Sleep(5 * time.Millisecond) // ensure mtime changes on fast filesystems
	newContent := fixtureJSONL("sess-001", "updated content", "updated response")
	if err := os.WriteFile(path, []byte(newContent), 0644); err != nil {
		t.Fatalf("rewrite file: %v", err)
	}
	// Force a different mtime
	newTime := time.Now().Add(time.Second)
	if err := os.Chtimes(path, newTime, newTime); err != nil {
		t.Fatalf("chtimes: %v", err)
	}

	// Second sync — should re-index because mtime changed
	if err := idx.Sync(projectsDir); err != nil {
		t.Fatalf("second Sync: %v", err)
	}

	// "original" should no longer match
	oldHits, err := idx.Search("original")
	if err != nil {
		t.Fatalf("Search 'original' after update: %v", err)
	}
	if len(oldHits) != 0 {
		t.Errorf("expected 0 hits for 'original' after re-index, got %d", len(oldHits))
	}

	// "updated" should match
	newHits, err := idx.Search("updated")
	if err != nil {
		t.Fatalf("Search 'updated' after update: %v", err)
	}
	if len(newHits) == 0 {
		t.Error("expected 'updated' to be found after re-index")
	}
}

func TestIndex_Sync_RemovesDeletedFile(t *testing.T) {
	tmpdir := t.TempDir()
	projectsDir := filepath.Join(tmpdir, "projects")
	if err := os.MkdirAll(projectsDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	path := writeIndexFixture(t, projectsDir, "session1.jsonl",
		fixtureJSONL("sess-001", "vanishing content", "vanishing response"))

	cacheDir := filepath.Join(tmpdir, "cache")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	idx, err := OpenIndex(cacheDir)
	if err != nil {
		t.Fatalf("OpenIndex: %v", err)
	}
	defer idx.Close()

	// Sync to index the file
	if err := idx.Sync(projectsDir); err != nil {
		t.Fatalf("first Sync: %v", err)
	}

	// Verify "vanishing" is found
	hits, err := idx.Search("vanishing")
	if err != nil {
		t.Fatalf("Search before delete: %v", err)
	}
	if len(hits) == 0 {
		t.Fatal("expected to find 'vanishing' before delete")
	}

	// Delete the file
	if err := os.Remove(path); err != nil {
		t.Fatalf("remove: %v", err)
	}

	// Second sync — should remove the deleted file from index
	if err := idx.Sync(projectsDir); err != nil {
		t.Fatalf("second Sync: %v", err)
	}

	// "vanishing" should no longer be in the index
	hits2, err := idx.Search("vanishing")
	if err != nil {
		t.Fatalf("Search after delete: %v", err)
	}
	if len(hits2) != 0 {
		t.Errorf("expected 0 hits after delete, got %d", len(hits2))
	}
}

func TestIndex_Search_FindsMatch(t *testing.T) {
	tmpdir := t.TempDir()
	projectsDir := filepath.Join(tmpdir, "projects")
	if err := os.MkdirAll(projectsDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	writeIndexFixture(t, projectsDir, "session1.jsonl",
		fixtureJSONL("sess-001", "authentication token refresh", "here is the auth code"))

	cacheDir := filepath.Join(tmpdir, "cache")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	idx, err := OpenIndex(cacheDir)
	if err != nil {
		t.Fatalf("OpenIndex: %v", err)
	}
	defer idx.Close()

	if err := idx.Sync(projectsDir); err != nil {
		t.Fatalf("Sync: %v", err)
	}

	hits, err := idx.Search("authentication")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(hits) != 1 {
		t.Fatalf("expected 1 hit, got %d", len(hits))
	}
	if hits[0].Session.Path != filepath.Join(projectsDir, "session1.jsonl") {
		t.Errorf("hit path = %q, want %q", hits[0].Session.Path, filepath.Join(projectsDir, "session1.jsonl"))
	}
}

func TestIndex_Search_NoMatch(t *testing.T) {
	tmpdir := t.TempDir()
	projectsDir := filepath.Join(tmpdir, "projects")
	if err := os.MkdirAll(projectsDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	writeIndexFixture(t, projectsDir, "session1.jsonl",
		fixtureJSONL("sess-001", "hello world", "goodbye world"))

	cacheDir := filepath.Join(tmpdir, "cache")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	idx, err := OpenIndex(cacheDir)
	if err != nil {
		t.Fatalf("OpenIndex: %v", err)
	}
	defer idx.Close()

	if err := idx.Sync(projectsDir); err != nil {
		t.Fatalf("Sync: %v", err)
	}

	hits, err := idx.Search("xyznotfoundterm")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(hits) != 0 {
		t.Errorf("expected 0 hits, got %d", len(hits))
	}
}

func TestIndex_Search_RankedByRelevance(t *testing.T) {
	tmpdir := t.TempDir()
	projectsDir := filepath.Join(tmpdir, "projects")
	if err := os.MkdirAll(projectsDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	// Session with many mentions of "token" — should rank higher
	writeIndexFixture(t, projectsDir, "session1.jsonl",
		fixtureJSONL("sess-001", "token token token token refresh", "token validation complete"))

	// Session with fewer mentions
	writeIndexFixture(t, projectsDir, "session2.jsonl",
		fixtureJSONL("sess-002", "one token here", "ok"))

	cacheDir := filepath.Join(tmpdir, "cache")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	idx, err := OpenIndex(cacheDir)
	if err != nil {
		t.Fatalf("OpenIndex: %v", err)
	}
	defer idx.Close()

	if err := idx.Sync(projectsDir); err != nil {
		t.Fatalf("Sync: %v", err)
	}

	hits, err := idx.Search("token")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(hits) != 2 {
		t.Fatalf("expected 2 hits, got %d", len(hits))
	}
	// First hit should be the one with more token mentions (sess-001)
	if hits[0].Session.Path != filepath.Join(projectsDir, "session1.jsonl") {
		t.Errorf("top hit should be session1 (more token mentions), got %q", hits[0].Session.Path)
	}
}

func TestIndex_Search_EmptyQuery(t *testing.T) {
	tmpdir := t.TempDir()
	idx, err := OpenIndex(tmpdir)
	if err != nil {
		t.Fatalf("OpenIndex: %v", err)
	}
	defer idx.Close()

	hits, err := idx.Search("")
	if err != nil {
		t.Fatalf("Search empty: %v", err)
	}
	if len(hits) != 0 {
		t.Errorf("empty query: expected 0 hits, got %d", len(hits))
	}
}

func TestIndexCacheDir_ReturnsPath(t *testing.T) {
	dir, err := indexCacheDir()
	if err != nil {
		t.Fatalf("indexCacheDir: %v", err)
	}
	if dir == "" {
		t.Error("indexCacheDir returned empty string")
	}
	// Should end with lore/index.db
	if len(dir) < 13 || dir[len(dir)-13:] != "lore/index.db" {
		t.Errorf("indexCacheDir = %q, want path ending with 'lore/index.db'", dir)
	}
}

func TestExtractSessionText_UserAndAssistant(t *testing.T) {
	content := fixtureJSONL("test-id", "user query about authentication", "assistant response about auth")
	text, err := extractSessionText([]byte(content))
	if err != nil {
		t.Fatalf("extractSessionText: %v", err)
	}
	if text == "" {
		t.Error("extractSessionText returned empty string")
	}
	// Should contain user content
	if !containsSubstr(text, "authentication") {
		t.Errorf("text missing 'authentication': %q", text)
	}
	// Should contain assistant content
	if !containsSubstr(text, "response") {
		t.Errorf("text missing 'response': %q", text)
	}
}

func TestExtractSessionText_SkipsToolUse(t *testing.T) {
	content := `{"type":"user","sessionId":"1","timestamp":"2026-01-01T00:00:00Z","cwd":"/tmp","message":{"content":"search for things"}}
{"type":"assistant","message":{"content":[{"type":"tool_use","name":"search","input":{"query":"secretterm"}},{"type":"text","text":"found results"}]}}
`
	text, err := extractSessionText([]byte(content))
	if err != nil {
		t.Fatalf("extractSessionText: %v", err)
	}
	// Should NOT contain tool_use input
	if containsSubstr(text, "secretterm") {
		t.Errorf("text should not contain tool_use content: %q", text)
	}
	// Should contain assistant text
	if !containsSubstr(text, "found results") {
		t.Errorf("text missing assistant text: %q", text)
	}
}

func TestExtractSessionText_SkipsThinking(t *testing.T) {
	content := `{"type":"user","sessionId":"1","timestamp":"2026-01-01T00:00:00Z","cwd":"/tmp","message":{"content":"what to do?"}}
{"type":"assistant","message":{"content":[{"type":"thinking","thinking":"private thought"},{"type":"text","text":"public answer"}]}}
`
	text, err := extractSessionText([]byte(content))
	if err != nil {
		t.Fatalf("extractSessionText: %v", err)
	}
	if containsSubstr(text, "private thought") {
		t.Errorf("text should not contain thinking block: %q", text)
	}
	if !containsSubstr(text, "public answer") {
		t.Errorf("text missing public answer: %q", text)
	}
}

// containsSubstr is a simple case-insensitive substring check helper.
func containsSubstr(s, sub string) bool {
	if len(sub) == 0 {
		return true
	}
	sLower := toLower(s)
	subLower := toLower(sub)
	return len(sLower) >= len(subLower) && containsStr(sLower, subLower)
}

func toLower(s string) string {
	b := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		b[i] = c
	}
	return string(b)
}

func containsStr(s, sub string) bool {
	if len(sub) == 0 {
		return true
	}
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
