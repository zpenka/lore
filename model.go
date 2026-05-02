package lore

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// Filter modes
const (
	filterModeNone = iota
	filterModeProject
	filterModeBranch
)

// sessionsLoadedMsg is dispatched when scanSessions finishes.
type sessionsLoadedMsg struct {
	sessions []Session
	err      error
}

// model is the Bubble Tea state for the session-list panel.
type model struct {
	projectsDir       string
	sessions          []Session
	visibleSessions   []Session
	cursor            int
	loading           bool
	err               error
	width             int
	height            int
	filterMode        int
	filterText        string
	appliedFilterMode int // Track which mode filter was applied in (used for display)
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
		m.visibleSessions = msg.sessions
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

	// If in filter entry mode, handle filter-specific keys
	if m.filterMode != filterModeNone {
		return m.handleFilterEntryKey(msg)
	}

	// Handle normal navigation keys
	switch msg.String() {
	case "q":
		return m, tea.Quit
	case "j", "down":
		if !m.loading && m.cursor < len(m.visibleSessions)-1 {
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
		if !m.loading && len(m.visibleSessions) > 0 {
			m.cursor = len(m.visibleSessions) - 1
		}
	case "p":
		if !m.loading {
			m.filterMode = filterModeProject
		}
	case "b":
		if !m.loading {
			m.filterMode = filterModeBranch
		}
	case "esc":
		// Clear filter and restore full list (only when filter is applied)
		if m.appliedFilterMode != filterModeNone {
			m.filterText = ""
			m.visibleSessions = m.sessions
			m.appliedFilterMode = filterModeNone
			m.cursor = 0
		}
	}
	return m, nil
}

func (m model) handleFilterEntryKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		// Apply filter
		m.appliedFilterMode = m.filterMode
		m.applyFilter()
		m.filterMode = filterModeNone
		// Clamp cursor
		if len(m.visibleSessions) == 0 {
			m.cursor = 0
		} else if m.cursor >= len(m.visibleSessions) {
			m.cursor = len(m.visibleSessions) - 1
		}
	case tea.KeyEsc:
		// Cancel filter entry and clear both text and visible list
		m.filterText = ""
		m.visibleSessions = m.sessions
		m.filterMode = filterModeNone
		m.appliedFilterMode = filterModeNone
		m.cursor = 0
	case tea.KeyBackspace:
		// Remove last rune from filter text
		runes := []rune(m.filterText)
		if len(runes) > 0 {
			m.filterText = string(runes[:len(runes)-1])
		}
	case tea.KeyRunes:
		// Append runes to filter text
		m.filterText += string(msg.Runes)
	}
	return m, nil
}

func (m *model) applyFilter() {
	m.visibleSessions = nil
	for _, s := range m.sessions {
		if m.matchesFilter(s) {
			m.visibleSessions = append(m.visibleSessions, s)
		}
	}
}

func (m model) matchesFilter(s Session) bool {
	filter := strings.ToLower(m.filterText)
	switch m.filterMode {
	case filterModeProject:
		return strings.Contains(strings.ToLower(s.Project), filter)
	case filterModeBranch:
		return strings.Contains(strings.ToLower(s.Branch), filter)
	default:
		return true
	}
}
