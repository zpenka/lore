package lore

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSearchSessions_EmptyQuery_ReturnsEmpty(t *testing.T) {
	ss := []Session{
		{ID: "1", Slug: "s1", Path: "/tmp/nonexistent"},
	}
	results := searchSessions(ss, "")
	if len(results) != 0 {
		t.Errorf("empty query: got %d results, want 0", len(results))
	}
}

func TestSearchSessions_NoMatches_ReturnsEmpty(t *testing.T) {
	tmpdir := t.TempDir()
	session1 := writeTestSession(t, tmpdir, "sess1.jsonl", `
{"type":"user","sessionId":"1","timestamp":"2026-05-01T10:00:00Z","cwd":"/test","gitBranch":"main","slug":"s1","message":{"content":"hello world"}}
{"type":"assistant","message":{"content":[{"type":"text","text":"goodbye world"}]}}
`)

	ss := []Session{
		{ID: "1", Slug: "s1", Path: session1},
	}
	results := searchSessions(ss, "xyz123notfound")
	if len(results) != 0 {
		t.Errorf("no matches: got %d results, want 0", len(results))
	}
}

func TestSearchSessions_MatchesUserContent_CaseInsensitive(t *testing.T) {
	tmpdir := t.TempDir()
	session1 := writeTestSession(t, tmpdir, "sess1.jsonl", `
{"type":"user","sessionId":"1","timestamp":"2026-05-01T10:00:00Z","cwd":"/test","gitBranch":"main","slug":"s1","message":{"content":"refresh token rotation"}}
{"type":"assistant","message":{"content":[{"type":"text","text":"here is the code"}]}}
`)

	ss := []Session{
		{ID: "1", Slug: "s1", Path: session1},
	}
	results := searchSessions(ss, "REFRESH")
	if len(results) != 1 {
		t.Fatalf("case-insensitive match: got %d results, want 1", len(results))
	}
	if results[0].HitCount != 1 {
		t.Errorf("HitCount = %d, want 1", results[0].HitCount)
	}
}

func TestSearchSessions_MatchesAssistantContent(t *testing.T) {
	tmpdir := t.TempDir()
	session1 := writeTestSession(t, tmpdir, "sess1.jsonl", `
{"type":"user","sessionId":"1","timestamp":"2026-05-01T10:00:00Z","cwd":"/test","gitBranch":"main","slug":"s1","message":{"content":"what is auth?"}}
{"type":"assistant","message":{"content":[{"type":"text","text":"Authentication is the process of verifying identity"}]}}
`)

	ss := []Session{
		{ID: "1", Slug: "s1", Path: session1},
	}
	results := searchSessions(ss, "Authentication")
	if len(results) != 1 {
		t.Fatalf("assistant match: got %d results, want 1", len(results))
	}
	if results[0].HitCount != 1 {
		t.Errorf("HitCount = %d, want 1", results[0].HitCount)
	}
}

func TestSearchSessions_SkipsToolUseBlocks(t *testing.T) {
	tmpdir := t.TempDir()
	session1 := writeTestSession(t, tmpdir, "sess1.jsonl", `
{"type":"user","sessionId":"1","timestamp":"2026-05-01T10:00:00Z","cwd":"/test","gitBranch":"main","slug":"s1","message":{"content":"we need to fix authentication"}}
{"type":"assistant","message":{"content":[{"type":"tool_use","name":"search","input":{"query":"authentication system"}},{"type":"text","text":"found something about authentication"}]}}
`)

	ss := []Session{
		{ID: "1", Slug: "s1", Path: session1},
	}
	results := searchSessions(ss, "authentication")
	// Should find in user content and assistant text block, but NOT in tool_use input
	if len(results) != 1 {
		t.Fatalf("tool skip: got %d results, want 1", len(results))
	}
	if results[0].HitCount != 2 {
		t.Errorf("HitCount = %d, want 2 (user + asst text, not tool)", results[0].HitCount)
	}
}

func TestSearchSessions_SkipsThinkingBlocks(t *testing.T) {
	tmpdir := t.TempDir()
	session1 := writeTestSession(t, tmpdir, "sess1.jsonl", `
{"type":"user","sessionId":"1","timestamp":"2026-05-01T10:00:00Z","cwd":"/test","gitBranch":"main","slug":"s1","message":{"content":"what should I do?"}}
{"type":"assistant","message":{"content":[{"type":"thinking","thinking":"Let me think about authentication"},{"type":"text","text":"here is my answer"}]}}
`)

	ss := []Session{
		{ID: "1", Slug: "s1", Path: session1},
	}
	results := searchSessions(ss, "authentication")
	// Should NOT find in thinking block
	if len(results) != 0 {
		t.Errorf("thinking skip: got %d results, want 0", len(results))
	}
}

