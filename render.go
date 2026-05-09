package lore

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

var (
	headerStyle     = lipgloss.NewStyle().Bold(true)
	bucketStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	selectedStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Bold(true)
	footerStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	flashStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	errorStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	diffAddStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	diffRemoveStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
)

// chromeLines is the constant number of fixed-height rows around the
// scrollable body in every mode: header + divider + divider + footer.
const chromeLines = 4

// Layout constants shared across list and search rows so both views
// render consistent column widths.
const (
	projectColWidth = 12
	branchColWidth  = 20
	// fixedCols accounts for cursor (2) + space (1) + time (5) + gap (2) +
	// projectColWidth + gap (2) + branchColWidth + gap (2) when the row
	// includes a trailing query column.
	fixedCols = 48

	// rerunMaxLines bounds the prompt-box height in rerun mode.
	rerunMaxLines = 5

	// snippetMaxLen is the search-result snippet character budget.
	snippetMaxLen = 80
)

func (m model) View() string {
	if m.showHelp {
		return renderHelpOverlay(m)
	}

	switch m.mode {
	case modeDetail:
		return renderDetailView(m)
	case modeList:
		return renderListView(m)
	case modeSearch:
		return renderSearchView(m)
	case modeProject:
		return renderProjectView(m, time.Now())
	case modeRerun:
		return renderRerunView(m)
	case modeStats:
		return renderStatsView(m)
	case modeTimeline:
		return renderTimelineView(m)
	}
	return ""
}

// bodyHeight returns the number of body lines to render for the current
// terminal height, or -1 if the height is unknown / too small to scroll.
func (m model) bodyHeight() int {
	if m.height <= chromeLines {
		return -1
	}
	return m.height - chromeLines
}

// renderBody slices lines to fit the available body height starting from offset.
func renderBody(lines []string, offset int, height int) []string {
	if height <= 0 {
		return lines
	}
	return sliceLines(lines, offset, height)
}

func renderDivider(width int) string {
	if width < 4 {
		width = 80
	}
	return strings.Repeat("─", width)
}

func renderRow(s Session, selected, bookmarked bool, width int) string {
	cursor := "  "
	if selected {
		cursor = " ►"
	}
	mark := " "
	if bookmarked {
		mark = "★"
	}
	query := s.Query
	if query == "" {
		query = s.Slug
	}
	queryWidth := width - fixedCols
	if queryWidth < 10 {
		queryWidth = 10
	}
	row := fmt.Sprintf("%s%s %s  %-*s  %-*s  %s",
		cursor,
		mark,
		s.Timestamp.Format("15:04"),
		projectColWidth, padTrunc(s.Project, projectColWidth),
		branchColWidth, padTrunc(s.Branch, branchColWidth),
		padTrunc(query, queryWidth),
	)
	if selected {
		return selectedStyle.Render(row)
	}
	return row
}

