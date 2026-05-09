package lore

import (
	"fmt"
	"strings"
	"time"
)

func listBodyLines(m model, now time.Time) (lines []string, cursorLine int) {
	var lastBucket string
	for i, s := range m.visibleSessions {
		bucket := timeBucket(s.Timestamp, now)
		if bucket != lastBucket {
			lines = append(lines, bucketStyle.Render(" "+bucket))
			lastBucket = bucket
		}
		if i == m.cursor {
			cursorLine = len(lines)
		}
		lines = append(lines, renderRow(s, i == m.cursor, m.bookmarks[s.ID], m.width))
	}
	return
}

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
	b.WriteString(renderListHeader(m))
	b.WriteByte('\n')
	b.WriteString(renderDivider(m.width))
	b.WriteByte('\n')

	body, cursorLine := listBodyLines(m, time.Now())
	height := m.bodyHeight()
	offset := clampOffset(m.listOffset, cursorLine, len(body), height)
	for _, line := range renderBody(body, offset, height) {
		b.WriteString(line)
		b.WriteByte('\n')
	}

	b.WriteString(renderDivider(m.width))
	b.WriteByte('\n')
	b.WriteString(renderListFooter(m))
	if m.detailLoading {
		b.WriteString("  (loading session...)")
	}
	b.WriteByte('\n')
	return b.String()
}

func renderListHeader(m model) string {
	nProjects := countProjects(m.sessions)
	skipped := ""
	if n := len(m.warnings); n > 0 {
		skipped = fmt.Sprintf("   (%d skipped)", n)
	}
	indexStatus := ""
	if m.indexing {
		indexStatus = "   indexing…"
	}
	return headerStyle.Render(fmt.Sprintf(
		" lore · %d session%s across %d project%s%s%s",
		len(m.sessions), plural(len(m.sessions)),
		nProjects, plural(nProjects),
		skipped,
		indexStatus,
	))
}

func renderListFooter(m model) string {
	if m.flashMsg != "" {
		return flashStyle.Render(" " + m.flashMsg)
	}
	if m.filterMode == filterModeProject {
		return footerStyle.Render(fmt.Sprintf(" project filter: %s_  [enter] apply  [esc] cancel", m.filterText))
	}
	if m.filterMode == filterModeBranch {
		return footerStyle.Render(fmt.Sprintf(" branch filter: %s_  [enter] apply  [esc] cancel", m.filterText))
	}
	if m.filterMode == filterModeFuzzy {
		return footerStyle.Render(fmt.Sprintf(" fuzzy filter: %s_  [enter] apply  [esc] cancel", m.filterText))
	}
	if m.filterText != "" && m.appliedFilterMode != filterModeNone {
		switch m.appliedFilterMode {
		case filterModeProject:
			return footerStyle.Render(fmt.Sprintf(" filtered by project: %s   j/k · enter open · esc clear   q quit", m.filterText))
		case filterModeBranch:
			return footerStyle.Render(fmt.Sprintf(" filtered by branch: %s   j/k · enter open · esc clear   q quit", m.filterText))
		case filterModeFuzzy:
			return footerStyle.Render(fmt.Sprintf(" fuzzy filter: %s   j/k · enter open · esc clear   q quit", m.filterText))
		}
	}
	return footerStyle.Render(" j/k move   d/u page   enter open   R resume   / search   p project   b branch   f fuzzy   m bookmark   M bookmarks   T timeline   P project view   S stats   g/G top/bottom   ? help   q quit")
}
