package lore

import (
	"os"
	"path/filepath"
	"testing"
)

// TestSmoke_RealClaudeProjects is a manual sanity check against
// ~/.claude/projects when present. It is skipped in clean environments where
// no transcripts exist (e.g. CI).
func TestSmoke_RealClaudeProjects(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home dir")
	}
	dir := filepath.Join(home, ".claude", "projects")
	if _, err := os.Stat(dir); err != nil {
		t.Skipf("no %s on this machine", dir)
	}
	sessions, _, err := scanSessions(dir)
	if err != nil {
		t.Fatalf("scanSessions(%q): %v", dir, err)
	}
	t.Logf("found %d sessions in %s", len(sessions), dir)
	for i, s := range sessions {
		if i >= 5 {
			break
		}
		t.Logf("  %s  %s  %s  %s", s.Timestamp.Format("2006-01-02 15:04"), s.Project, s.Branch, s.Slug)
	}
}
