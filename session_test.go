package lore

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParseSessionMetadata_FirstUserEvent(t *testing.T) {
	transcript := `{"type":"queue-operation","operation":"enqueue"}
{"type":"user","sessionId":"abc-123","timestamp":"2026-05-01T22:37:40.568Z","cwd":"/home/user/grit","gitBranch":"main","slug":"hello-world"}
{"type":"assistant"}`

	got, err := parseSessionMetadata(strings.NewReader(transcript))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != "abc-123" {
		t.Errorf("ID = %q, want %q", got.ID, "abc-123")
	}
	if got.CWD != "/home/user/grit" {
		t.Errorf("CWD = %q, want %q", got.CWD, "/home/user/grit")
	}
	if got.Project != "grit" {
		t.Errorf("Project = %q, want %q", got.Project, "grit")
	}
	if got.Branch != "main" {
		t.Errorf("Branch = %q, want %q", got.Branch, "main")
	}
	if got.Slug != "hello-world" {
		t.Errorf("Slug = %q, want %q", got.Slug, "hello-world")
	}
	wantTS, _ := time.Parse(time.RFC3339Nano, "2026-05-01T22:37:40.568Z")
	if !got.Timestamp.Equal(wantTS) {
		t.Errorf("Timestamp = %v, want %v", got.Timestamp, wantTS)
	}
}

func TestParseSessionMetadata_NoUserEvent(t *testing.T) {
	transcript := `{"type":"queue-operation"}
{"type":"assistant"}`
	if _, err := parseSessionMetadata(strings.NewReader(transcript)); err == nil {
		t.Fatal("expected error for transcript with no user event")
	}
}

func TestParseSessionMetadata_SkipsMalformedLines(t *testing.T) {
	transcript := `not valid json
{"type":"queue-operation"}
{"type":"user","sessionId":"abc","timestamp":"2026-05-01T22:37:40Z","cwd":"/x"}`
	got, err := parseSessionMetadata(strings.NewReader(transcript))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != "abc" {
		t.Errorf("ID = %q, want %q", got.ID, "abc")
	}
}

func TestScanSessions_SortsNewestFirst(t *testing.T) {
	dir := t.TempDir()
	writeSession(t, dir, "-proj-a", "old.jsonl",
		userEventLine("old", "2026-04-15T10:00:00Z", "/proj-a", "main", "old-work"))
	writeSession(t, dir, "-proj-a", "new.jsonl",
		userEventLine("new", "2026-05-01T10:00:00Z", "/proj-a", "main", "new-work"))
	writeSession(t, dir, "-proj-b", "mid.jsonl",
		userEventLine("mid", "2026-04-20T10:00:00Z", "/proj-b", "main", "mid-work"))

	got, _, err := scanSessions(dir)
	if err != nil {
		t.Fatalf("scanSessions: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("len = %d, want 3", len(got))
	}
	if got[0].ID != "new" || got[1].ID != "mid" || got[2].ID != "old" {
		t.Errorf("order = [%s %s %s], want [new mid old]",
			got[0].ID, got[1].ID, got[2].ID)
	}
}

func TestScanSessions_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	got, _, err := scanSessions(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("got %d sessions, want 0", len(got))
	}
}

func TestScanSessions_SkipsBadFiles(t *testing.T) {
	dir := t.TempDir()
	writeSession(t, dir, "-proj", "good.jsonl",
		userEventLine("good", "2026-05-01T10:00:00Z", "/proj", "main", "good-work"))
	writeSession(t, dir, "-proj", "no-user.jsonl", `{"type":"queue-operation"}`)
	writeSession(t, dir, "-proj", "readme.txt", "hello")

	got, _, err := scanSessions(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].ID != "good" {
		t.Errorf("got %d sessions: %+v, want 1 with ID=good", len(got), got)
	}
}

