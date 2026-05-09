package lore

import (
	tea "github.com/charmbracelet/bubbletea"
)

func (m model) handleRerunKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		return m, m.rerunFn(m.rerunPrompt, m.rerunCWD)
	case "esc", "q", "h", "left":
		m.mode = modeDetail
		m.rerunPrompt = ""
		m.rerunCWD = ""
		return m, nil
	}
	return m, nil
}
