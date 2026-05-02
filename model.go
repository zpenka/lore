package lore

import (
	"os"
	"strings"
	"time"

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
	mode          int                // modeList, modeDetail, or modeSearch
	detailSession Session            // The session being displayed in detail
	turns         []turn             // Parsed turns from the session
	cursorDetail  int                // Cursor position in detail view
	detailErr     error              // Error loading/parsing detail session
	detailLoading bool               // True while loading session content
	expandedTurns map[int]bool       // Tracks which turns are expanded (index -> expanded)
	showThinking  bool               // Whether thinking turns are visible
	justCopied    bool               // Brief flag set after successful copy
	clipboardFn   func(string) error // Dependency-injected clipboard function

	// Search state
	searchMode    int         // searchModeEntry or searchModeResults
	searchQuery   string      // Current search query text
	searchResults []SearchHit // Results from last search
	searchCursor  int         // Cursor position in search results

	// Project view state
	projectCWD      string    // The CWD this view is showing
	projectSessions []Session // Pre-filtered subset by CWD
	projectCursor   int       // Cursor row within the visible flat list

	// Re-run state
	rerunPrompt string                           // The user prompt being re-run
	rerunCWD    string                           // The session's CWD for re-run
	rerunFn     func(prompt, cwd string) tea.Cmd // Dependency-injected re-run hook; returns a tea.Cmd so the exec can be routed through tea.ExecProcess (or a fake in tests).

	// Viewport scroll offsets (one per mode). Updated in the key handlers
	// when the cursor moves; used by the renderers to slice the body.
	listOffset    int
	detailOffset  int
	searchOffset  int
	projectOffset int

	// flashMsg is a transient one-render-cycle hint shown in the footer
	// when a key press did nothing the user could see (e.g. `r` on a
	// non-user turn). Cleared at the start of every keystroke.
	flashMsg string

	// showHelp indicates whether to display the help overlay instead of
	// the normal view. Any key dismisses it.
	showHelp bool
}

func newModel(projectsDir string) model {
	return model{
		projectsDir:   projectsDir,
		loading:       true,
		expandedTurns: make(map[int]bool),
		showThinking:  false,
		justCopied:    false,
		clipboardFn:   copyToClipboard, // Default to real implementation
		rerunFn:       rerunClaude,     // Default to real implementation
	}
}

func (m model) Init() tea.Cmd {
	return loadSessionsCmd(m.projectsDir)
}

// Offset-update helpers, called from key handlers after a cursor move so
// the stored offset stays consistent across renders (without these the
// renderer would re-edge-snap from offset 0 every render and the cursor
// would feel anchored to the bottom edge during k navigation).

func (m model) clampListOffsetNow() model {
	h := m.bodyHeight()
	if h <= 0 {
		return m
	}
	body, cursorLine := listBodyLines(m, time.Now())
	m.listOffset = clampOffset(m.listOffset, cursorLine, len(body), h)
	return m
}

func (m model) clampDetailOffsetNow() model {
	h := m.bodyHeight()
	if h <= 0 {
		return m
	}
	body, cursorLine := detailBodyLines(m)
	m.detailOffset = clampOffset(m.detailOffset, cursorLine, len(body), h)
	return m
}

func (m model) clampSearchOffsetNow() model {
	h := m.bodyHeight()
	if h <= 0 {
		return m
	}
	body, cursorLine := searchBodyLines(m)
	m.searchOffset = clampOffset(m.searchOffset, cursorLine, len(body), h)
	return m
}