func TestSearchSessions_SkipsToolResultOnlyUserEvents(t *testing.T) {
	tmpdir := t.TempDir()
	session1 := writeTestSession(t, tmpdir, "sess1.jsonl", `
{"type":"user","sessionId":"1","timestamp":"2026-05-01T10:00:00Z","cwd":"/test","gitBranch":"main","slug":"s1","message":{"content":"search for tokens"}}
{"type":"assistant","message":{"content":[{"type":"text","text":"ok"}]}}
{"type":"user","message":{"content":[{"type":"tool_result","content":"found token in file.js"}]}}
`)

	ss := []Session{
		{ID: "1", Slug: "s1", Path: session1},
	}
	results := searchSessions(ss, "token")
	// Should find in first user event, NOT in tool_result-only user event
	if len(results) != 1 {
		t.Fatalf("tool-result-only skip: got %d results, want 1", len(results))
	}
	if results[0].HitCount != 1 {
		t.Errorf("HitCount = %d, want 1", results[0].HitCount)
	}
}

func TestSearchSessions_MultipleMatches_CountedCorrectly(t *testing.T) {
	tmpdir := t.TempDir()
	session1 := writeTestSession(t, tmpdir, "sess1.jsonl", `
{"type":"user","sessionId":"1","timestamp":"2026-05-01T10:00:00Z","cwd":"/test","gitBranch":"main","slug":"s1","message":{"content":"refresh the cache"}}
{"type":"assistant","message":{"content":[{"type":"text","text":"ok, refreshing"},{"type":"text","text":"refresh complete"}]}}
`)

	ss := []Session{
		{ID: "1", Slug: "s1", Path: session1},
	}
	results := searchSessions(ss, "refresh")
	if len(results) != 1 {
		t.Fatalf("multiple matches: got %d results, want 1", len(results))
	}
	if results[0].HitCount != 3 {
		t.Errorf("HitCount = %d, want 3", results[0].HitCount)
	}
}

func TestSearchSessions_ResultsSortedByHitCount(t *testing.T) {
	tmpdir := t.TempDir()
	session1 := writeTestSession(t, tmpdir, "sess1.jsonl", `
{"type":"user","sessionId":"1","timestamp":"2026-05-01T10:00:00Z","cwd":"/test","gitBranch":"main","slug":"s1","message":{"content":"token token"}}
`)
	session2 := writeTestSession(t, tmpdir, "sess2.jsonl", `
{"type":"user","sessionId":"2","timestamp":"2026-05-01T11:00:00Z","cwd":"/test","gitBranch":"main","slug":"s2","message":{"content":"token"}}
`)
	session3 := writeTestSession(t, tmpdir, "sess3.jsonl", `
{"type":"user","sessionId":"3","timestamp":"2026-05-01T12:00:00Z","cwd":"/test","gitBranch":"main","slug":"s3","message":{"content":"token token token"}}
`)

	ss := []Session{
		{ID: "1", Slug: "s1", Path: session1, Timestamp: timeFromString("2026-05-01T10:00:00Z")},
		{ID: "2", Slug: "s2", Path: session2, Timestamp: timeFromString("2026-05-01T11:00:00Z")},
		{ID: "3", Slug: "s3", Path: session3, Timestamp: timeFromString("2026-05-01T12:00:00Z")},
	}
	results := searchSessions(ss, "token")
	if len(results) != 3 {
		t.Fatalf("three sessions: got %d results, want 3", len(results))
	}
	// Should be sorted: 3 hits, 2 hits, 1 hit
	if results[0].HitCount != 3 {
		t.Errorf("first result HitCount = %d, want 3", results[0].HitCount)
	}
	if results[1].HitCount != 2 {
		t.Errorf("second result HitCount = %d, want 2", results[1].HitCount)
	}
	if results[2].HitCount != 1 {
		t.Errorf("third result HitCount = %d, want 1", results[2].HitCount)
	}
}

