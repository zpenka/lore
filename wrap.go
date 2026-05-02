package lore

import (
	"strings"
	"unicode/utf8"
)

// wrapText soft-wraps s into lines no longer than w (in runes), preferring
// to break at whitespace. Long words that don't fit on a single line are
// hard-cut at the rune boundary. A literal "\n" in s starts a new
// paragraph and is preserved as an empty output line if doubled.
//
// Used by detailBodyLines to flatten multi-line, over-width turn bodies
// into a slice of visual rows so the viewport math (which counts body
// lines, not turns) reflects what the terminal actually renders.
func wrapText(s string, w int) []string {
	if w <= 0 {
		w = 80
	}
	if s == "" {
		return []string{""}
	}

	var out []string
	for _, paragraph := range strings.Split(s, "\n") {
		if paragraph == "" {
			out = append(out, "")
			continue
		}
		out = append(out, wrapParagraph(paragraph, w)...)
	}
	return out
}

func wrapParagraph(p string, w int) []string {
	words := strings.Fields(p)
	if len(words) == 0 {
		return []string{p}
	}

	var out []string
	line := ""
	lineWidth := 0

	for _, word := range words {
		ww := utf8.RuneCountInString(word)
		// Hard-cut a word that's longer than w (e.g. a long URL).
		for ww > w {
			if line != "" {
				out = append(out, line)
				line = ""
				lineWidth = 0
			}
			cut := runePrefix(word, w)
			out = append(out, cut)
			word = strings.TrimPrefix(word, cut)
			ww = utf8.RuneCountInString(word)
		}
		need := ww
		if lineWidth > 0 {
			need++ // account for the leading space
		}
		if lineWidth+need > w {
			if line != "" {
				out = append(out, line)
			}
			line = word
			lineWidth = ww
			continue
		}
		if lineWidth > 0 {
			line += " "
			lineWidth++
		}
		line += word
		lineWidth += ww
	}
	if line != "" {
		out = append(out, line)
	}
	return out
}

// runePrefix returns the first n runes of s.
func runePrefix(s string, n int) string {
	if n <= 0 {
		return ""
	}
	count := 0
	for i := range s {
		if count == n {
			return s[:i]
		}
		count++
	}
	return s
}
