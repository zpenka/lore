package lore

import (
	tea "github.com/charmbracelet/bubbletea"
)

// sessionsLoadedMsg is dispatched when scanSessions finishes.
type sessionsLoadedMsg struct {
	sessions []Session
	err      error
}

// model is the Bubble Tea state for the session-list panel.
type model struct {
	projectsDir string
	sessions    []Session
	cursor      int
	loading     bool
	err         error
	width       int
	height      int
}

func newModel(projectsDir string) model {
	return model{
		projectsDir: projectsDir,
		loading:     true,
	}
}

func (m model) Init() tea.Cmd {
	return loadSessionsCmd(m.projectsDir)
}

func loadSessionsCmd(dir string) tea.Cmd {
	return func() tea.Msg {
		ss, err := scanSessions(dir)
		return sessionsLoadedMsg{sessions: ss, err: err}
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case sessionsLoadedMsg:
		m.loading = false
		m.sessions = msg.sessions
		m.err = msg.err
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.Type == tea.KeyCtrlC {
		return m, tea.Quit
	}
	switch msg.String() {
	case "q":
		return m, tea.Quit
	case "j", "down":
		if !m.loading && m.cursor < len(m.sessions)-1 {
			m.cursor++
		}
	case "k", "up":
		if !m.loading && m.cursor > 0 {
			m.cursor--
		}
	case "g":
		if !m.loading {
			m.cursor = 0
		}
	case "G":
		if !m.loading && len(m.sessions) > 0 {
			m.cursor = len(m.sessions) - 1
		}
	}
	return m, nil
}
