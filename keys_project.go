package lore

import (
	tea "github.com/charmbracelet/bubbletea"
)

func (m model) handleProjectKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc", "h", "left":
		m.mode = modeList
		m.projectCWD = ""
		m.projectSessions = nil
		m.projectCursor = 0
		m.projectOffset = 0
		return m, nil
	case "j", "k", "d", "u", "g", "G", "down", "up":
		half := m.bodyHeight() / 2
		if half < 1 {
			half = 1
		}
		m.projectCursor = nav(msg.String(), m.projectCursor, len(m.projectSessions), half)
		m = m.clampProjectOffsetNow()
	case "enter", "l", "right":
		if len(m.projectSessions) > 0 {
			m.detailLoading = true
			selected := m.projectSessions[m.projectCursor]
			m.detailSession = selected
			return m, loadSessionDetailCmd(selected.Path)
		}
	}
	return m, nil
}
