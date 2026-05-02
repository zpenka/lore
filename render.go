package lore

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

var (
	headerStyle   = lipgloss.NewStyle().Bold(true)
	bucketStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Bold(true)
	footerStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	errorStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
)

func (m model) View() string {
	// Dispatch based on current mode
	switch m.mode {
	case modeDetail:
		return renderDetailView(m)
	case modeList:
		return renderListView(m)
	case modeSearch:
		return renderSearchView(m)
	}
	return ""
}

// renderListView renders the session list
func renderListView(m model) string {
	if m.err != nil {
		return errorStyle.Render(fmt.Sprintf(" error: %v", m.err)) + "\n"
	}
	if m.loading {
		return " loading sessions...\n"
	}
	if len(m.sessions) == 0 {
		return fmt.Sprintf(" No sessions found in %s\n", m.projectsDir)
	}

	var b strings.Builder
	b.WriteString(renderHeader(m))
	b.WriteByte('\n')
	b.WriteString(renderDivider(m.width))
	b.WriteByte('\n')
	b.WriteString(renderRows(m, time.Now()))
	b.WriteString(renderDivider(m.width))
	b.WriteByte('\n')
	b.WriteString(renderFooter(m))
	if m.detailLoading {
		b.WriteString(" (loading session...)")
	}
	b.WriteByte('\n')
	return b.String()
}

// renderDetailView renders the session detail
func renderDetailView(m model) string {
	if m.detailErr != nil {
		return errorStyle.Render(fmt.Sprintf(" error: %v", m.detailErr)) + "\n"
	}
	if m.detailLoading {
		return " loading session...\n"
	}

	var b strings.Builder

	// Header: slug · project · branch   YYYY-MM-DD
	dateStr := m.detailSession.Timestamp.Format("2006-01-02")
	headerLine := fmt.Sprintf(" %s · %s · %s   %s",
		m.detailSession.Slug,
		m.detailSession.Project,
		m.detailSession.Branch,
		dateStr,
	)
	b.WriteString(headerStyle.Render(headerLine))
	b.WriteByte('\n')
	b.WriteString(renderDivider(m.width))
	b.WriteByte('\n')

	// Body: turns (filtered by visibility)
	visible := m.visibleTurns()
	if len(visible) == 0 {
		b.WriteString(" (no turns to display)\n")
	} else {
		for i, t := range visible {
			isSelected := (i == m.cursorDetail)
			fullIdx := m.visibleIndexToFullIndex(i)
			b.WriteString(renderDetailTurnLine(t, isSelected, m.expandedTurns[fullIdx], m.width))
			b.WriteByte('\n')
			// Render expanded tool input if expanded
			if t.kind == "tool" && m.expandedTurns[fullIdx] {
				b.WriteString(renderExpandedToolInput(t, m.width))
			}
		}
	}

	b.WriteString(renderDivider(m.width))
	b.WriteByte('\n')
	b.WriteString(renderDetailFooter(m))
	b.WriteByte('\n')

	return b.String()
}

// renderDetailTurnLine renders a single turn as a line in detail view.
func renderDetailTurnLine(t turn, selected bool, expanded bool, width int) string {
	// Format:
	// user     │ <text>
	// asst     │ <text>
	// thinking │ 〰 <text>
	//          │ ▸ <tool>
	var prefix string
	var marker string
	switch t.kind {
	case "user":
		prefix = " user"
		marker = "│"
	case "asst":
		prefix = " asst"
		marker = "│"
	case "thinking":
		prefix = " think"
		marker = "│ 〰"
	case "tool":
		prefix = "      "
		marker = "│ ▸"
	default:
		prefix = "      "
		marker = "│"
	}

	line := fmt.Sprintf("%s %s %s", prefix, marker, t.body)
	if selected {
		return selectedStyle.Render(line)
	}
	return line
}

