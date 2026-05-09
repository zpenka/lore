package lore

import (
	tea "github.com/charmbracelet/bubbletea"
)

func (m model) handleDetailKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc", "h", "left":
		m.mode = modeList
		m.turns = nil
		m.cursorDetail = 0
		m.detailOffset = 0
		m.expandedTurns = make(map[int]bool)
		m.sidechainTurns = nil
		m.justCopied = false
		return m, nil
	case "j", "down":
		visible := m.visibleTurns()
		if m.cursorDetail < len(visible)-1 {
			m.cursorDetail++
		}
		m.justCopied = false
		m = m.clampDetailOffsetNow()
	case "k", "up":
		if m.cursorDetail > 0 {
			m.cursorDetail--
		}
		m.justCopied = false
		m = m.clampDetailOffsetNow()
	case "d":
		visible := m.visibleTurns()
		half := m.bodyHeight() / 2
		if half < 1 {
			half = 1
		}
		m.cursorDetail += half
		if m.cursorDetail >= len(visible) {
			m.cursorDetail = len(visible) - 1
		}
		if m.cursorDetail < 0 {
			m.cursorDetail = 0
		}
		m.justCopied = false
		m = m.clampDetailOffsetNow()
	case "u":
		half := m.bodyHeight() / 2
		if half < 1 {
			half = 1
		}
		m.cursorDetail -= half
		if m.cursorDetail < 0 {
			m.cursorDetail = 0
		}
		m.justCopied = false
		m = m.clampDetailOffsetNow()
	case "g":
		m.cursorDetail = 0
		m.justCopied = false
		m = m.clampDetailOffsetNow()
	case "G":
		visible := m.visibleTurns()
		if len(visible) > 0 {
			m.cursorDetail = len(visible) - 1
		}
		m.justCopied = false
		m = m.clampDetailOffsetNow()
	case " ":
		visible := m.visibleTurns()
		if m.cursorDetail < len(visible) {
			t := visible[m.cursorDetail]
			if t.kind == "tool" {
				fullIdx := m.visibleIndexToFullIndex(m.cursorDetail)
				m.expandedTurns[fullIdx] = !m.expandedTurns[fullIdx]
				if m.expandedTurns[fullIdx] && t.sidechainPath != "" {
					if _, loaded := m.sidechainTurns[fullIdx]; !loaded {
						if scTurns, err := loadSidechainTurns(t.sidechainPath); err == nil {
							if m.sidechainTurns == nil {
								m.sidechainTurns = make(map[int][]turn)
							}
							m.sidechainTurns[fullIdx] = scTurns
						}
					}
				}
			} else {
				m.flashMsg = "space: cursor is not on a tool turn"
			}
		}
		m.justCopied = false
	case "y":
		visible := m.visibleTurns()
		copied := false
		if m.cursorDetail < len(visible) {
			if t := visible[m.cursorDetail]; t.kind == "user" {
				if err := m.clipboardFn(t.body); err == nil {
					m.justCopied = true
					copied = true
				}
			}
		}
		if !copied {
			for i := m.cursorDetail - 1; i >= 0; i-- {
				if visible[i].kind == "user" {
					if err := m.clipboardFn(visible[i].body); err == nil {
						m.justCopied = true
						copied = true
					}
					break
				}
			}
		}
		if !copied {
			m.flashMsg = "y: no user prompt at or before cursor"
		}
	case "r":
		visible := m.visibleTurns()
		if m.cursorDetail < len(visible) {
			t := visible[m.cursorDetail]
			if t.kind == "user" {
				m.mode = modeRerun
				m.rerunPrompt = t.body
				m.rerunCWD = m.detailSession.CWD
			} else {
				m.flashMsg = "r: cursor is not on a user turn"
			}
		}
	case "/":
		m.mode = modeSearch
		m.searchMode = searchModeEntry
		m.searchQuery = ""
		m.searchResults = nil
		m.searchCursor = 0
	case "m":
		on := toggleBookmark(m.bookmarks, m.detailSession.ID)
		if on {
			m.flashMsg = "bookmarked"
		} else {
			m.flashMsg = "unbookmarked"
		}
		if m.bookmarksPath != "" {
			_ = saveBookmarks(m.bookmarksPath, m.bookmarks)
		}
	}
	return m, nil
}
