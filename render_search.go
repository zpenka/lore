package lore

import (
	"fmt"
	"strings"
)

func searchBodyLines(m model) (lines []string, cursorLine int) {
	if m.searchMode == searchModeEntry {
		return nil, 0
	}
	if len(m.searchResults) == 0 {
		return []string{" (no matches)"}, 0
	}
	for i, hit := range m.searchResults {
		isSelected := (i == m.searchCursor)
		if isSelected {
			cursorLine = len(lines)
		}
		mark := " "
		if m.bookmarks[hit.Session.ID] {
			mark = "★"
		}
		row := fmt.Sprintf("  %s %s  %-*s  %-*s  %s",
			mark,
			hit.Session.Timestamp.Format("15:04"),
			projectColWidth, padTrunc(hit.Session.Project, projectColWidth),
			branchColWidth, padTrunc(hit.Session.Branch, branchColWidth),
			hit.Session.Slug,
		)
		snippet := "    ▸ " + hit.Snippet
		if isSelected {
			lines = append(lines, selectedStyle.Render(row))
			lines = append(lines, selectedStyle.Render(snippet))
		} else {
			lines = append(lines, row)
			lines = append(lines, snippet)
		}
	}
	return
}

func renderSearchHeader(m model) string {
	if m.searchMode == searchModeEntry {
		return headerStyle.Render(fmt.Sprintf(" search: %s_   [enter] run   [esc] cancel", m.searchQuery))
	}
	hitWord := "hit"
	if len(m.searchResults) != 1 {
		hitWord = "hits"
	}
	hitCount := 0
	for _, r := range m.searchResults {
		hitCount += r.HitCount
	}
	return headerStyle.Render(fmt.Sprintf(" search: %s     %d %s across %d session%s",
		m.searchQuery, hitCount, hitWord,
		len(m.searchResults), plural(len(m.searchResults)),
	))
}

func renderSearchView(m model) string {
	var b strings.Builder

	b.WriteString(renderSearchHeader(m))
	b.WriteByte('\n')
	b.WriteString(renderDivider(m.width))
	b.WriteByte('\n')

	body, cursorLine := searchBodyLines(m)
	height := m.bodyHeight()
	offset := clampOffset(m.searchOffset, cursorLine, len(body), height)
	for _, line := range renderBody(body, offset, height) {
		b.WriteString(line)
		b.WriteByte('\n')
	}

	b.WriteString(renderDivider(m.width))
	b.WriteByte('\n')
	b.WriteString(renderSearchFooter(m))
	b.WriteByte('\n')

	return b.String()
}

func renderSearchFooter(m model) string {
	if m.flashMsg != "" {
		return flashStyle.Render(" " + m.flashMsg)
	}
	if m.searchMode == searchModeEntry {
		return footerStyle.Render(" search: " + m.searchQuery + "_   [enter] run   [esc] cancel")
	}
	return footerStyle.Render(" j/k move   d/u page   enter open   / new search   g/G top/bottom   ? help   q/esc/h/← back")
}