// renderExpandedToolInput renders the full input JSON for an expanded tool turn.
func renderExpandedToolInput(t turn, width int) string {
	if t.input == nil || len(t.input) == 0 {
		return ""
	}

	var b strings.Builder
	for key, val := range t.input {
		// Format each field on its own line with indentation
		line := fmt.Sprintf("      │   %s: %v", key, val)
		// Truncate to width if necessary
		if len(line) > width {
			line = line[:width-3] + "…"
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return b.String()
}

// renderDetailFooter renders the footer for detail view.
func renderDetailFooter(m model) string {
	thinkingLabel := "thinking"
	if m.showThinking {
		thinkingLabel = "hide thinking"
	}
	copyStatus := ""
	if m.justCopied {
		copyStatus = "  ✓ copied"
	}
	return footerStyle.Render(fmt.Sprintf(" j/k move   space expand   t %s   y copy   q/esc back%s",
		thinkingLabel, copyStatus))
}

func renderHeader(m model) string {
	nProjects := countProjects(m.sessions)
	return headerStyle.Render(fmt.Sprintf(
		" lore · %d session%s across %d project%s",
		len(m.sessions), plural(len(m.sessions)),
		nProjects, plural(nProjects),
	))
}

func renderDivider(width int) string {
	if width < 4 {
		width = 80
	}
	return strings.Repeat("─", width)
}

func renderRows(m model, now time.Time) string {
	var b strings.Builder
	var lastBucket string
	for i, s := range m.visibleSessions {
		bucket := timeBucket(s.Timestamp, now)
		if bucket != lastBucket {
			b.WriteString(bucketStyle.Render(" " + bucket))
			b.WriteByte('\n')
			lastBucket = bucket
		}
		b.WriteString(renderRow(s, i == m.cursor))
		b.WriteByte('\n')
	}
	return b.String()
}

func renderRow(s Session, selected bool) string {
	cursor := "  "
	if selected {
		cursor = " ►"
	}
	row := fmt.Sprintf("%s %s  %-12s  %-26s  %s",
		cursor,
		s.Timestamp.Format("15:04"),
		padTrunc(s.Project, 12),
		padTrunc(s.Branch, 26),
		s.Slug,
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

func renderFooter(m model) string {
	if m.filterMode == filterModeProject {
		// In project filter entry mode
		prompt := fmt.Sprintf("project filter: %s_  [enter] apply  [esc] cancel", m.filterText)
		return footerStyle.Render(" " + prompt)
	}
	if m.filterMode == filterModeBranch {
		// In branch filter entry mode
		prompt := fmt.Sprintf("branch filter: %s_  [enter] apply  [esc] cancel", m.filterText)
		return footerStyle.Render(" " + prompt)
	}
	if m.filterText != "" && m.appliedFilterMode != filterModeNone {
		// Filter is applied, show clear option
		if m.appliedFilterMode == filterModeProject {
			prompt := fmt.Sprintf("filtered by project: %s  [esc] clear", m.filterText)
			return footerStyle.Render(" " + prompt)
		}
		if m.appliedFilterMode == filterModeBranch {
			prompt := fmt.Sprintf("filtered by branch: %s  [esc] clear", m.filterText)
			return footerStyle.Render(" " + prompt)
		}
	}
	// Default footer
	return footerStyle.Render(" j/k move   g/G top/bottom   p project filter   b branch filter   q quit")
}

// padTrunc trims s to max display columns or right-pads it to fit.
// Naive byte-length — fine for ASCII-leaning project names and branches;
// can be tightened later with rivo/uniseg if real-world cases need it.
func padTrunc(s string, max int) string {
	if len(s) > max {
		if max <= 1 {
			return s[:max]
		}
		return s[:max-1] + "…"
	}
	return s + strings.Repeat(" ", max-len(s))
}

// renderSearchView renders the search mode (entry or results)
func renderSearchView(m model) string {
	var b strings.Builder

	// Header
	if m.searchMode == searchModeEntry {
		headerLine := fmt.Sprintf(" search: %s_   [enter] run   [esc] cancel", m.searchQuery)
		b.WriteString(headerStyle.Render(headerLine))
	} else {
		hitWord := "hit"
		if len(m.searchResults) != 1 {
			hitWord = "hits"
		}
		hitCount := 0
		for _, r := range m.searchResults {
			hitCount += r.HitCount
		}
		headerLine := fmt.Sprintf(" search: %s     %d %s across %d session%s",
			m.searchQuery,
			hitCount,
			hitWord,
			len(m.searchResults),
			plural(len(m.searchResults)),
		)
		b.WriteString(headerStyle.Render(headerLine))
	}
	b.WriteByte('\n')

	b.WriteString(renderDivider(m.width))
	b.WriteByte('\n')

	// Body
	if m.searchMode == searchModeEntry {
		// Empty body during entry
	} else if len(m.searchResults) == 0 {
		b.WriteString(" (no matches)\n")
	} else {
		// Render results
		for i, hit := range m.searchResults {
			isSelected := (i == m.searchCursor)
			line := fmt.Sprintf("  %s  %-12s  %-26s  %s",
				hit.Session.Timestamp.Format("15:04"),
				padTrunc(hit.Session.Project, 12),
				padTrunc(hit.Session.Branch, 26),
				hit.Session.Slug,
			)
			if isSelected {
				b.WriteString(selectedStyle.Render(line))
			} else {
				b.WriteString(line)
			}
			b.WriteByte('\n')

			// Snippet line (indented with marker)
			snippetLine := "    ▸ " + hit.Snippet
			if isSelected {
				b.WriteString(selectedStyle.Render(snippetLine))
			} else {
				b.WriteString(snippetLine)
			}
			b.WriteByte('\n')
		}
	}

	b.WriteString(renderDivider(m.width))
	b.WriteByte('\n')

	// Footer
	if m.searchMode == searchModeEntry {
		b.WriteString(footerStyle.Render(" search: " + m.searchQuery + "_   [enter] run   [esc] cancel"))
	} else {
		b.WriteString(footerStyle.Render(" j/k move   enter open   / new search   esc back"))
	}
	b.WriteByte('\n')

	return b.String()
}
