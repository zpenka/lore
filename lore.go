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

// Version is the lore binary version. Bumped manually until we wire releases.
const Version = "0.1.0-phase1"

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
	fs.BoolVar(&showVersion, "v", false, "print version and exit")
	fs.BoolVar(&showVersion, "version", false, "print version and exit")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if showVersion {
		fmt.Fprintln(out, "lore", Version)
		return nil
	}

	dir, err := defaultProjectsDir()
	if err != nil {
		return err
	}

	p := tea.NewProgram(newModel(dir), tea.WithAltScreen())
	_, err = p.Run()
	return err
}

func defaultProjectsDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("locate home dir: %w", err)
	}
	return filepath.Join(home, ".claude", "projects"), nil
}
