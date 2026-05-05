package lore

import (
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

	got, err := scanSessions(dir)
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
	got, err := scanSessions(dir)
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

	got, err := scanSessions(dir)
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

	got, err := scanSessions(dir)
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

	got, err := scanSessions(dir)
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
