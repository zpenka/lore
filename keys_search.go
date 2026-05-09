package lore

import (
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
)

func (m model) handleSearchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.searchMode {
	case searchModeEntry:
		return m.handleSearchEntryKey(msg)
	case searchModeResults:
		return m.handleSearchResultsKey(msg)
	}
	return m, nil
}

func (m model) handleSearchEntryKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		// Lazy-open the FTS5 index on first search
		if m.index == nil && m.projectsDir != "" {
			cacheDir, err := indexCacheDir()
			if err == nil {
				idx, err := OpenIndex(filepath.Dir(cacheDir))
				if err == nil {
					idx.Sync(m.projectsDir)
					m.index = idx
				}
			}
		}
		// Try indexed search, fall back to linear scan
		if m.index != nil {
			if hits, err := m.index.Search(m.searchQuery); err == nil && len(hits) > 0 {
				m.searchResults = hits
			} else {
				m.searchResults = searchSessions(m.sessions, m.searchQuery)
			}
		} else {
			m.searchResults = searchSessions(m.sessions, m.searchQuery)
		}
		m.searchMode = searchModeResults
		m.searchCursor = 0
	case tea.KeyEsc:
		m.mode = modeList
		m.searchQuery = ""
		m.searchResults = nil
		m.searchCursor = 0
	case tea.KeyBackspace:
		runes := []rune(m.searchQuery)
		if len(runes) > 0 {
			m.searchQuery = string(runes[:len(runes)-1])
		}
	case tea.KeyRunes:
		m.searchQuery += string(msg.Runes)
	}
	return m, nil
}

func (m model) handleSearchResultsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "j", "k", "d", "u", "g", "G", "down", "up":
		half := m.bodyHeight() / 2
		if half < 1 {
			half = 1
		}
		m.searchCursor = nav(msg.String(), m.searchCursor, len(m.searchResults), half)
		m = m.clampSearchOffsetNow()
	case "enter", "l", "right":
		if len(m.searchResults) > 0 {
			m.detailLoading = true
			selected := m.searchResults[m.searchCursor].Session
			m.detailSession = selected
			return m, loadSessionDetailCmd(selected.Path)
		}
	case "/":
		m.searchMode = searchModeEntry
	case "q", "esc", "h", "left":
		m.mode = modeList
		m.searchQuery = ""
		m.searchResults = nil
		m.searchCursor = 0
		m.searchOffset = 0
	}
	return m, nil
}
