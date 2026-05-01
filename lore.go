// Package lore is a TUI for browsing Claude Code session transcripts under
// ~/.claude/projects/.
package lore

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
)

// Version is the lore binary version. Bumped manually until we wire releases.
const Version = "0.1.0-phase1"

// Run is the entry point used by cmd/lore/main.go.
func Run() error {
	var showVersion bool
	flag.BoolVar(&showVersion, "v", false, "print version and exit")
	flag.BoolVar(&showVersion, "version", false, "print version and exit")
	flag.Parse()

	if showVersion {
		fmt.Println("lore", Version)
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