func (m model) clampProjectOffsetNow() model {
	h := m.bodyHeight()
	if h <= 0 {
		return m
	}
	body, cursorLine := projectBodyLines(m, time.Now())
	m.projectOffset = clampOffset(m.projectOffset, cursorLine, len(body), h)
	return m
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

	case rerunDoneMsg:
		// Claude has exited (or failed to launch). v1 quits lore so the
		// terminal returns cleanly to the user; they can re-launch lore
		// manually. The error is currently discarded — surfacing it to
		// the user is a follow-up.
		_ = msg.err
		return m, tea.Quit

	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Any keystroke clears the previous one-cycle flash. Handlers are free
	// to set a fresh flashMsg below.
	m.flashMsg = ""

	if msg.Type == tea.KeyCtrlC {
		return m, tea.Quit
	}

	// Handle help overlay: if showing, any key dismisses it (except ctrl-c above).
	if m.showHelp {
		m.showHelp = false
		return m, nil
	}

	// If user presses ?, show help overlay.
	if msg.String() == "?" {
		m.showHelp = true
		return m, nil
	}

	// Dispatch based on current mode
	switch m.mode {
	case modeDetail:
		return m.handleDetailKey(msg)
	case modeList:
		return m.handleListKey(msg)
	case modeSearch:
		return m.handleSearchKey(msg)
	case modeProject:
		return m.handleProjectKey(msg)
	case modeRerun:
		return m.handleRerunKey(msg)
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
		m = m.clampListOffsetNow()
	case "k", "up":
		if !m.loading && m.cursor > 0 {
			m.cursor--
		}
		m = m.clampListOffsetNow()
	case "g":
		if !m.loading {
			m.cursor = 0
		}
		m = m.clampListOffsetNow()
	case "G":
		if !m.loading && len(m.visibleSessions) > 0 {
			m.cursor = len(m.visibleSessions) - 1
		}
		m = m.clampListOffsetNow()
	case "p":
		if !m.loading {
			m.filterMode = filterModeProject
		}
	case "P":
		// Capital P: open project view for the selected session
		if !m.loading && len(m.visibleSessions) > 0 {
			selected := m.visibleSessions[m.cursor]
			m.mode = modeProject
			m.projectCWD = selected.CWD
			// Filter the full session list (not visible list) to this CWD
			m.projectSessions = nil
			for _, s := range m.sessions {
				if s.CWD == selected.CWD {
					m.projectSessions = append(m.projectSessions, s)
				}
			}
			m.projectCursor = 0
		}
	case "b":
		if !m.loading {
			m.filterMode = filterModeBranch
		}
	case "/":
		if !m.loading {
			m.mode = modeSearch
			m.searchMode = searchModeEntry
			m.searchQuery = ""
			m.searchResults = nil
			m.searchCursor = 0
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

// handleDetailKey handles keys in detail mode.
func (m model) handleDetailKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc":
		// Return to list mode (preserve cursor in list)
		m.mode = modeList
		m.turns = nil
		m.cursorDetail = 0
		m.detailOffset = 0
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
		m = m.clampDetailOffsetNow()
	case "k", "up":
		if m.cursorDetail > 0 {
			m.cursorDetail--
		}
		m.justCopied = false
		m = m.clampDetailOffsetNow()
	case "g":
		m.cursorDetail = 0
		m.justCopied = false
		m = m.clampDetailOffsetNow()
	case "G":
		visible := m.visibleTurns()
		if len(visible) > 0 {
			m.cursorDetail = len(visible) - 1
		}
		m.justCopied = false
		m = m.clampDetailOffsetNow()
	case " ":
		visible := m.visibleTurns()
		if m.cursorDetail < len(visible) {
			t := visible[m.cursorDetail]
			if t.kind == "tool" {
				fullIdx := m.visibleIndexToFullIndex(m.cursorDetail)
				m.expandedTurns[fullIdx] = !m.expandedTurns[fullIdx]
			} else {
				m.flashMsg = "space: cursor is not on a tool turn"
			}
		}
		m.justCopied = false
	case "t":
		m.showThinking = !m.showThinking
		visible := m.visibleTurns()
		if m.cursorDetail >= len(visible) && len(visible) > 0 {
			m.cursorDetail = len(visible) - 1
		}
		m.justCopied = false
	case "y":
		visible := m.visibleTurns()
		copied := false
		if m.cursorDetail < len(visible) {
			if t := visible[m.cursorDetail]; t.kind == "user" {
				if err := m.clipboardFn(t.body); err == nil {
					m.justCopied = true
					copied = true
				}
			}
		}
		if !copied {
			for i := m.cursorDetail - 1; i >= 0; i-- {
				if visible[i].kind == "user" {
					if err := m.clipboardFn(visible[i].body); err == nil {
						m.justCopied = true
						copied = true
					}
					break
				}
			}
		}
		if !copied {
			m.flashMsg = "y: no user prompt at or before cursor"
		}
	case "r":
		visible := m.visibleTurns()
		if m.cursorDetail < len(visible) {
			t := visible[m.cursorDetail]
			if t.kind == "user" {
				m.mode = modeRerun
				m.rerunPrompt = t.body
				m.rerunCWD = m.detailSession.CWD
			} else {
				m.flashMsg = "r: cursor is not on a user turn"
			}
		}
	}
	return m, nil
}

// handleSearchKey handles keys in search mode
func (m model) handleSearchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.searchMode {
	case searchModeEntry:
		return m.handleSearchEntryKey(msg)
	case searchModeResults:
		return m.handleSearchResultsKey(msg)
	}
	return m, nil
}

