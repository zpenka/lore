package lore

import (
	"fmt"
	"strings"
)

func statsBodyLines(m model) (lines []string, cursorLine int) {
	if len(m.statsData) == 0 {
		return []string{" (no sessions)"}, 0
	}
	for i, row := range m.statsData {
		isSelected := (i == m.statsCursor)
		if isSelected {
			cursorLine = len(lines)
		}
		cursor := "  "
		if isSelected {
			cursor = " ►"
		}
		s := row.Session
		st := row.Stats

		mdl := padTrunc(st.Model, 20)
		inTok := formatTokenCount(st.InputTokens)
		outTok := formatTokenCount(st.OutputTokens)
		tokStr := fmt.Sprintf("%s / %s", inTok, outTok)

		var costStr string
		if st.EstimatedCostUSD == 0 && st.Model == "" {
			costStr = "   —"
		} else {
			costStr = fmt.Sprintf("$%.2f", st.EstimatedCostUSD)
		}

		line := fmt.Sprintf("%s %-14s  %-22s  %-20s  %-14s  %s",
			cursor,
			padTrunc(s.Project, 14),
			padTrunc(s.Branch, 22),
			mdl,
			tokStr,
			costStr,
		)
		if isSelected {
			lines = append(lines, selectedStyle.Render(line))
		} else {
			lines = append(lines, line)
		}
	}
	return
}

func renderStatsView(m model) string {
	var b strings.Builder

	b.WriteString(renderStatsHeader(m))
	b.WriteByte('\n')
	b.WriteString(renderDivider(m.width))
	b.WriteByte('\n')

	colHeader := "    project         branch                  model                 in / out        cost"
	b.WriteString(footerStyle.Render(colHeader))
	b.WriteByte('\n')

	body, cursorLine := statsBodyLines(m)
	height := m.bodyHeight() - 1
	if height <= 0 {
		height = 1
	}
	offset := clampOffset(m.statsOffset, cursorLine, len(body), height)
	for _, line := range renderBody(body, offset, height) {
		b.WriteString(line)
		b.WriteByte('\n')
	}

	b.WriteString(renderDivider(m.width))
	b.WriteByte('\n')
	b.WriteString(renderStatsFooter(m))
	b.WriteByte('\n')
	return b.String()
}

func renderStatsHeader(m model) string {
	n := len(m.statsData)
	return headerStyle.Render(fmt.Sprintf(" lore · usage stats · %d session%s", n, plural(n)))
}

func renderStatsFooter(m model) string {
	if m.flashMsg != "" {
		return flashStyle.Render(" " + m.flashMsg)
	}
	return footerStyle.Render(" j/k move   d/u page   g/G top/bottom   ? help   q/esc/h/← back")
}
