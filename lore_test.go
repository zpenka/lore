package lore

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVersion_IsV090(t *testing.T) {
	var buf bytes.Buffer
	if err := runWith([]string{"-v"}, &buf); err != nil {
		t.Fatalf("runWith(-v): %v", err)
	}
	got := buf.String()
	if !strings.Contains(got, "0.9.0") {
		t.Errorf("version output = %q, want it to contain '0.9.0'", got)
	}
}

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

// ----- resolveProjectsDir tests -----

func TestResolveProjectsDir_FlagTakesPrecedence(t *testing.T) {
	t.Setenv("LORE_PROJECTS_DIR", "/env/dir")
	got, err := resolveProjectsDir("/flag/dir")
	if err != nil {
		t.Fatalf("resolveProjectsDir: %v", err)
	}
	if got != "/flag/dir" {
		t.Errorf("got %q, want '/flag/dir' (flag should beat env)", got)
	}
}

func TestResolveProjectsDir_EnvUsedWhenNoFlag(t *testing.T) {
	t.Setenv("LORE_PROJECTS_DIR", "/env/dir")
	got, err := resolveProjectsDir("")
	if err != nil {
		t.Fatalf("resolveProjectsDir: %v", err)
	}
	if got != "/env/dir" {
		t.Errorf("got %q, want '/env/dir' (env should be used)", got)
	}
}

func TestResolveProjectsDir_DefaultWhenNeitherSet(t *testing.T) {
	t.Setenv("LORE_PROJECTS_DIR", "")
	got, err := resolveProjectsDir("")
	if err != nil {
		t.Fatalf("resolveProjectsDir: %v", err)
	}
	home, _ := os.UserHomeDir()
	want := filepath.Join(home, ".claude", "projects")
	if got != want {
		t.Errorf("got %q, want %q (default)", got, want)
	}
}

func TestRunWith_DirFlag(t *testing.T) {
	// --dir flag should be accepted without error (will just fail to load sessions from nonexistent dir)
	var buf bytes.Buffer
	// We only need to ensure the flag parses; the TUI won't launch because
	// we're passing -v alongside it.
	if err := runWith([]string{"--dir", "/tmp/nonexistent-lore-test", "-v"}, &buf); err != nil {
		t.Fatalf("runWith(--dir, -v): %v", err)
	}
	if !strings.Contains(buf.String(), Version) {
		t.Errorf("output = %q, want version", buf.String())
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

// ----- resolveCacheDir tests -----

func TestResolveCacheDir_EnvTakesPrecedence(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("LORE_CACHE_DIR", tmp)
	got, err := resolveCacheDir()
	if err != nil {
		t.Fatalf("resolveCacheDir: %v", err)
	}
	if got != tmp {
		t.Errorf("got %q, want %q", got, tmp)
	}
}

func TestResolveCacheDir_DefaultWhenEnvUnset(t *testing.T) {
	t.Setenv("LORE_CACHE_DIR", "")
	got, err := resolveCacheDir()
	if err != nil {
		t.Fatalf("resolveCacheDir: %v", err)
	}
	cacheBase, _ := os.UserCacheDir()
	want := filepath.Join(cacheBase, "lore")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestResolveCacheDir_CreatesDirectory(t *testing.T) {
	tmp := t.TempDir()
	newDir := filepath.Join(tmp, "lore-cache-test")
	t.Setenv("LORE_CACHE_DIR", newDir)
	_, err := resolveCacheDir()
	if err != nil {
		t.Fatalf("resolveCacheDir: %v", err)
	}
	if _, statErr := os.Stat(newDir); os.IsNotExist(statErr) {
		t.Error("resolveCacheDir did not create the directory")
	}
}