func TestSearchSessions_TiebrokerByTimestampDescending(t *testing.T) {
	tmpdir := t.TempDir()
	session1 := writeTestSession(t, tmpdir, "sess1.jsonl", `
{"type":"user","sessionId":"1","timestamp":"2026-05-01T10:00:00Z","cwd":"/test","gitBranch":"main","slug":"s1","message":{"content":"auth"}}
`)
	session2 := writeTestSession(t, tmpdir, "sess2.jsonl", `
{"type":"user","sessionId":"2","timestamp":"2026-05-01T11:00:00Z","cwd":"/test","gitBranch":"main","slug":"s2","message":{"content":"auth"}}
`)

	ss := []Session{
		{ID: "1", Slug: "s1", Path: session1, Timestamp: timeFromString("2026-05-01T10:00:00Z")},
		{ID: "2", Slug: "s2", Path: session2, Timestamp: timeFromString("2026-05-01T11:00:00Z")},
	}
	results := searchSessions(ss, "auth")
	if len(results) != 2 {
		t.Fatalf("tiebreak: got %d results, want 2", len(results))
	}
	// Same hit count, so should be sorted by timestamp descending (newer first)
	if results[0].Session.ID != "2" {
		t.Errorf("first result ID = %q, want '2' (newer)", results[0].Session.ID)
	}
	if results[1].Session.ID != "1" {
		t.Errorf("second result ID = %q, want '1' (older)", results[1].Session.ID)
	}
}

func TestSearchSessions_SnippetFromFirstMatch(t *testing.T) {
	tmpdir := t.TempDir()
	session1 := writeTestSession(t, tmpdir, "sess1.jsonl", `
{"type":"user","sessionId":"1","timestamp":"2026-05-01T10:00:00Z","cwd":"/test","gitBranch":"main","slug":"s1","message":{"content":"the quick brown fox jumps over the lazy dog"}}
`)

	ss := []Session{
		{ID: "1", Slug: "s1", Path: session1},
	}
	results := searchSessions(ss, "brown")
	if len(results) != 1 {
		t.Fatalf("snippet: got %d results, want 1", len(results))
	}
	snippet := results[0].Snippet
	if !strings.Contains(snippet, "brown") {
		t.Errorf("snippet missing match: %q", snippet)
	}
	if len(snippet) > 80 {
		t.Errorf("snippet length %d exceeds 80", len(snippet))
	}
}

func TestSearchSessions_SnippetCentersMatch(t *testing.T) {
	tmpdir := t.TempDir()
	// A match at position 50+
	session1 := writeTestSession(t, tmpdir, "sess1.jsonl", `
{"type":"user","sessionId":"1","timestamp":"2026-05-01T10:00:00Z","cwd":"/test","gitBranch":"main","slug":"s1","message":{"content":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaabbbbcccc"}}
`)

	ss := []Session{
		{ID: "1", Slug: "s1", Path: session1},
	}
	results := searchSessions(ss, "cccc")
	if len(results) != 1 {
		t.Fatalf("center test: got %d results, want 1", len(results))
	}
	snippet := results[0].Snippet
	if !strings.Contains(snippet, "cccc") {
		t.Errorf("snippet missing match: %q", snippet)
	}
	// If match is past char 40, snippet should be centered
	if len(snippet) < 50 {
		// Snippet was truncated but should still contain the match
		if !strings.Contains(snippet, "cccc") {
			t.Errorf("centered snippet lost the match: %q", snippet)
		}
	}
}

func TestSearchSessions_MultipleTextBlocks(t *testing.T) {
	tmpdir := t.TempDir()
	session1 := writeTestSession(t, tmpdir, "sess1.jsonl", `
{"type":"user","sessionId":"1","timestamp":"2026-05-01T10:00:00Z","cwd":"/test","gitBranch":"main","slug":"s1","message":{"content":[{"type":"text","text":"first block"},{"type":"text","text":"second block"}]}}
`)

	ss := []Session{
		{ID: "1", Slug: "s1", Path: session1},
	}
	results := searchSessions(ss, "block")
	if len(results) != 1 {
		t.Fatalf("multiple blocks: got %d results, want 1", len(results))
	}
	if results[0].HitCount != 2 {
		t.Errorf("HitCount = %d, want 2", results[0].HitCount)
	}
}

// Helpers

func writeTestSession(t *testing.T, tmpdir, filename, content string) string {
	path := filepath.Join(tmpdir, filename)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write session: %v", err)
	}
	return path
}

func TestBuildSnippet_ShortText_NoTruncation(t *testing.T) {
	text := "hello world"
	snippet := buildSnippet(text, "world", 6)
	if snippet != text {
		t.Errorf("short text: got %q, want %q", snippet, text)
	}
	if len(snippet) > 80 {
		t.Errorf("snippet length %d exceeds 80", len(snippet))
	}
}

