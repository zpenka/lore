package lore

import (
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

func (m model) handleStatsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc", "h", "left":
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
