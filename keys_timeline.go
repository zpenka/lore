package lore

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func (m model) handleTimelineKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	hm := buildHeatmap(m.sessions, time.Now())
	switch msg.String() {
	case "q", "esc":
		m.mode = modeList
		return m, nil
	case "h", "left":
		next := m.timelineCursor.AddDate(0, 0, -1)
		if !next.Before(hm.earliestDay()) {
			m.timelineCursor = next
		}
	case "l", "right":
		today := startOfDay(time.Now())
		next := m.timelineCursor.AddDate(0, 0, 1)
		if !next.After(today) {
			m.timelineCursor = next
		}
	case "enter":
		m.dateFilter = m.timelineCursor
		m.applyFilter()
		m.cursor = 0
		m.listOffset = 0
		m.mode = modeList
	}
	return m, nil
}
