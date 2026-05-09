// Package lore is a TUI for browsing Claude Code session transcripts under
// ~/.claude/projects/.
package lore

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
)

// Version is the lore binary version. Set at build time via ldflags by GoReleaser;
// falls back to the literal below for local builds.
var Version = "0.7.0"

// Run is the entry point used by cmd/lore/main.go.
func Run() error {
	return runWith(os.Args[1:], os.Stdout)
}

// runWith is the testable core of Run. It parses args against an isolated flag
// set (no global state), writes -v/--version output to out, and otherwise
// launches the bubbletea program.
func runWith(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("lore", flag.ContinueOnError)
	fs.SetOutput(out)
	var showVersion bool
	var dirFlag string
	fs.BoolVar(&showVersion, "v", false, "print version and exit")
	fs.BoolVar(&showVersion, "version", false, "print version and exit")
	fs.StringVar(&dirFlag, "dir", "", "path to Claude projects directory (overrides LORE_PROJECTS_DIR and the default)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if showVersion {
		fmt.Fprintln(out, "lore", Version)
		return nil
	}

	dir, err := resolveProjectsDir(dirFlag)
	if err != nil {
		return err
	}

	p := tea.NewProgram(newModel(dir), tea.WithAltScreen())
	_, err = p.Run()
	return err
}

// resolveProjectsDir picks the projects directory with the following precedence:
//  1. dirFlag (from --dir flag) if non-empty
//  2. LORE_PROJECTS_DIR environment variable if set
//  3. defaultProjectsDir() (~/.claude/projects)
func resolveProjectsDir(dirFlag string) (string, error) {
	if dirFlag != "" {
		return dirFlag, nil
	}
	if envDir := os.Getenv("LORE_PROJECTS_DIR"); envDir != "" {
		return envDir, nil
	}
	return defaultProjectsDir()
}

func defaultProjectsDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("locate home dir: %w", err)
	}
	return filepath.Join(home, ".claude", "projects"), nil
}

// resolveCacheDir picks the cache directory with the following precedence:
//  1. LORE_CACHE_DIR environment variable if set and non-empty
//  2. os.UserCacheDir() + "/lore"
//
// The directory is created if it does not exist.
func resolveCacheDir() (string, error) {
	dir := os.Getenv("LORE_CACHE_DIR")
	if dir == "" {
		base, err := os.UserCacheDir()
		if err != nil {
			return "", fmt.Errorf("locate cache dir: %w", err)
		}
		dir = filepath.Join(base, "lore")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create cache dir %q: %w", dir, err)
	}
	return dir, nil
}
