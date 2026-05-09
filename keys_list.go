package lore

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func (m model) handleListKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.filterMode != filterModeNone {
		return m.handleFilterEntryKey(msg)
	}

	switch msg.String() {
	case "q":
		return m, tea.Quit
	case "j", "down":
		if !m.loading && m.cursor < len(m.visibleSessions)-1 {
			m.cursor++
		}
		m = m.clampListOffsetNow()
	case "k", "up":
		if !m.loading && m.cursor > 0 {
			m.cursor--
		}
		m = m.clampListOffsetNow()
	case "d":
		if !m.loading && len(m.visibleSessions) > 0 {
			half := m.bodyHeight() / 2
			if half < 1 {
				half = 1
			}
			m.cursor += half
			if m.cursor >= len(m.visibleSessions) {
				m.cursor = len(m.visibleSessions) - 1
			}
		}
		m = m.clampListOffsetNow()
	case "u":
		if !m.loading {
			half := m.bodyHeight() / 2
			if half < 1 {
				half = 1
			}
			m.cursor -= half
			if m.cursor < 0 {
				m.cursor = 0
			}
		}
		m = m.clampListOffsetNow()
	case "g":
		if !m.loading {
			m.cursor = 0
		}
		m = m.clampListOffsetNow()
	case "G":
		if !m.loading && len(m.visibleSessions) > 0 {
			m.cursor = len(m.visibleSessions) - 1
		}
		m = m.clampListOffsetNow()
	case "p":
		if !m.loading {
			m.filterMode = filterModeProject
		}
	case "P":
		if !m.loading && len(m.visibleSessions) > 0 {
			selected := m.visibleSessions[m.cursor]
			m.mode = modeProject
			m.projectCWD = selected.CWD
			m.projectSessions = nil
			for _, s := range m.sessions {
				if s.CWD == selected.CWD {
					m.projectSessions = append(m.projectSessions, s)
				}
			}
			m.projectCursor = 0
		}
	case "b":
		if !m.loading {
			m.filterMode = filterModeBranch
		}
	case "f":
		if !m.loading {
			m.filterMode = filterModeFuzzy
		}
	case "/":
		if !m.loading {
			m.mode = modeSearch
			m.searchMode = searchModeEntry
			m.searchQuery = ""
			m.searchResults = nil
			m.searchCursor = 0
		}
	case "S":
		if !m.loading {
			m.statsData = computeStatsRows(m.sessions)
			m.statsCursor = 0
			m.statsOffset = 0
			m.mode = modeStats
		}
	case "T":
		if !m.loading {
			m.mode = modeTimeline
			m.timelineCursor = startOfDay(time.Now())
		}
	case "m":
		if !m.loading && len(m.visibleSessions) > 0 {
			selected := m.visibleSessions[m.cursor]
			on := toggleBookmark(m.bookmarks, selected.ID)
			if on {
				m.flashMsg = "bookmarked"
			} else {
				m.flashMsg = "unbookmarked"
			}
			if m.bookmarksPath != "" {
				_ = saveBookmarks(m.bookmarksPath, m.bookmarks)
			}
			if m.bookmarkOnly {
				m.applyFilter()
				if m.cursor >= len(m.visibleSessions) {
					m.cursor = len(m.visibleSessions) - 1
				}
				if m.cursor < 0 {
					m.cursor = 0
				}
			}
		}
	case "M":
		if !m.loading {
			if !m.bookmarkOnly && len(m.bookmarks) == 0 {
				m.flashMsg = "no bookmarks yet (press m to mark a session)"
			} else {
				m.bookmarkOnly = !m.bookmarkOnly
				m.applyFilter()
				m.cursor = 0
				m.listOffset = 0
			}
		}
	case "enter", "l", "right":
		if !m.loading && len(m.visibleSessions) > 0 {
			m.detailLoading = true
			selected := m.visibleSessions[m.cursor]
			m.detailSession = selected
			return m, loadSessionDetailCmd(selected.Path)
		}
	case "esc":
		if m.appliedFilterMode != filterModeNone || m.bookmarkOnly || !m.dateFilter.IsZero() {
			m.filterText = ""
			m.appliedFilterMode = filterModeNone
			m.bookmarkOnly = false
			m.dateFilter = time.Time{}
			m.applyFilter()
			m.cursor = 0
		}
	}
	return m, nil
}

func (m model) handleFilterEntryKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		m.appliedFilterMode = m.filterMode
		m.applyFilter()
		m.filterMode = filterModeNone
		if len(m.visibleSessions) == 0 {
			m.cursor = 0
		} else if m.cursor >= len(m.visibleSessions) {
			m.cursor = len(m.visibleSessions) - 1
		}
	case tea.KeyEsc:
		m.filterText = ""
		m.visibleSessions = m.sessions
		m.filterMode = filterModeNone
		m.appliedFilterMode = filterModeNone
		m.cursor = 0
	case tea.KeyBackspace:
		runes := []rune(m.filterText)
		if len(runes) > 0 {
			m.filterText = string(runes[:len(runes)-1])
		}
	case tea.KeyRunes:
		m.filterText += string(msg.Runes)
	}
	return m, nil
}
