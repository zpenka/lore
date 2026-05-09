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
	case "j", "down":
		if m.projectCursor < len(m.projectSessions)-1 {
			m.projectCursor++
		}
		m = m.clampProjectOffsetNow()
	case "k", "up":
		if m.projectCursor > 0 {
			m.projectCursor--
		}
		m = m.clampProjectOffsetNow()
	case "d":
		if len(m.projectSessions) > 0 {
			half := m.bodyHeight() / 2
			if half < 1 {
				half = 1
			}
			m.projectCursor += half
			if m.projectCursor >= len(m.projectSessions) {
				m.projectCursor = len(m.projectSessions) - 1
			}
		}
		m = m.clampProjectOffsetNow()
	case "u":
		half := m.bodyHeight() / 2
		if half < 1 {
			half = 1
		}
		m.projectCursor -= half
		if m.projectCursor < 0 {
			m.projectCursor = 0
		}
		m = m.clampProjectOffsetNow()
	case "g":
		m.projectCursor = 0
		m = m.clampProjectOffsetNow()
	case "G":
		if len(m.projectSessions) > 0 {
			m.projectCursor = len(m.projectSessions) - 1
		}
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
