package lore

import (
	"os"
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

// sessionDetailLoadedMsg is dispatched when a single session's turns are loaded.
type sessionDetailLoadedMsg struct {
	turns []turn
	err   error
}

// model is the Bubble Tea state for the session-list and detail panels.
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

	// Detail view state
	mode          int                // modeList or modeDetail
	detailSession Session            // The session being displayed in detail
	turns         []turn             // Parsed turns from the session
	cursorDetail  int                // Cursor position in detail view
	detailErr     error              // Error loading/parsing detail session
	detailLoading bool               // True while loading session content
	expandedTurns map[int]bool       // Tracks which turns are expanded (index -> expanded)
	showThinking  bool               // Whether thinking turns are visible
	justCopied    bool               // Brief flag set after successful copy
	clipboardFn   func(string) error // Dependency-injected clipboard function
}

func newModel(projectsDir string) model {
	return model{
		projectsDir:   projectsDir,
		loading:       true,
		expandedTurns: make(map[int]bool),
		showThinking:  false,
		justCopied:    false,
		clipboardFn:   copyToClipboard, // Default to real implementation
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

// loadSessionDetailCmd loads the full session JSONL and parses turns
func loadSessionDetailCmd(path string) tea.Cmd {
	return func() tea.Msg {
		f, err := os.Open(path)
		if err != nil {
			return sessionDetailLoadedMsg{err: err}
		}
		defer f.Close()
		turns, err := parseTurnsFromJSONL(f)
		return sessionDetailLoadedMsg{turns: turns, err: err}
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

	case sessionDetailLoadedMsg:
		// Detail view session loaded
		m.detailLoading = false
		m.turns = msg.turns
		m.detailErr = msg.err
		m.mode = modeDetail
		m.cursorDetail = 0
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

	// Dispatch based on current mode
	switch m.mode {
	case modeDetail:
		return m.handleDetailKey(msg)
	case modeList:
		return m.handleListKey(msg)
	}

	return m, nil
}

// handleListKey handles keys in list mode
func (m model) handleListKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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
	case "enter", "l", "right":
		// Open session detail
		if !m.loading && len(m.visibleSessions) > 0 {
			m.detailLoading = true
			selected := m.visibleSessions[m.cursor]
			m.detailSession = selected
			return m, loadSessionDetailCmd(selected.Path)
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

// handleDetailKey handles keys in detail mode
func (m model) handleDetailKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc":
		// Return to list mode (preserve cursor)
		m.mode = modeList
		m.turns = nil
		m.cursorDetail = 0
		m.expandedTurns = make(map[int]bool)
		m.showThinking = false
		m.justCopied = false
		return m, nil
	case "j", "down":
		visible := m.visibleTurns()
		if m.cursorDetail < len(visible)-1 {
			m.cursorDetail++
		}
		m.justCopied = false
	case "k", "up":
		if m.cursorDetail > 0 {
			m.cursorDetail--
		}
		m.justCopied = false
	case " ":
		// Expand/collapse tool turn
		visible := m.visibleTurns()
		if m.cursorDetail < len(visible) {
			t := visible[m.cursorDetail]
			if t.kind == "tool" {
				// Find the index in the full turns list
				fullIdx := m.visibleIndexToFullIndex(m.cursorDetail)
				m.expandedTurns[fullIdx] = !m.expandedTurns[fullIdx]
			}
		}
		m.justCopied = false
	case "t":
		// Toggle thinking visibility
		m.showThinking = !m.showThinking
		visible := m.visibleTurns()
		// Clamp cursor if it's on a hidden thinking turn
		if m.cursorDetail >= len(visible) && len(visible) > 0 {
			m.cursorDetail = len(visible) - 1
		}
		m.justCopied = false
	case "y":
		// Copy user turn
		visible := m.visibleTurns()
		copied := false
		if m.cursorDetail < len(visible) {
			t := visible[m.cursorDetail]
			if t.kind == "user" {
				// Copy current user turn
				if err := m.clipboardFn(t.body); err == nil {
					m.justCopied = true
					copied = true
				}
			}
		}
		if !copied {
			// Find most recent user turn before cursor
			for i := m.cursorDetail - 1; i >= 0; i-- {
				if visible[i].kind == "user" {
					if err := m.clipboardFn(visible[i].body); err == nil {
						m.justCopied = true
					}
					break
				}
			}
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

// visibleTurns returns the list of turns filtered by visibility (e.g., thinking blocks).
func (m model) visibleTurns() []turn {
	if m.showThinking {
		return m.turns
	}
	// Filter out thinking turns
	var visible []turn
	for _, t := range m.turns {
		if t.kind != "thinking" {
			visible = append(visible, t)
		}
	}
	return visible
}

// visibleIndexToFullIndex maps a cursor position in visibleTurns to the index in m.turns.
func (m model) visibleIndexToFullIndex(visibleIdx int) int {
	count := 0
	for i, t := range m.turns {
		if m.showThinking || t.kind != "thinking" {
			if count == visibleIdx {
				return i
			}
			count++
		}
	}
	return -1
}