func countProjects(ss []Session) int {
	seen := make(map[string]struct{}, len(ss))
	for _, s := range ss {
		seen[s.CWD] = struct{}{}
	}
	return len(seen)
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

func padTrunc(s string, max int) string {
	if len(s) > max {
		if max <= 1 {
			return s[:max]
		}
		return s[:max-1] + "…"
	}
	return s + strings.Repeat(" ", max-len(s))
}

func renderHelpOverlay(m model) string {
	var helpText string

	switch m.mode {
	case modeList:
		helpText = `
 ┌─ List Mode Help ─────────────────────────────────────────────────────────┐
 │                                                                           │
 │  Navigation:                                                              │
 │    j/k, ↑/↓     Move cursor                                              │
 │    d/u          Half-page down/up                                        │
 │    g/G          Jump to top/bottom                                       │
 │    enter, l, →  Open the highlighted session                            │
 │                                                                           │
 │  Filtering:                                                               │
 │    p             Filter to one project (inline)                          │
 │    b             Filter to one branch (inline)                           │
 │    f             Fuzzy filter across slug, project, and branch           │
 │    M             Toggle bookmark-only filter                             │
 │    esc           Clear filter                                            │
 │                                                                           │
 │  Other:                                                                   │
 │    m             Bookmark / unbookmark the selected session              │
 │    P             Open project view for current session's CWD             │
 │    S             Open usage stats panel (token counts + estimated cost)  │
 │    T             Open timeline activity heatmap                          │
 │    /             Enter full-text search                                  │
 │    ?             Show this help overlay                                  │
 │    q             Quit                                                    │
 │                                                                           │
 └─────────────────────────────────────────────────────────────────────────┘
`

	case modeDetail:
		helpText = `
 ┌─ Detail Mode Help ────────────────────────────────────────────────────────┐
 │                                                                            │
 │  Navigation:                                                               │
 │    j/k, ↑/↓     Scroll through turns                                      │
 │    d/u          Half-page down/up                                          │
 │    g/G          Jump to top/bottom                                        │
 │                                                                            │
 │  Turn Actions:                                                             │
 │    space         Expand/collapse tool turn; Agent ⧑ loads sidechain      │
 │    y             Copy the nearest user prompt to clipboard                │
 │    r             Enter re-run mode with the selected user prompt          │
 │    m             Bookmark / unbookmark this session                       │
 │                                                                            │
 │  Other:                                                                    │
 │    /             Enter full-text search                                   │
 │    ?             Show this help overlay                                   │
 │                                                                            │
 │  Return to List:                                                           │
 │    esc, q, h, ←  Back to the session list                                  │
 │                                                                            │
 └────────────────────────────────────────────────────────────────────────┘
`

	case modeSearch:
		helpText = `
 ┌─ Search Mode Help ────────────────────────────────────────────────────────┐
 │                                                                            │
 │  Search Entry:                                                             │
 │    Type         Build search query                                        │
 │    enter        Run linear scan search                                    │
 │    esc          Cancel, return to list                                    │
 │                                                                            │
 │  Search Results:                                                           │
 │    j/k, ↑/↓     Move through results (sorted by hit count)               │
 │    d/u          Half-page down/up                                          │
 │    g/G          Jump to top/bottom                                        │
 │    enter        Open the selected session in detail                       │
 │    /            Re-search (edit query)                                    │
 │    esc, q, h, ← Back to list                                              │
 │    ?            Show this help overlay                                    │
 │                                                                            │
 └────────────────────────────────────────────────────────────────────────┘
`

	case modeProject:
		helpText = `
 ┌─ Project Mode Help ───────────────────────────────────────────────────────┐
 │                                                                            │
 │  Navigation:                                                               │
 │    j/k, ↑/↓     Move within the project's sessions                        │
 │    d/u          Half-page down/up                                          │
 │    g/G          Jump to top/bottom                                        │
 │    enter        Open session detail                                       │
 │                                                                            │
 │  Return to List:                                                           │
 │    esc, q, h, ← Back to session list                                      │
 │    ?            Show this help overlay                                    │
 │                                                                            │
 │  Sessions are grouped by branch (latest branch first).                    │
 │                                                                            │
 └────────────────────────────────────────────────────────────────────────┘
`

	case modeRerun:
		helpText = `
 ┌─ Re-run Mode Help ────────────────────────────────────────────────────────┐
 │                                                                            │
 │  Actions:                                                                  │
 │    enter        Spawn 'claude' with the chosen prompt                     │
 │                 (subprocess owns the TTY; lore exits cleanly on return)    │
 │    esc, q, h, ← Cancel and return to detail view                          │
 │    ?            Show this help overlay                                    │
 │                                                                            │
 └────────────────────────────────────────────────────────────────────────┘
`

	case modeStats:
		helpText = `
 ┌─ Usage Stats Mode Help ───────────────────────────────────────────────────┐
 │                                                                            │
 │  Navigation:                                                               │
 │    j/k, ↑/↓     Move cursor through sessions                              │
 │    g/G          Jump to top/bottom                                        │
 │                                                                            │
 │  Columns: project · branch · model · input tokens · output tokens · cost  │
 │  Token counts use k (thousands) or M (millions) suffix.                   │
 │  Cost is an estimate based on published per-token pricing.                 │
 │                                                                            │
 │  Return to List:                                                           │
 │    esc, q, h, ← Back to session list                                      │
 │    ?            Show this help overlay                                    │
 │                                                                            │
 └────────────────────────────────────────────────────────────────────────┘
`

	case modeTimeline:
		helpText = `
 ┌─ Timeline Mode Help ──────────────────────────────────────────────────────┐
 │                                                                            │
 │  Activity heatmap: 8 weeks × 7 days. Each cell shows the day's            │
 │  session count, shaded by intensity (dim → bright).                       │
 │                                                                            │
 │  Navigation:                                                               │
 │    h, ←          Move cursor one day earlier                              │
 │    l, →          Move cursor one day later                                │
 │                                                                            │
 │  Actions:                                                                  │
 │    enter         Filter list to the highlighted day                       │
 │    esc, q        Back to session list                                     │
 │    ?             Show this help overlay                                   │
 │                                                                            │
 └────────────────────────────────────────────────────────────────────────┘
`

	default:
		helpText = `
 ┌─ Help ────────────────────────────────────────────────────────────────────┐
 │                                                                            │
 │  Press ? in any mode to see mode-specific keybindings.                   │
 │  Any key dismisses this help overlay.                                     │
 │                                                                            │
 └────────────────────────────────────────────────────────────────────────┘
`
	}

	return helpText
}