func TestBuildSnippet_LongText_MatchEarly(t *testing.T) {
	text := "a very short match here and then more content and more content and more content at the end of the string"
	// Match at position 20 (early, < 40)
	snippet := buildSnippet(text, "match", 20)
	if !strings.Contains(snippet, "match") {
		t.Errorf("snippet missing match: %q", snippet)
	}
	if len(snippet) > 80 {
		t.Errorf("snippet length %d exceeds 80", len(snippet))
	}
	// Match is early, should take first 80 chars and end with ...
	if !strings.HasSuffix(snippet, "...") {
		t.Errorf("snippet should end with ...: %q", snippet)
	}
}

func TestBuildSnippet_LongText_MatchLate_Centered(t *testing.T) {
	// Match at position 60+
	text := strings.Repeat("a", 70) + "match" + strings.Repeat("b", 20)
	snippet := buildSnippet(text, "match", 70)
	if !strings.Contains(snippet, "match") {
		t.Errorf("snippet missing match: %q", snippet)
	}
	if len(snippet) > 80 {
		t.Errorf("snippet length %d exceeds 80", len(snippet))
	}
}

func TestBuildSnippet_MatchAtStart(t *testing.T) {
	text := "match is here at the start of a very long text that should be truncated when rendered"
	snippet := buildSnippet(text, "match", 0)
	if !strings.Contains(snippet, "match") {
		t.Errorf("snippet missing match: %q", snippet)
	}
	if len(snippet) > 80 {
		t.Errorf("snippet length %d exceeds 80", len(snippet))
	}
}

func TestBuildSnippet_MatchAtEnd(t *testing.T) {
	text := strings.Repeat("x", 50) + "this is a very long text that ends with match"
	// Match is at position where we need centering
	matchPos := len(text) - 5
	snippet := buildSnippet(text, "match", matchPos)
	if !strings.Contains(snippet, "match") {
		t.Errorf("snippet missing match: %q", snippet)
	}
	if len(snippet) > 80 {
		t.Errorf("snippet length %d exceeds 80", len(snippet))
	}
}

func TestSearchSession_FileNotFound_ReturnsNil(t *testing.T) {
	sess := Session{ID: "1", Slug: "s1", Path: "/nonexistent/file.jsonl"}
	hit := searchSession(sess, "test")
	if hit != nil {
		t.Errorf("nonexistent file: got %v, want nil", hit)
	}
}

func TestCountAndSnippet_NoMatch_ReturnsZero(t *testing.T) {
	count, snippet := countAndSnippet("hello world", "xyz")
	if count != 0 {
		t.Errorf("no match: count = %d, want 0", count)
	}
	if snippet != "" {
		t.Errorf("no match: snippet = %q, want ''", snippet)
	}
}

func TestCountAndSnippet_CaseInsensitive(t *testing.T) {
	count, _ := countAndSnippet("Hello WORLD", "hello")
	if count != 1 {
		t.Errorf("case insensitive: count = %d, want 1", count)
	}
}

func TestCountAndSnippet_MultipleMatches(t *testing.T) {
	count, _ := countAndSnippet("test test test", "test")
	if count != 3 {
		t.Errorf("multiple: count = %d, want 3", count)
	}
}

func TestMatchUserEvent_StringContent(t *testing.T) {
	ev := &rawEvent{
		Type: "user",
		Message: &rawMessage{
			Content: "hello world",
		},
	}
	hits, snippet := matchUserEvent(ev, "hello")
	if hits != 1 {
		t.Errorf("string content: hits = %d, want 1", hits)
	}
	if !strings.Contains(snippet, "hello") {
		t.Errorf("string content: snippet missing match: %q", snippet)
	}
}

func TestMatchUserEvent_ArrayContent(t *testing.T) {
	ev := &rawEvent{
		Type: "user",
		Message: &rawMessage{
			Content: []interface{}{
				map[string]interface{}{"type": "text", "text": "hello"},
				map[string]interface{}{"type": "text", "text": "world"},
			},
		},
	}
	hits, _ := matchUserEvent(ev, "hello")
	if hits != 1 {
		t.Errorf("array content: hits = %d, want 1", hits)
	}
}