// handleSearchEntryKey handles keys while typing search query
func (m model) handleSearchEntryKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		// Run search
		m.searchResults = searchSessions(m.sessions, m.searchQuery)
		m.searchMode = searchModeResults
		m.searchCursor = 0
	case tea.KeyEsc:
		// Cancel search, return to list
		m.mode = modeList
		m.searchQuery = ""
		m.searchResults = nil
		m.searchCursor = 0
	case tea.KeyBackspace:
		// Remove last rune from query
		runes := []rune(m.searchQuery)
		if len(runes) > 0 {
			m.searchQuery = string(runes[:len(runes)-1])
		}
	case tea.KeyRunes:
		// Append runes to query
		m.searchQuery += string(msg.Runes)
	}
	return m, nil
}

// handleProjectKey handles keys in project mode
func (m model) handleProjectKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc":
		m.mode = modeList
		m.projectCWD = ""
		m.projectSessions = nil
		m.projectCursor = 0
		m.projectOffset = 0
		return m, nil
	case "j", "down":
		if m.projectCursor < len(m.projectSessions)-1 {
			m.projectCursor++
		}
		m = m.clampProjectOffsetNow()
	case "k", "up":
		if m.projectCursor > 0 {
			m.projectCursor--
		}
		m = m.clampProjectOffsetNow()
	case "g":
		m.projectCursor = 0
		m = m.clampProjectOffsetNow()
	case "G":
		if len(m.projectSessions) > 0 {
			m.projectCursor = len(m.projectSessions) - 1
		}
		m = m.clampProjectOffsetNow()
	case "enter", "l", "right":
		if len(m.projectSessions) > 0 {
			m.detailLoading = true
			selected := m.projectSessions[m.projectCursor]
			m.detailSession = selected
			return m, loadSessionDetailCmd(selected.Path)
		}
	}
	return m, nil
}

// handleSearchResultsKey handles keys while viewing search results
func (m model) handleSearchResultsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		if m.searchCursor < len(m.searchResults)-1 {
			m.searchCursor++
		}
		m = m.clampSearchOffsetNow()
	case "k", "up":
		if m.searchCursor > 0 {
			m.searchCursor--
		}
		m = m.clampSearchOffsetNow()
	case "g":
		m.searchCursor = 0
		m = m.clampSearchOffsetNow()
	case "G":
		if len(m.searchResults) > 0 {
			m.searchCursor = len(m.searchResults) - 1
		}
		m = m.clampSearchOffsetNow()
	case "enter", "l", "right":
		if len(m.searchResults) > 0 {
			m.detailLoading = true
			selected := m.searchResults[m.searchCursor].Session
			m.detailSession = selected
			return m, loadSessionDetailCmd(selected.Path)
		}
	case "/":
		m.searchMode = searchModeEntry
	case "esc":
		m.mode = modeList
		m.searchQuery = ""
		m.searchResults = nil
		m.searchCursor = 0
		m.searchOffset = 0
	}
	return m, nil
}

// handleRerunKey handles keys in rerun mode.
//
// On enter we hand off to rerunFn, which returns a tea.Cmd. The default
// rerunClaude wraps tea.ExecProcess: bubbletea suspends, the child
// process owns the terminal, and a rerunDoneMsg fires when it exits. We
// then quit lore in the rerunDoneMsg handler.
func (m model) handleRerunKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		return m, m.rerunFn(m.rerunPrompt, m.rerunCWD)
	case "esc", "q":
		m.mode = modeDetail
		m.rerunPrompt = ""
		m.rerunCWD = ""
		return m, nil
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