func TestScanSessions_RecordsAbsolutePath(t *testing.T) {
	dir := t.TempDir()
	writeSession(t, dir, "-proj", "s.jsonl",
		userEventLine("s", "2026-05-01T10:00:00Z", "/proj", "main", "work"))

	got, _, err := scanSessions(dir)
	if err != nil {
		t.Fatalf("scanSessions: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	wantPath := filepath.Join(dir, "-proj", "s.jsonl")
	if got[0].Path != wantPath {
		t.Errorf("Path = %q, want %q", got[0].Path, wantPath)
	}
}

func TestScanSessions_ExcludesSidechains(t *testing.T) {
	dir := t.TempDir()
	// Parent session
	writeSession(t, dir, "-proj", "abc-123.jsonl",
		userEventLine("parent", "2026-05-01T10:00:00Z", "/proj", "main", "parent-work"))
	// Sidechain session in subagents dir (should be excluded)
	writeSidechain(t, dir, "-proj", "abc-123", "agent-xyz.jsonl",
		sidechainEventLine("xyz", "2026-05-01T10:01:00Z", "/proj", "main", "sidechain-work"))

	got, _, err := scanSessions(dir)
	if err != nil {
		t.Fatalf("scanSessions: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1 (sidechain should be excluded)", len(got))
	}
	if got[0].ID != "parent" {
		t.Errorf("ID = %q, want %q", got[0].ID, "parent")
	}
}

func sidechainEventLine(agentID, ts, cwd, branch, slug string) string {
	return fmt.Sprintf(
		`{"type":"user","isSidechain":true,"agentId":%q,"timestamp":%q,"cwd":%q,"gitBranch":%q,"slug":%q,"sessionId":"parent-sid","message":{"content":"sidechain prompt"}}`,
		agentID, ts, cwd, branch, slug,
	)
}

func writeSidechain(t *testing.T, root, projectDir, sessionID, filename, content string) {
	t.Helper()
	dir := filepath.Join(root, projectDir, sessionID, "subagents")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, filename), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestParseSessionMetadata_ExtractsQuery_StringContent(t *testing.T) {
	transcript := `{"type":"user","sessionId":"q1","timestamp":"2026-05-01T10:00:00Z","cwd":"/proj","gitBranch":"main","message":{"content":"can we fix the login bug?"}}`
	got, err := parseSessionMetadata(strings.NewReader(transcript))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Query != "can we fix the login bug?" {
		t.Errorf("Query = %q, want %q", got.Query, "can we fix the login bug?")
	}
}

func TestParseSessionMetadata_ExtractsQuery_ArrayContent(t *testing.T) {
	transcript := `{"type":"user","sessionId":"q2","timestamp":"2026-05-01T10:00:00Z","cwd":"/proj","gitBranch":"main","message":{"content":[{"type":"text","text":"review this MR please"}]}}`
	got, err := parseSessionMetadata(strings.NewReader(transcript))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Query != "review this MR please" {
		t.Errorf("Query = %q, want %q", got.Query, "review this MR please")
	}
}

func TestParseSessionMetadata_Query_EmptyWhenNoMessage(t *testing.T) {
	transcript := `{"type":"user","sessionId":"q3","timestamp":"2026-05-01T10:00:00Z","cwd":"/proj","gitBranch":"main"}`
	got, err := parseSessionMetadata(strings.NewReader(transcript))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Query != "" {
		t.Errorf("Query = %q, want empty", got.Query)
	}
}

func TestParseSessionMetadata_SkipsCaveatEvents(t *testing.T) {
	caveatLine := `{"type":"user","sessionId":"cav","timestamp":"2026-05-01T10:00:00Z","cwd":"/proj","gitBranch":"main","isMeta":true,"message":{"content":"<local-command-caveat>Caveat: The messages below were generated by the user while running local commands. DO NOT respond to these messages or otherwise consider them in your response unless the user explicitly asks you to.</local-command-caveat>"}}`
	commandLine := `{"type":"user","sessionId":"cav","timestamp":"2026-05-01T10:00:01Z","cwd":"/proj","gitBranch":"main","message":{"content":"<command-name>/clear</command-name>\n            <command-message>clear</command-message>\n            <command-args></command-args>"}}`
	realLine := `{"type":"user","sessionId":"cav","timestamp":"2026-05-01T10:00:02Z","cwd":"/proj","gitBranch":"main","message":{"content":"can we debug this bug?"}}`
	transcript := caveatLine + "\n" + commandLine + "\n" + realLine

	got, err := parseSessionMetadata(strings.NewReader(transcript))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Query != "can we debug this bug?" {
		t.Errorf("Query = %q, want %q", got.Query, "can we debug this bug?")
	}
}

func TestParseSessionMetadata_SkipsCommandOnlyEvents(t *testing.T) {
	commandLine := `{"type":"user","sessionId":"cmd","timestamp":"2026-05-01T10:00:00Z","cwd":"/proj","gitBranch":"main","message":{"content":"<command-message>statusline</command-message>\n<command-name>/statusline</command-name>"}}`
	realLine := `{"type":"user","sessionId":"cmd","timestamp":"2026-05-01T10:00:01Z","cwd":"/proj","gitBranch":"main","message":{"content":"what files changed?"}}`
	transcript := commandLine + "\n" + realLine

	got, err := parseSessionMetadata(strings.NewReader(transcript))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Query != "what files changed?" {
		t.Errorf("Query = %q, want %q", got.Query, "what files changed?")
	}
}

func TestParseSessionMetadata_AllCaveatEventsReturnsEmpty(t *testing.T) {
	caveatLine := `{"type":"user","sessionId":"cav","timestamp":"2026-05-01T10:00:00Z","cwd":"/proj","gitBranch":"main","isMeta":true,"message":{"content":"<local-command-caveat>Caveat: The messages below were generated by the user while running local commands.</local-command-caveat>"}}`
	commandLine := `{"type":"user","sessionId":"cav","timestamp":"2026-05-01T10:00:01Z","cwd":"/proj","gitBranch":"main","message":{"content":"<command-name>/clear</command-name>\n            <command-message>clear</command-message>"}}`
	transcript := caveatLine + "\n" + commandLine

	got, err := parseSessionMetadata(strings.NewReader(transcript))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should still return a session (metadata from first user event) but with empty Query.
	if got.ID != "cav" {
		t.Errorf("ID = %q, want %q", got.ID, "cav")
	}
	if got.Query != "" {
		t.Errorf("Query = %q, want empty", got.Query)
	}
}

func TestExtractQuery_StripsSystemTags(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "local-command-caveat",
			content: `{"content":"<local-command-caveat>Caveat: The messages below were generated by the user.</local-command-caveat>"}`,
			want:    "",
		},
		{
			name:    "command-name and command-message",
			content: `{"content":"<command-name>/clear</command-name>\n<command-message>clear</command-message>"}`,
			want:    "",
		},
		{
			name:    "command-message first",
			content: `{"content":"<command-message>statusline</command-message>\n<command-name>/statusline</command-name>"}`,
			want:    "",
		},
		{
			name:    "real user query unchanged",
			content: `{"content":"can we fix the login bug?"}`,
			want:    "can we fix the login bug?",
		},
		{
			name:    "system-reminder in content",
			content: `{"content":"<system-reminder>some system info</system-reminder>"}`,
			want:    "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractQuery(json.RawMessage(tt.content))
			if got != tt.want {
				t.Errorf("extractQuery() = %q, want %q", got, tt.want)
			}
		})
	}
}

