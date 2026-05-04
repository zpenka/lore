package lore

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
)

var (
	headerStyle     = lipgloss.NewStyle().Bold(true)
	bucketStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	selectedStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Bold(true)
	footerStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	flashStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	errorStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	diffAddStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	diffRemoveStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
)

// chromeLines is the constant number of fixed-height rows around the
// scrollable body in every mode: header + divider + divider + footer.
const chromeLines = 4

func (m model) View() string {
	// If help overlay is showing, render it instead of the normal view
	if m.showHelp {
		return renderHelpOverlay(m)
	}

	switch m.mode {
	case modeDetail:
		return renderDetailView(m)
	case modeList:
		return renderListView(m)
	case modeSearch:
		return renderSearchView(m)
	case modeProject:
		return renderProjectView(m, time.Now())
	case modeRerun:
		return renderRerunView(m)
	}
	return ""
}

// bodyHeight returns the number of body lines to render for the current
// terminal height, or -1 if the height is unknown / too small to scroll
// (in which case callers should render the whole body unsliced).
func (m model) bodyHeight() int {
	if m.height <= chromeLines {
		return -1
	}
	return m.height - chromeLines
}

// renderBody slices `lines` to fit the available body height starting
// from `offset`. When the height is unknown (<=0) it returns the lines
// unchanged so pre-window-size renders aren't truncated.
func renderBody(lines []string, offset int, height int) []string {
	if height <= 0 {
		return lines
	}
	return sliceLines(lines, offset, height)
}

// ----- list mode -----

// listBodyLines builds the rendered rows for list mode and returns both
// the flat slice and the line index of the selected session.
func listBodyLines(m model, now time.Time) (lines []string, cursorLine int) {
	var lastBucket string
	for i, s := range m.visibleSessions {
		bucket := timeBucket(s.Timestamp, now)
		if bucket != lastBucket {
			lines = append(lines, bucketStyle.Render(" "+bucket))
			lastBucket = bucket
		}
		if i == m.cursor {
			cursorLine = len(lines)
		}
		lines = append(lines, renderRow(s, i == m.cursor))
	}
	return
}

func renderListView(m model) string {
	if m.err != nil {
		return errorStyle.Render(fmt.Sprintf(" error: %v", m.err)) + "\n"
	}
	if m.loading {
		return " loading sessions...\n"
	}
	if len(m.sessions) == 0 {
		return fmt.Sprintf(" No sessions found in %s\n", m.projectsDir)
	}

	var b strings.Builder
	b.WriteString(renderHeader(m))
	b.WriteByte('\n')
	b.WriteString(renderDivider(m.width))
	b.WriteByte('\n')

	body, cursorLine := listBodyLines(m, time.Now())
	height := m.bodyHeight()
	offset := clampOffset(m.listOffset, cursorLine, len(body), height)
	for _, line := range renderBody(body, offset, height) {
		b.WriteString(line)
		b.WriteByte('\n')
	}

	b.WriteString(renderDivider(m.width))
	b.WriteByte('\n')
	b.WriteString(renderFooter(m))
	if m.detailLoading {
		b.WriteString("  (loading session...)")
	}
	b.WriteByte('\n')
	return b.String()
}

// ----- detail mode -----

// detailBodyLines builds the rendered turn rows for detail mode and
// returns the line index of the FIRST visual row of the selected turn.
//
// Multi-line and over-width turn bodies are wrapped into multiple body
// lines so viewport math matches what the terminal actually renders.
// The first visual row of a turn carries the kind marker (` user │ `,
// ` asst │ `, etc.); continuation rows use a blank-prefixed continuation
// (`      │ `) that keeps the gutter aligned.
func detailBodyLines(m model) (lines []string, cursorLine int) {
	visible := m.visibleTurns()
	for i, t := range visible {
		isSelected := (i == m.cursorDetail)
		fullIdx := m.visibleIndexToFullIndex(i)
		expanded := m.expandedTurns[fullIdx]
		if isSelected {
			cursorLine = len(lines)
		}
		for _, ln := range wrapTurnLines(t, expanded, m.width) {
			if isSelected {
				lines = append(lines, selectedStyle.Render(ln))
			} else {
				lines = append(lines, ln)
			}
		}
	}
	return
}

