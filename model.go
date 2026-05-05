package lore

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// Filter modes
const (
	filterModeNone = iota
	filterModeProject
	filterModeBranch
	filterModeFuzzy
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

	// Stats view state
	statsData   []statsRow // per-session stats rows (computed on enter)
	statsCursor int        // cursor position in stats view
	statsOffset int        // viewport scroll offset for stats view

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

	// Sidechain turns loaded on demand when expanding Agent tool turns
	sidechainTurns map[int][]turn

	// FTS5 search index (nil until first search; fallback to linear scan if nil)
	index *Index
}

func newModel(projectsDir string) model {
	return model{
		projectsDir:   projectsDir,
		loading:       true,
		expandedTurns: make(map[int]bool),
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

func (m model) clampStatsOffsetNow() model {
	h := m.bodyHeight() - 1 // account for the column header line in stats view
	if h <= 0 {
		return m
	}
	body, cursorLine := statsBodyLines(m)
	m.statsOffset = clampOffset(m.statsOffset, cursorLine, len(body), h)
	return m
}

func loadSessionsCmd(dir string) tea.Cmd {
	return func() tea.Msg {
		ss, err := scanSessions(dir)
		return sessionsLoadedMsg{sessions: ss, err: err}
	}
}

func loadSessionDetailCmd(path string) tea.Cmd {
	return func() tea.Msg {
		turns, err := loadSessionTurns(path)
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
		// Claude has exited (or failed to launch). Return to the session
		// list and reload sessions so any new session created by the re-run
		// appears immediately. Surface spawn errors via a flash message.
		m.mode = modeList
		if msg.err != nil {
			m.flashMsg = fmt.Sprintf("re-run failed: %v", msg.err)
		}
		return m, loadSessionsCmd(m.projectsDir)

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
	case modeStats:
		return m.handleStatsKey(msg)
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
	case "d":
		if !m.loading && len(m.visibleSessions) > 0 {
			half := m.bodyHeight() / 2
			if half < 1 {
				half = 1
			}
			m.cursor += half
			if m.cursor >= len(m.visibleSessions) {
				m.cursor = len(m.visibleSessions) - 1
			}
		}
		m = m.clampListOffsetNow()
	case "u":
		if !m.loading {
			half := m.bodyHeight() / 2
			if half < 1 {
				half = 1
			}
			m.cursor -= half
			if m.cursor < 0 {
				m.cursor = 0
			}
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
	case "f":
		if !m.loading {
			m.filterMode = filterModeFuzzy
		}
	case "/":
		if !m.loading {
			m.mode = modeSearch
			m.searchMode = searchModeEntry
			m.searchQuery = ""
			m.searchResults = nil
			m.searchCursor = 0
		}
	case "S":
		if !m.loading {
			m.statsData = computeStatsRows(m.sessions)
			m.statsCursor = 0
			m.statsOffset = 0
			m.mode = modeStats
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
	case "q", "esc", "h", "left":
		m.mode = modeList
		m.turns = nil
		m.cursorDetail = 0
		m.detailOffset = 0
		m.expandedTurns = make(map[int]bool)
		m.sidechainTurns = nil
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
	case "d":
		visible := m.visibleTurns()
		half := m.bodyHeight() / 2
		if half < 1 {
			half = 1
		}
		m.cursorDetail += half
		if m.cursorDetail >= len(visible) {
			m.cursorDetail = len(visible) - 1
		}
		if m.cursorDetail < 0 {
			m.cursorDetail = 0
		}
		m.justCopied = false
		m = m.clampDetailOffsetNow()
	case "u":
		half := m.bodyHeight() / 2
		if half < 1 {
			half = 1
		}
		m.cursorDetail -= half
		if m.cursorDetail < 0 {
			m.cursorDetail = 0
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
				if m.expandedTurns[fullIdx] && t.sidechainPath != "" {
					if _, loaded := m.sidechainTurns[fullIdx]; !loaded {
						if scTurns, err := loadSidechainTurns(t.sidechainPath); err == nil {
							if m.sidechainTurns == nil {
								m.sidechainTurns = make(map[int][]turn)
							}
							m.sidechainTurns[fullIdx] = scTurns
						}
					}
				}
			} else {
				m.flashMsg = "space: cursor is not on a tool turn"
			}
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
		// Lazy-open the FTS5 index on first search
		if m.index == nil && m.projectsDir != "" {
			cacheDir, err := indexCacheDir()
			if err == nil {
				idx, err := OpenIndex(filepath.Dir(cacheDir))
				if err == nil {
					idx.Sync(m.projectsDir)
					m.index = idx
				}
			}
		}
		// Try indexed search, fall back to linear scan
		if m.index != nil {
			if hits, err := m.index.Search(m.searchQuery); err == nil && len(hits) > 0 {
				m.searchResults = hits
			} else {
				m.searchResults = searchSessions(m.sessions, m.searchQuery)
			}
		} else {
			m.searchResults = searchSessions(m.sessions, m.searchQuery)
		}
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
	case "d":
		if len(m.projectSessions) > 0 {
			half := m.bodyHeight() / 2
			if half < 1 {
				half = 1
			}
			m.projectCursor += half
			if m.projectCursor >= len(m.projectSessions) {
				m.projectCursor = len(m.projectSessions) - 1
			}
		}
		m = m.clampProjectOffsetNow()
	case "u":
		half := m.bodyHeight() / 2
		if half < 1 {
			half = 1
		}
		m.projectCursor -= half
		if m.projectCursor < 0 {
			m.projectCursor = 0
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
	case "d":
		if len(m.searchResults) > 0 {
			half := m.bodyHeight() / 2
			if half < 1 {
				half = 1
			}
			m.searchCursor += half
			if m.searchCursor >= len(m.searchResults) {
				m.searchCursor = len(m.searchResults) - 1
			}
		}
		m = m.clampSearchOffsetNow()
	case "u":
		half := m.bodyHeight() / 2
		if half < 1 {
			half = 1
		}
		m.searchCursor -= half
		if m.searchCursor < 0 {
			m.searchCursor = 0
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

// handleStatsKey handles keys in stats mode.
func (m model) handleStatsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc":
		m.mode = modeList
		return m, nil
	case "j", "down":
		if m.statsCursor < len(m.statsData)-1 {
			m.statsCursor++
		}
		m = m.clampStatsOffsetNow()
	case "k", "up":
		if m.statsCursor > 0 {
			m.statsCursor--
		}
		m = m.clampStatsOffsetNow()
	case "g":
		m.statsCursor = 0
		m = m.clampStatsOffsetNow()
	case "G":
		if len(m.statsData) > 0 {
			m.statsCursor = len(m.statsData) - 1
		}
		m = m.clampStatsOffsetNow()
	}
	return m, nil
}

// computeStatsRows iterates sessions, opens each file, and parses token usage.
// Sessions whose files cannot be opened produce a zero-stats row (displayed as dashes).
func computeStatsRows(sessions []Session) []statsRow {
	rows := make([]statsRow, 0, len(sessions))
	for _, s := range sessions {
		row := statsRow{Session: s}
		if f, err := os.Open(s.Path); err == nil {
			if stats, err := parseSessionStats(f); err == nil {
				stats.EstimatedCostUSD = estimateCost(stats)
				row.Stats = stats
			}
			f.Close()
		}
		rows = append(rows, row)
	}
	return rows
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
	filterText := strings.TrimSpace(m.filterText)

	// Empty filter shows all sessions
	if filterText == "" {
		m.visibleSessions = m.sessions
		return
	}

	switch m.filterMode {
	case filterModeProject:
		// Build candidate list of project names
		var projects []string
		for _, s := range m.sessions {
			projects = append(projects, s.Project)
		}
		// Get fuzzy-ranked projects
		rankedProjects := fuzzyFilterCandidates(filterText, projects)
		// Map back to sessions in fuzzy rank order
		projectMap := make(map[string]bool)
		for _, p := range rankedProjects {
			projectMap[p] = true
		}
		m.visibleSessions = nil
		for _, s := range m.sessions {
			if projectMap[s.Project] {
				m.visibleSessions = append(m.visibleSessions, s)
			}
		}
	case filterModeBranch:
		// Build candidate list of branch names
		var branches []string
		for _, s := range m.sessions {
			branches = append(branches, s.Branch)
		}
		// Get fuzzy-ranked branches
		rankedBranches := fuzzyFilterCandidates(filterText, branches)
		// Map back to sessions in fuzzy rank order
		branchMap := make(map[string]bool)
		for _, b := range rankedBranches {
			branchMap[b] = true
		}
		m.visibleSessions = nil
		for _, s := range m.sessions {
			if branchMap[s.Branch] {
				m.visibleSessions = append(m.visibleSessions, s)
			}
		}
	case filterModeFuzzy:
		// Build candidates by concatenating slug + project + branch per session.
		// fuzzy.Find matches against each candidate string; sessions whose
		// combined field string matches are included.
		candidates := make([]string, len(m.sessions))
		for i, s := range m.sessions {
			candidates[i] = s.Slug + " " + s.Project + " " + s.Branch
		}
		matched := fuzzyFilterCandidates(filterText, candidates)
		// Build a set of matched composite strings for O(1) lookup.
		matchSet := make(map[string]bool, len(matched))
		for _, c := range matched {
			matchSet[c] = true
		}
		m.visibleSessions = nil
		for i, s := range m.sessions {
			if matchSet[candidates[i]] {
				m.visibleSessions = append(m.visibleSessions, s)
			}
		}
	default:
		m.visibleSessions = m.sessions
	}
}

// visibleTurns returns the list of turns filtered by visibility.
// Thinking blocks are always filtered out (session files redact their content).
func (m model) visibleTurns() []turn {
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
		if t.kind != "thinking" {
			if count == visibleIdx {
				return i
			}
			count++
		}
	}
	return -1
}
