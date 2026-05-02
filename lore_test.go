package lore

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultProjectsDir(t *testing.T) {
	dir, err := defaultProjectsDir()
	if err != nil {
		t.Fatalf("defaultProjectsDir: %v", err)
	}
	home, _ := os.UserHomeDir()
	want := filepath.Join(home, ".claude", "projects")
	if dir != want {
		t.Errorf("dir = %q, want %q", dir, want)
	}
}

func TestVersionConstant(t *testing.T) {
	if Version == "" {
		t.Fatal("Version constant is empty")
	}
}

func TestRunWith_VersionShortFlag(t *testing.T) {
	var buf bytes.Buffer
	if err := runWith([]string{"-v"}, &buf); err != nil {
		t.Fatalf("runWith(-v): %v", err)
	}
	got := buf.String()
	if !strings.Contains(got, "lore "+Version) {
		t.Errorf("output = %q, want it to contain %q", got, "lore "+Version)
	}
}

func TestRunWith_VersionLongFlag(t *testing.T) {
	var buf bytes.Buffer
	if err := runWith([]string{"--version"}, &buf); err != nil {
		t.Fatalf("runWith(--version): %v", err)
	}
	if !strings.Contains(buf.String(), Version) {
		t.Errorf("output missing version: %q", buf.String())
	}
}

func TestRunWith_UnknownFlag(t *testing.T) {
	var buf bytes.Buffer
	err := runWith([]string{"--definitely-not-a-flag"}, &buf)
	if err == nil {
		t.Fatal("expected error for unknown flag, got nil")
	}
}

// TestRun_VersionPath exercises the public Run() entry point through os.Args
// so the wiring from os.Args / os.Stdout into runWith is covered.
func TestRun_VersionPath(t *testing.T) {
	savedArgs := os.Args
	savedStdout := os.Stdout
	t.Cleanup(func() {
		os.Args = savedArgs
		os.Stdout = savedStdout
	})

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stdout = w
	os.Args = []string{"lore", "-v"}

	runErr := Run()
	w.Close()

	var buf bytes.Buffer
	if _, copyErr := io.Copy(&buf, r); copyErr != nil {
		t.Fatalf("read pipe: %v", copyErr)
	}

	if runErr != nil {
		t.Fatalf("Run(): %v", runErr)
	}
	if !strings.Contains(buf.String(), Version) {
		t.Errorf("Run() stdout = %q, want it to contain %q", buf.String(), Version)
	}
}