func TestMatchAssistantEvent_TextOnly(t *testing.T) {
	ev := &rawEvent{
		Type: "assistant",
		Message: &rawMessage{
			Content: []interface{}{
				map[string]interface{}{"type": "text", "text": "hello there"},
				map[string]interface{}{"type": "tool_use", "name": "Bash", "input": map[string]interface{}{}},
			},
		},
	}
	hits, _ := matchAssistantEvent(ev, "hello")
	if hits != 1 {
		t.Errorf("assistant text: hits = %d, want 1", hits)
	}
}

func TestMatchAssistantEvent_SkipsThinking(t *testing.T) {
	ev := &rawEvent{
		Type: "assistant",
		Message: &rawMessage{
			Content: []interface{}{
				map[string]interface{}{"type": "thinking", "thinking": "hello"},
			},
		},
	}
	hits, _ := matchAssistantEvent(ev, "hello")
	if hits != 0 {
		t.Errorf("skip thinking: hits = %d, want 0", hits)
	}
}

// ----- Task 9: parseSearchQuery tests -----

func TestParseSearchQuery_NoPrefix(t *testing.T) {
	text, filters := parseSearchQuery("hello world")
	if text != "hello world" {
		t.Errorf("text = %q, want %q", text, "hello world")
	}
	if filters.project != "" || filters.branch != "" {
		t.Errorf("unexpected filters: %+v", filters)
	}
}

func TestParseSearchQuery_ProjectPrefix(t *testing.T) {
	text, filters := parseSearchQuery("project:lore refresh token")
	if text != "refresh token" {
		t.Errorf("text = %q, want %q", text, "refresh token")
	}
	if filters.project != "lore" {
		t.Errorf("project = %q, want %q", filters.project, "lore")
	}
}

func TestParseSearchQuery_BranchPrefix(t *testing.T) {
	text, filters := parseSearchQuery("branch:main foo")
	if text != "foo" {
		t.Errorf("text = %q, want %q", text, "foo")
	}
	if filters.branch != "main" {
		t.Errorf("branch = %q, want %q", filters.branch, "main")
	}
}

func TestParseSearchQuery_BothPrefixes(t *testing.T) {
	text, filters := parseSearchQuery("project:lore branch:main query text")
	if text != "query text" {
		t.Errorf("text = %q, want %q", text, "query text")
	}
	if filters.project != "lore" {
		t.Errorf("project = %q, want %q", filters.project, "lore")
	}
	if filters.branch != "main" {
		t.Errorf("branch = %q, want %q", filters.branch, "main")
	}
}

func TestParseSearchQuery_PrefixAtEnd(t *testing.T) {
	text, filters := parseSearchQuery("foo project:bar")
	if filters.project != "bar" {
		t.Errorf("project = %q, want %q", filters.project, "bar")
	}
	if text != "foo" {
		t.Errorf("text = %q, want %q", text, "foo")
	}
}

func TestParseSearchQuery_OnlyPrefixes(t *testing.T) {
	text, filters := parseSearchQuery("project:lore branch:feat/v0.8")
	if text != "" {
		t.Errorf("text = %q, want empty", text)
	}
	if filters.project != "lore" || filters.branch != "feat/v0.8" {
		t.Errorf("unexpected filters: %+v", filters)
	}
}

func TestSearchSessions_ProjectFilter(t *testing.T) {
	// Two sessions on different projects, same content.
	dir := t.TempDir()

	writeSearchFixture(t, dir, "a.jsonl", "lore", "main", "index content")
	writeSearchFixture(t, dir, "b.jsonl", "other", "main", "index content")

	ss := []Session{
		{ID: "a", Project: "lore", Branch: "main", Path: filepath.Join(dir, "a.jsonl")},
		{ID: "b", Project: "other", Branch: "main", Path: filepath.Join(dir, "b.jsonl")},
	}

	text, filters := parseSearchQuery("project:lore index")
	results := searchSessionsFiltered(ss, text, filters)

	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	if results[0].Session.ID != "a" {
		t.Errorf("got session %q, want %q", results[0].Session.ID, "a")
	}
}

func writeSearchFixture(t *testing.T, dir, name, project, branch, text string) {
	t.Helper()
	line := `{"type":"user","sessionId":"x","timestamp":"2026-01-01T00:00:00Z","cwd":"/` + project + `","gitBranch":"` + branch + `","message":{"content":"` + text + `"}}`
	if err := os.WriteFile(filepath.Join(dir, name), []byte(line+"\n"), 0o644); err != nil {
		t.Fatalf("writeSearchFixture: %v", err)
	}
}
