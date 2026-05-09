package lore

import (
	"fmt"
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
	warnings []string
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

	// warnings are short messages produced during session scan for files
	// that were skipped (unreadable, malformed, no user event). Surfaced
	// in the list header as "(N skipped)" when non-empty.
	warnings []string

	// bookmarks holds the currently bookmarked session IDs (toggled with 'm');
	// bookmarksPath is the JSON file they're persisted to. bookmarkOnly is
	// the binary "show only bookmarked sessions" filter, toggled with 'M'.
	bookmarks     map[string]bool
	bookmarksPath string
	bookmarkOnly  bool

	// Timeline view state. timelineCursor is the highlighted day; dateFilter
	// (when non-zero) restricts the list view to a specific calendar day,
	// set when the user presses enter on a heatmap cell.
	timelineCursor time.Time
	dateFilter     time.Time
}

func newModel(projectsDir string) model {
	bmPath, _ := bookmarksFile() // best-effort; empty path disables persistence
	bookmarks, _ := loadBookmarks(bmPath)
	if bookmarks == nil {
		bookmarks = map[string]bool{}
	}
	return model{
		projectsDir:   projectsDir,
		loading:       true,
		expandedTurns: make(map[int]bool),
		justCopied:    false,
		clipboardFn:   copyToClipboard, // Default to real implementation
		rerunFn:       rerunClaude,     // Default to real implementation
		bookmarks:     bookmarks,
		bookmarksPath: bmPath,
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
		ss, warnings, err := scanSessions(dir)
		return sessionsLoadedMsg{sessions: ss, warnings: warnings, err: err}
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
		m.warnings = msg.warnings
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
	case modeTimeline:
		return m.handleTimelineKey(msg)
	}

	return m, nil
}

func (m *model) applyFilter() {
	sessions := m.sessions
	if strings.TrimSpace(m.filterText) != "" {
		switch m.filterMode {
		case filterModeProject:
			sessions = fuzzyFilterSessions(m.filterText,
				func(s Session) string { return s.Project }, sessions)
		case filterModeBranch:
			sessions = fuzzyFilterSessions(m.filterText,
				func(s Session) string { return s.Branch }, sessions)
		case filterModeFuzzy:
			sessions = fuzzyFilterSessions(m.filterText,
				func(s Session) string { return s.Slug + " " + s.Project + " " + s.Branch },
				sessions)
		}
	}
	if m.bookmarkOnly {
		filtered := make([]Session, 0, len(sessions))
		for _, s := range sessions {
			if m.bookmarks[s.ID] {
				filtered = append(filtered, s)
			}
		}
		sessions = filtered
	}
	if !m.dateFilter.IsZero() {
		want := startOfDay(m.dateFilter)
		filtered := make([]Session, 0, len(sessions))
		for _, s := range sessions {
			if startOfDay(s.Timestamp).Equal(want) {
				filtered = append(filtered, s)
			}
		}
		sessions = filtered
	}
	m.visibleSessions = sessions
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
