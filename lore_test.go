package lore

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultProjectsDir(t *testing.T) {
	dir, err := defaultProjectsDir()
	if err != nil {
		t.Fatalf("defaultProjectsDir failed: %v", err)
	}
	home, _ := os.UserHomeDir()
	want := filepath.Join(home, ".claude", "projects")
	if dir != want {
		t.Errorf("dir = %q, want %q", dir, want)
	}
}

func TestVersionConstant(t *testing.T) {
	// Just verify Version constant exists and is non-empty
	if Version == "" {
		t.Error("Version constant is empty")
	}
	if Version != "0.1.0-phase1" {
		t.Errorf("Version = %q, expected 0.1.0-phase1", Version)
	}
}