// wrapTurnLines flattens one turn into a slice of visual rows.
func wrapTurnLines(t turn, expanded bool, width int) []string {
	first, cont := turnPrefixes(t.kind)
	avail := width - utf8.RuneCountInString(first)
	if avail < 10 {
		avail = 10
	}
	wrapped := wrapText(t.body, avail)
	out := make([]string, 0, len(wrapped))
	for i, line := range wrapped {
		if i == 0 {
			out = append(out, first+line)
		} else {
			out = append(out, cont+line)
		}
	}
	if t.kind == "tool" && expanded {
		// Check if this is an Edit or Write tool that should render as a diff
		toolName := extractToolName(t.body)
		if toolName == "Edit" {
			out = append(out, renderEditDiff(t.input, cont, avail)...)
		} else if toolName == "Write" {
			out = append(out, renderWriteDiff(t.input, cont, avail)...)
		} else {
			// Render input as `key: value` rows under a continuation gutter,
			// each row also wrapped to the same width.
			for k, v := range t.input {
				kv := fmt.Sprintf("  %s: %v", k, v)
				for _, line := range wrapText(kv, avail) {
					out = append(out, cont+line)
				}
			}
		}
	}
	return out
}

// turnPrefixes returns the first-line and continuation-line gutters for
// a given turn kind. Both are visually 8 columns wide (modulo the unicode
// │ glyph) so wrapped continuation rows align with the marker column.
func turnPrefixes(kind string) (first, cont string) {
	switch kind {
	case "user":
		return " user │ ", "      │ "
	case "asst":
		return " asst │ ", "      │ "
	case "thinking":
		return " think │ 〰 ", "       │   "
	case "tool":
		return "      │ ▸ ", "      │   "
	}
	return "      │ ", "      │ "
}

func renderDetailView(m model) string {
	if m.detailErr != nil {
		return errorStyle.Render(fmt.Sprintf(" error: %v", m.detailErr)) + "\n"
	}
	if m.detailLoading {
		return " loading session...\n"
	}

	var b strings.Builder

	dateStr := m.detailSession.Timestamp.Format("2006-01-02")
	headerLine := fmt.Sprintf(" %s · %s · %s   %s",
		m.detailSession.Slug,
		m.detailSession.Project,
		m.detailSession.Branch,
		dateStr,
	)
	b.WriteString(headerStyle.Render(headerLine))
	b.WriteByte('\n')
	b.WriteString(renderDivider(m.width))
	b.WriteByte('\n')

	body, cursorLine := detailBodyLines(m)
	if len(body) == 0 {
		body = []string{" (no turns to display)"}
		cursorLine = 0
	}
	height := m.bodyHeight()
	offset := clampOffset(m.detailOffset, cursorLine, len(body), height)
	for _, line := range renderBody(body, offset, height) {
		b.WriteString(line)
		b.WriteByte('\n')
	}

	b.WriteString(renderDivider(m.width))
	b.WriteByte('\n')
	b.WriteString(renderDetailFooter(m))
	b.WriteByte('\n')

	return b.String()
}

// renderDetailFooter renders the footer for detail view.
func renderDetailFooter(m model) string {
	if m.flashMsg != "" {
		return flashStyle.Render(" " + m.flashMsg)
	}
	thinkingLabel := "thinking"
	if m.showThinking {
		thinkingLabel = "hide thinking"
	}
	copyStatus := ""
	if m.justCopied {
		copyStatus = "  ✓ copied"
	}
	return footerStyle.Render(fmt.Sprintf(
		" j/k move   g/G top/bottom   space expand   t %s   y copy   r run   q/esc back%s",
		thinkingLabel, copyStatus))
}

// ----- list header / row helpers -----

func renderHeader(m model) string {
	nProjects := countProjects(m.sessions)
	return headerStyle.Render(fmt.Sprintf(
		" lore · %d session%s across %d project%s",
		len(m.sessions), plural(len(m.sessions)),
		nProjects, plural(nProjects),
	))
}

func renderDivider(width int) string {
	if width < 4 {
		width = 80
	}
	return strings.Repeat("─", width)
}

// renderRows is retained for tests that don't care about viewport semantics.
// New code should use listBodyLines + clampOffset + sliceLines instead.
func renderRows(m model, now time.Time) string {
	lines, _ := listBodyLines(m, now)
	return strings.Join(lines, "\n") + "\n"
}

func renderRow(s Session, selected bool) string {
	cursor := "  "
	if selected {
		cursor = " ►"
	}
	row := fmt.Sprintf("%s %s  %-12s  %-26s  %s",
		cursor,
		s.Timestamp.Format("15:04"),
		padTrunc(s.Project, 12),
		padTrunc(s.Branch, 26),
		s.Slug,
	)
	if selected {
		return selectedStyle.Render(row)
	}
	return row
}

