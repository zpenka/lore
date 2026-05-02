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
	b.WriteByte('\n')
	return b.String()
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
