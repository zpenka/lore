package lore

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

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
		if expanded {
			if scTurns, ok := m.sidechainTurns[fullIdx]; ok {
				for _, ln := range renderSidechainTurns(scTurns, m.width) {
					lines = append(lines, ln)
				}
			}
		}
	}
	return
}

func wrapTurnLines(t turn, expanded bool, width int) []string {
	first, cont := turnPrefixes(t.kind)
	avail := width - utf8.RuneCountInString(first)
	if avail < 10 {
		avail = 10
	}
	body := t.body
	if t.sidechainPath != "" {
		body = "⧑ " + body
	}
	wrapped := wrapText(body, avail)
	out := make([]string, 0, len(wrapped))
	for i, line := range wrapped {
		if i == 0 {
			out = append(out, first+line)
		} else {
			out = append(out, cont+line)
		}
	}
	if t.kind == "tool" && expanded {
		toolName := extractToolName(t.body)
		if toolName == "Edit" {
			out = append(out, renderEditDiff(t.input, cont, avail)...)
		} else if toolName == "Write" {
			out = append(out, renderWriteDiff(t.input, cont, avail)...)
		} else {
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

func renderSidechainTurns(turns []turn, width int) []string {
	const indent = "      │     "
	avail := width - utf8.RuneCountInString(indent)
	if avail < 10 {
		avail = 10
	}
	var lines []string
	for _, t := range turns {
		prefix := ""
		switch t.kind {
		case "user":
			prefix = "user: "
		case "asst":
			prefix = "asst: "
		case "tool":
			prefix = "▸ "
		case "thinking":
			continue
		}
		wrapped := wrapText(prefix+t.body, avail)
		for _, ln := range wrapped {
			lines = append(lines, indent+ln)
		}
	}
	return lines
}

func renderDetailView(m model) string {
	if m.detailErr != nil {
		return errorStyle.Render(fmt.Sprintf(" error: %v", m.detailErr)) + "\n"
	}
	if m.detailLoading {
		return " loading session...\n"
	}

	var b strings.Builder

	b.WriteString(renderDetailHeader(m))
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

func renderDetailHeader(m model) string {
	dateStr := m.detailSession.Timestamp.Format("2006-01-02")
	visible := m.visibleTurns()
	turnInfo := ""
	if len(visible) > 0 {
		turnInfo = fmt.Sprintf("   turn %d/%d", m.cursorDetail+1, len(visible))
	}
	headerLine := fmt.Sprintf(" %s · %s · %s   %s%s",
		m.detailSession.Slug,
		m.detailSession.Project,
		m.detailSession.Branch,
		dateStr,
		turnInfo,
	)
	return headerStyle.Render(headerLine)
}

func renderDetailFooter(m model) string {
	if m.flashMsg != "" {
		return flashStyle.Render(" " + m.flashMsg)
	}
	copyStatus := ""
	if m.justCopied {
		copyStatus = "  ✓ copied"
	}
	return footerStyle.Render(fmt.Sprintf(
		" j/k move   d/u page   g/G top/bottom   space expand   y copy   r run   R resume   m bookmark   / search   ? help   q/esc/h/← back%s",
		copyStatus))
}

func extractToolName(body string) string {
	parts := strings.SplitN(body, " ", 2)
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}

func renderEditDiff(input map[string]interface{}, cont string, avail int) []string {
	var out []string
	if filePath, ok := input["file_path"].(string); ok {
		line := fmt.Sprintf("  file: %s", filePath)
		for _, l := range wrapText(line, avail) {
			out = append(out, cont+l)
		}
	}
	if oldStr, ok := input["old_string"].(string); ok {
		lines := strings.Split(oldStr, "\n")
		for _, line := range lines {
			prefixed := "- " + line
			for _, wrappedLine := range wrapText(prefixed, avail) {
				out = append(out, cont+diffRemoveStyle.Render(wrappedLine))
			}
		}
	}
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

func renderWriteDiff(input map[string]interface{}, cont string, avail int) []string {
	var out []string
	if filePath, ok := input["file_path"].(string); ok {
		line := fmt.Sprintf("  file: %s", filePath)
		for _, l := range wrapText(line, avail) {
			out = append(out, cont+l)
		}
	}
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

// truncatePromptLine limits s to maxLen runes with "…" if truncated.
// Delegates to truncateRunes (wrap.go) which is the canonical implementation.
func truncatePromptLine(s string, maxLen int) string {
	return truncateRunes(s, maxLen)
}