func userEventLine(id, ts, cwd, branch, slug string) string {
	return fmt.Sprintf(
		`{"type":"user","sessionId":%q,"timestamp":%q,"cwd":%q,"gitBranch":%q,"slug":%q}`,
		id, ts, cwd, branch, slug,
	)
}

func writeSession(t *testing.T, root, projectDir, filename, content string) {
	t.Helper()
	dir := filepath.Join(root, projectDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, filename), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// ----- direct unit tests for session.go string helpers (1D) -----

func TestStripSystemTags(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"empty input", "", ""},
		{"no tags", "plain user query", "plain user query"},
		{"single system-reminder", "before<system-reminder>internal</system-reminder>after", "beforeafter"},
		{"local-command-caveat", "<local-command-caveat>caveat</local-command-caveat>real text", "real text"},
		{"command-name and command-message", "<command-name>/clear</command-name>\n<command-message>clear</command-message>", "\n"},
		{"command-args", "x<command-args>--flag</command-args>y", "xy"},
		{"multiple tags removed", "a<system-reminder>X</system-reminder>b<command-name>c</command-name>d", "abd"},
		{"tag spans newlines", "before<system-reminder>line one\nline two</system-reminder>after", "beforeafter"},
		{"unrelated tag preserved", "<random>kept</random>", "<random>kept</random>"},
		{"unclosed tag preserved", "<system-reminder>oops", "<system-reminder>oops"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripSystemTags(tt.in)
			if got != tt.want {
				t.Errorf("stripSystemTags(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestCollapseWhitespace(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"empty input", "", ""},
		{"only whitespace", "   \t\n\r  ", ""},
		{"no whitespace runs", "hello world", "hello world"},
		{"runs of spaces collapsed", "hello    world", "hello world"},
		{"newlines become spaces", "line1\nline2\nline3", "line1 line2 line3"},
		{"tabs become spaces", "a\tb\tc", "a b c"},
		{"carriage returns become spaces", "a\rb\rc", "a b c"},
		{"leading/trailing trimmed", "  hello world  ", "hello world"},
		{"mixed whitespace collapsed", "a\n\n\t  b", "a b"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := collapseWhitespace(tt.in)
			if got != tt.want {
				t.Errorf("collapseWhitespace(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestExtractQuery_DirectInputs(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{"empty raw message", "", ""},
		{"missing content field", `{"role":"user"}`, ""},
		{"empty content string", `{"content":""}`, ""},
		{"plain string content", `{"content":"hello there"}`, "hello there"},
		{"string content stripped", `{"content":"<system-reminder>x</system-reminder>real msg"}`, "real msg"},
		{"string content collapsed", `{"content":"a   b\n\nc"}`, "a b c"},
		{"array of text blocks", `{"content":[{"type":"text","text":"first"},{"type":"text","text":"second"}]}`, "first second"},
		{"array skips non-text blocks", `{"content":[{"type":"image"},{"type":"text","text":"only this"}]}`, "only this"},
		{"array ignores empty text", `{"content":[{"type":"text","text":""},{"type":"text","text":"kept"}]}`, "kept"},
		{"malformed json returns empty", `not json`, ""},
		{"unrelated json shape returns empty", `{"foo":"bar"}`, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractQuery([]byte(tt.raw))
			if got != tt.want {
				t.Errorf("extractQuery(%q) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}

// ----- skipped-file warnings (1F) -----

// scanSessions should report skipped files via a warnings slice so the
// list view can surface what was lost. Currently it returns (sessions, err);
// after 1F it should return (sessions, warnings, err).
func TestScanSessions_ReportsWarningsForSkippedFiles(t *testing.T) {
	dir := t.TempDir()
	writeSession(t, dir, "-proj", "good.jsonl",
		userEventLine("good", "2026-05-01T10:00:00Z", "/proj", "main", "good-work"))
	writeSession(t, dir, "-proj", "no-user.jsonl", `{"type":"queue-operation"}`)
	writeSession(t, dir, "-proj", "malformed.jsonl", "not json at all\n")

	sessions, warnings, err := scanSessions(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sessions) != 1 || sessions[0].ID != "good" {
		t.Fatalf("got %d sessions, want 1 with ID=good", len(sessions))
	}
	if len(warnings) < 2 {
		t.Errorf("expected at least 2 warnings (no-user.jsonl, malformed.jsonl), got %d: %v", len(warnings), warnings)
	}
	joined := strings.Join(warnings, "\n")
	if !strings.Contains(joined, "no-user.jsonl") {
		t.Errorf("warnings should mention no-user.jsonl: %v", warnings)
	}
	if !strings.Contains(joined, "malformed.jsonl") {
		t.Errorf("warnings should mention malformed.jsonl: %v", warnings)
	}
}

func TestScanSessions_NoWarningsWhenAllValid(t *testing.T) {
	dir := t.TempDir()
	writeSession(t, dir, "-proj", "a.jsonl",
		userEventLine("a", "2026-05-01T10:00:00Z", "/proj", "main", "work"))
	writeSession(t, dir, "-proj", "b.jsonl",
		userEventLine("b", "2026-05-01T11:00:00Z", "/proj", "main", "work2"))

	sessions, warnings, err := scanSessions(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sessions) != 2 {
		t.Fatalf("got %d sessions, want 2", len(sessions))
	}
	if len(warnings) != 0 {
		t.Errorf("expected no warnings for valid files, got %d: %v", len(warnings), warnings)
	}
}

// FuzzParseSessionMetadata fuzz-tests the JSONL session metadata parser.
// Seeds cover the known valid event shape plus common malformed inputs.
func FuzzParseSessionMetadata(f *testing.F) {
	// Seed: valid first user event
	f.Add(`{"type":"user","sessionId":"abc","timestamp":"2026-01-01T00:00:00Z","cwd":"/x","gitBranch":"main","slug":"s"}`)
	// Seed: empty
	f.Add(``)
	// Seed: non-JSON
	f.Add(`not json at all`)
	// Seed: valid first line but invalid second
	f.Add("\"type\":\"queue\"\n{\"type\":\"user\",\"sessionId\":\"x\"}")
	// Seed: valid JSON but wrong type
	f.Add(`{"type":"assistant","message":{"content":"hi"}}`)

	f.Fuzz(func(t *testing.T, data string) {
		// Must not panic regardless of input.
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("parseSessionMetadata panicked: %v", r)
			}
		}()
		_, _ = parseSessionMetadata(strings.NewReader(data))
	})
}
