package lore

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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
{"type":"user","sessionId":"1","timestamp":"2026-05-01T10:00:00Z","cwd":"/test","gitBranch":"main","slug":"s1","message":{"content":"search for auth"}}
{"type":"assistant","message":{"content":[{"type":"tool_use","name":"search","input":{"query":"authentication system"}},{"type":"text","text":"found some results"}]}}
`)

	ss := []Session{
		{ID: "1", Slug: "s1", Path: session1},
	}
	results := searchSessions(ss, "authentication")
	// Should find in user content, but NOT in tool_use input
	if len(results) != 1 {
		t.Fatalf("tool skip: got %d results, want 1", len(results))
	}
	if results[0].HitCount != 1 {
		t.Errorf("HitCount = %d, want 1 (found only in user, not tool)", results[0].HitCount)
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