func countProjects(ss []Session) int {
	seen := make(map[string]struct{}, len(ss))
	for _, s := range ss {
		seen[s.CWD] = struct{}{}
	}
	return len(seen)
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

func renderFooter(m model) string {
	if m.flashMsg != "" {
		return flashStyle.Render(" " + m.flashMsg)
	}
	if m.filterMode == filterModeProject {
		return footerStyle.Render(fmt.Sprintf(" project filter: %s_  [enter] apply  [esc] cancel", m.filterText))
	}
	if m.filterMode == filterModeBranch {
		return footerStyle.Render(fmt.Sprintf(" branch filter: %s_  [enter] apply  [esc] cancel", m.filterText))
	}
	if m.filterMode == filterModeFuzzy {
		return footerStyle.Render(fmt.Sprintf(" fuzzy filter: %s_  [enter] apply  [esc] cancel", m.filterText))
	}
	if m.filterText != "" && m.appliedFilterMode != filterModeNone {
		switch m.appliedFilterMode {
		case filterModeProject:
			return footerStyle.Render(fmt.Sprintf(" filtered by project: %s   j/k · enter open · esc clear   q quit", m.filterText))
		case filterModeBranch:
			return footerStyle.Render(fmt.Sprintf(" filtered by branch: %s   j/k · enter open · esc clear   q quit", m.filterText))
		case filterModeFuzzy:
			return footerStyle.Render(fmt.Sprintf(" fuzzy filter: %s   j/k · enter open · esc clear   q quit", m.filterText))
		}
	}
	return footerStyle.Render(" j/k move   enter open   / search   p project   b branch   f fuzzy   P project view   g/G top/bottom   q quit")
}

// padTrunc trims s to max display columns or right-pads it to fit.
func padTrunc(s string, max int) string {
	if len(s) > max {
		if max <= 1 {
			return s[:max]
		}
		return s[:max-1] + "…"
	}
	return s + strings.Repeat(" ", max-len(s))
}

// ----- search mode -----

// searchBodyLines builds the rendered result rows for search-results
// mode. Each result spans two lines (session row + snippet); cursorLine
// is the index of the session row for the selected hit.
func searchBodyLines(m model) (lines []string, cursorLine int) {
	if m.searchMode == searchModeEntry {
		return nil, 0
	}
	if len(m.searchResults) == 0 {
		return []string{" (no matches)"}, 0
	}
	for i, hit := range m.searchResults {
		isSelected := (i == m.searchCursor)
		if isSelected {
			cursorLine = len(lines)
		}
		row := fmt.Sprintf("  %s  %-12s  %-26s  %s",
			hit.Session.Timestamp.Format("15:04"),
			padTrunc(hit.Session.Project, 12),
			padTrunc(hit.Session.Branch, 26),
			hit.Session.Slug,
		)
		snippet := "    ▸ " + hit.Snippet
		if isSelected {
			lines = append(lines, selectedStyle.Render(row))
			lines = append(lines, selectedStyle.Render(snippet))
		} else {
			lines = append(lines, row)
			lines = append(lines, snippet)
		}
	}
	return
}

func renderSearchView(m model) string {
	var b strings.Builder

	if m.searchMode == searchModeEntry {
		headerLine := fmt.Sprintf(" search: %s_   [enter] run   [esc] cancel", m.searchQuery)
		b.WriteString(headerStyle.Render(headerLine))
	} else {
		hitWord := "hit"
		if len(m.searchResults) != 1 {
			hitWord = "hits"
		}
		hitCount := 0
		for _, r := range m.searchResults {
			hitCount += r.HitCount
		}
		headerLine := fmt.Sprintf(" search: %s     %d %s across %d session%s",
			m.searchQuery, hitCount, hitWord,
			len(m.searchResults), plural(len(m.searchResults)),
		)
		b.WriteString(headerStyle.Render(headerLine))
	}
	b.WriteByte('\n')
	b.WriteString(renderDivider(m.width))
	b.WriteByte('\n')

	body, cursorLine := searchBodyLines(m)
	height := m.bodyHeight()
	offset := clampOffset(m.searchOffset, cursorLine, len(body), height)
	for _, line := range renderBody(body, offset, height) {
		b.WriteString(line)
		b.WriteByte('\n')
	}

	b.WriteString(renderDivider(m.width))
	b.WriteByte('\n')

	if m.flashMsg != "" {
		b.WriteString(flashStyle.Render(" " + m.flashMsg))
	} else if m.searchMode == searchModeEntry {
		b.WriteString(footerStyle.Render(" search: " + m.searchQuery + "_   [enter] run   [esc] cancel"))
	} else {
		b.WriteString(footerStyle.Render(" j/k move   enter open   / new search   g/G top/bottom   esc back"))
	}
	b.WriteByte('\n')

	return b.String()
}

// ----- re-run -----

func renderRerunView(m model) string {
	var b strings.Builder

	headerLine := fmt.Sprintf(" re-run · source: %s", m.detailSession.Slug)
	b.WriteString(headerStyle.Render(headerLine))
	b.WriteByte('\n')
	b.WriteString(renderDivider(m.width))
	b.WriteByte('\n')

	b.WriteString(" prompt:\n")
	boxWidth := m.width - 4
	if boxWidth < 10 {
		boxWidth = 10
	}
	b.WriteString(" ┌" + strings.Repeat("─", boxWidth) + "┐\n")
	const maxLines = 5
	promptLines := strings.Split(m.rerunPrompt, "\n")
	rendered := 0
	for _, line := range promptLines {
		if rendered >= maxLines {
			b.WriteString(" │ " + truncatePromptLine("...", boxWidth-2) + "\n")
			break
		}
		truncated := truncatePromptLine(line, boxWidth-2)
		padded := truncated + strings.Repeat(" ", boxWidth-2-len(truncated))
		b.WriteString(" │ " + padded + " │\n")
		rendered++
	}
	for rendered < maxLines && rendered < len(promptLines) {
		padded := strings.Repeat(" ", boxWidth-2)
		b.WriteString(" │ " + padded + " │\n")
		rendered++
	}
	b.WriteString(" └" + strings.Repeat("─", boxWidth) + "┘\n")

	cwdLine := fmt.Sprintf(" cwd:    %s\n", m.rerunCWD)
	b.WriteString(cwdLine)

	b.WriteString(renderDivider(m.width))
	b.WriteByte('\n')
	b.WriteString(footerStyle.Render(" enter run   esc cancel"))
	b.WriteByte('\n')
	return b.String()
}

// extractToolName extracts the tool name from the tool body string.
// The body format is "ToolName <snippet>", so we split on the first space.
func extractToolName(body string) string {
	parts := strings.SplitN(body, " ", 2)
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}

// renderEditDiff renders the input for an Edit tool as a diff with old_string (red) and new_string (green).
func renderEditDiff(input map[string]interface{}, cont string, avail int) []string {
	var out []string

	// Render file path
	if filePath, ok := input["file_path"].(string); ok {
		line := fmt.Sprintf("  file: %s", filePath)
		for _, l := range wrapText(line, avail) {
			out = append(out, cont+l)
		}
	}

	// Render old_string lines with "- " prefix in red
	if oldStr, ok := input["old_string"].(string); ok {
		lines := strings.Split(oldStr, "\n")
		for _, line := range lines {
			prefixed := "- " + line
			for _, wrappedLine := range wrapText(prefixed, avail) {
				out = append(out, cont+diffRemoveStyle.Render(wrappedLine))
			}
		}
	}

	// Render new_string lines with "+ " prefix in green
	if newStr, ok := input["new_string"].(string); ok {
		lines := strings.Split(newStr, "\n")
		for _, line := range lines {
			prefixed := "+ " + line
			for _, wrappedLine := range wrapText(prefixed, avail) {
				out = append(out, cont+diffAddStyle.Render(wrappedLine))
			}
		}
	}

	return out
}

// renderWriteDiff renders the input for a Write tool as add-only (all lines green with "+ " prefix).
func renderWriteDiff(input map[string]interface{}, cont string, avail int) []string {
	var out []string

	// Render file path
	if filePath, ok := input["file_path"].(string); ok {
		line := fmt.Sprintf("  file: %s", filePath)
		for _, l := range wrapText(line, avail) {
			out = append(out, cont+l)
		}
	}

	// Render content lines with "+ " prefix in green
	if content, ok := input["content"].(string); ok {
		lines := strings.Split(content, "\n")
		for _, line := range lines {
			prefixed := "+ " + line
			for _, wrappedLine := range wrapText(prefixed, avail) {
				out = append(out, cont+diffAddStyle.Render(wrappedLine))
			}
		}
	}

	return out
}

func truncatePromptLine(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	if maxLen <= 1 {
		return string(runes[:maxLen])
	}
	return string(runes[:maxLen-1]) + "…"
}

// renderHelpOverlay renders a help screen showing keybindings for the current mode.
func renderHelpOverlay(m model) string {
	var helpText string

	switch m.mode {
	case modeList:
		helpText = `
 ┌─ List Mode Help ─────────────────────────────────────────────────────────┐
 │                                                                           │
 │  Navigation:                                                              │
 │    j/k, ↑/↓     Move cursor                                              │
 │    g/G          Jump to top/bottom                                       │
 │    enter, l, →  Open the highlighted session                            │
 │                                                                           │
 │  Filtering:                                                               │
 │    p             Filter to one project (inline)                          │
 │    b             Filter to one branch (inline)                           │
 │    f             Fuzzy filter across slug, project, and branch           │
 │    esc           Clear filter                                            │
 │                                                                           │
 │  Other:                                                                   │
 │    P             Open project view for current session's CWD             │
 │    /             Enter full-text search                                  │
 │    ?             Show this help overlay                                  │
 │    q             Quit                                                    │
 │                                                                           │
 └─────────────────────────────────────────────────────────────────────────┘
`

	case modeDetail:
		helpText = `
 ┌─ Detail Mode Help ────────────────────────────────────────────────────────┐
 │                                                                            │
 │  Navigation:                                                               │
 │    j/k, ↑/↓     Scroll through turns                                      │
 │    g/G          Jump to top/bottom                                        │
 │                                                                            │
 │  Turn Actions:                                                             │
 │    space         Expand/collapse a tool turn (shows key: value)           │
 │    y             Copy the nearest user prompt to clipboard                │
 │    r             Enter re-run mode with the selected user prompt          │
 │                                                                            │
 │  Display:                                                                  │
 │    t             Toggle thinking blocks (hidden by default)               │
 │    ?             Show this help overlay                                   │
 │                                                                            │
 │  Return to List:                                                           │
 │    esc, q       Back to the session list                                  │
 │                                                                            │
 └────────────────────────────────────────────────────────────────────────┘
`

	case modeSearch:
		helpText = `
 ┌─ Search Mode Help ────────────────────────────────────────────────────────┐
 │                                                                            │
 │  Search Entry:                                                             │
 │    Type         Build search query                                        │
 │    enter        Run linear scan search                                    │
 │    esc          Cancel, return to list                                    │
 │                                                                            │
 │  Search Results:                                                           │
 │    j/k, ↑/↓     Move through results (sorted by hit count)               │
 │    g/G          Jump to top/bottom                                        │
 │    enter        Open the selected session in detail                       │
 │    /            Re-search (edit query)                                    │
 │    esc          Back to list                                              │
 │    ?            Show this help overlay                                    │
 │                                                                            │
 └────────────────────────────────────────────────────────────────────────┘
`

	case modeProject:
		helpText = `
 ┌─ Project Mode Help ───────────────────────────────────────────────────────┐
 │                                                                            │
 │  Navigation:                                                               │
 │    j/k, ↑/↓     Move within the project's sessions                        │
 │    g/G          Jump to top/bottom                                        │
 │    enter        Open session detail                                       │
 │                                                                            │
 │  Return to List:                                                           │
 │    esc, q       Back to session list                                      │
 │    ?            Show this help overlay                                    │
 │                                                                            │
 │  Sessions are grouped by branch (latest branch first).                    │
 │                                                                            │
 └────────────────────────────────────────────────────────────────────────┘
`

	case modeRerun:
		helpText = `
 ┌─ Re-run Mode Help ────────────────────────────────────────────────────────┐
 │                                                                            │
 │  Actions:                                                                  │
 │    enter        Spawn 'claude' with the chosen prompt                     │
 │                 (subprocess owns the TTY; lore exits cleanly on return)    │
 │    esc, q       Cancel and return to detail view                          │
 │    ?            Show this help overlay                                    │
 │                                                                            │
 └────────────────────────────────────────────────────────────────────────┘
`

	default:
		helpText = `
 ┌─ Help ────────────────────────────────────────────────────────────────────┐
 │                                                                            │
 │  Press ? in any mode to see mode-specific keybindings.                   │
 │  Any key dismisses this help overlay.                                     │
 │                                                                            │
 └────────────────────────────────────────────────────────────────────────┘
`
	}

	return helpText
}
