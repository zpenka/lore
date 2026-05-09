package lore

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

var heatmapStyles = [4]lipgloss.Style{
	lipgloss.NewStyle().Foreground(lipgloss.Color("236")),
	lipgloss.NewStyle().Foreground(lipgloss.Color("28")),
	lipgloss.NewStyle().Foreground(lipgloss.Color("34")),
	lipgloss.NewStyle().Foreground(lipgloss.Color("46")),
}

const heatmapEmptyGlyph = "░░"
const heatmapFilledGlyph = "██"

func heatmapGlyph(count int) string {
	if count == 0 {
		return heatmapEmptyGlyph
	}
	return heatmapFilledGlyph
}

func renderTimelineHeader(m model) string {
	hm := buildHeatmap(m.sessions, time.Now())
	total := 0
	for r := 0; r < heatmapRows; r++ {
		for c := 0; c < heatmapCols; c++ {
			total += hm.Cells[r][c].Count
		}
	}
	return headerStyle.Render(fmt.Sprintf(" lore · activity heatmap · %d session%s in last 8 weeks",
		total, plural(total)))
}

func renderTimelineFooter(m model) string {
	if m.flashMsg != "" {
		return flashStyle.Render(" " + m.flashMsg)
	}
	hm := buildHeatmap(m.sessions, time.Now())
	count := hm.countOn(m.timelineCursor)
	dateStr := m.timelineCursor.Format("2006-01-02 (Mon)")
	hint := footerStyle.Render(" h/← l/→ move day   enter filter list   ? help   q/esc back")
	info := footerStyle.Render(fmt.Sprintf(" %s   %d session%s", dateStr, count, plural(count)))
	return info + "\n" + hint
}

func renderTimelineView(m model) string {
	var b strings.Builder
	hm := buildHeatmap(m.sessions, time.Now())
	cursorRow, cursorCol, _ := hm.cellOf(m.timelineCursor)

	b.WriteString(renderTimelineHeader(m))
	b.WriteByte('\n')
	b.WriteString(renderDivider(m.width))
	b.WriteByte('\n')

	weekdayLabels := [heatmapRows]string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}
	for row := 0; row < heatmapRows; row++ {
		var line strings.Builder
		line.WriteString("  " + weekdayLabels[row] + "  ")
		for col := 0; col < heatmapCols; col++ {
			cell := hm.Cells[row][col]
			glyph := heatmapGlyph(cell.Count)
			style := heatmapStyles[heatmapBucket(cell.Count)]
			rendered := style.Render(glyph)
			if row == cursorRow && col == cursorCol {
				rendered = selectedStyle.Render("[" + glyph + "]")
			} else {
				rendered = " " + rendered + " "
			}
			line.WriteString(rendered)
		}
		b.WriteString(line.String())
		b.WriteByte('\n')
	}

	b.WriteByte('\n')
	var legend strings.Builder
	legend.WriteString("        less ")
	for i := 0; i < 4; i++ {
		legend.WriteString(heatmapStyles[i].Render(heatmapFilledGlyph))
		legend.WriteString(" ")
	}
	legend.WriteString("more")
	b.WriteString(legend.String())
	b.WriteByte('\n')

	b.WriteString(renderDivider(m.width))
	b.WriteByte('\n')
	b.WriteString(renderTimelineFooter(m))
	b.WriteByte('\n')
	return b.String()
}
