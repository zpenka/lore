package lore

import (
	"fmt"
	"strings"
)

func renderRerunHeader(m model) string {
	return headerStyle.Render(fmt.Sprintf(" re-run · source: %s", m.detailSession.Slug))
}

func renderRerunView(m model) string {
	var b strings.Builder

	b.WriteString(renderRerunHeader(m))
	b.WriteByte('\n')
	b.WriteString(renderDivider(m.width))
	b.WriteByte('\n')

	b.WriteString(" prompt:\n")
	boxWidth := m.width - 4
	if boxWidth < 10 {
		boxWidth = 10
	}
	b.WriteString(" ┌" + strings.Repeat("─", boxWidth) + "┐\n")
	promptLines := strings.Split(m.rerunPrompt, "\n")
	rendered := 0
	for _, line := range promptLines {
		if rendered >= rerunMaxLines {
			b.WriteString(" │ " + truncatePromptLine("...", boxWidth-2) + "\n")
			break
		}
		truncated := truncatePromptLine(line, boxWidth-2)
		padded := truncated + strings.Repeat(" ", boxWidth-2-len(truncated))
		b.WriteString(" │ " + padded + " │\n")
		rendered++
	}
	for rendered < rerunMaxLines && rendered < len(promptLines) {
		padded := strings.Repeat(" ", boxWidth-2)
		b.WriteString(" │ " + padded + " │\n")
		rendered++
	}
	b.WriteString(" └" + strings.Repeat("─", boxWidth) + "┘\n")

	cwdLine := fmt.Sprintf(" cwd:    %s\n", m.rerunCWD)
	b.WriteString(cwdLine)

	b.WriteString(renderDivider(m.width))
	b.WriteByte('\n')
	b.WriteString(renderRerunFooter(m))
	b.WriteByte('\n')
	return b.String()
}

func renderRerunFooter(m model) string {
	if m.flashMsg != "" {
		return flashStyle.Render(" " + m.flashMsg)
	}
	return footerStyle.Render(" enter run   ? help   q/esc/h/← back")
}
