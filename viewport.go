package lore

// Each "view" mode (list, detail, search, project) renders a body of
// rows along with a header + footer. clampOffset and sliceLines are the
// two primitives every renderer uses to keep the cursor visible without
// pulling in a heavy viewport dependency.

// clampOffset returns the offset that keeps cursorLine visible inside a
// window of `height` lines. It uses edge-snap semantics: the offset only
// moves when the cursor would otherwise leave the window. The result is
// clamped to [0, max(0, totalLines-height)].
func clampOffset(offset, cursorLine, totalLines, height int) int {
	if height <= 0 || totalLines <= height {
		return 0
	}
	if cursorLine < offset {
		offset = cursorLine
	}
	if cursorLine >= offset+height {
		offset = cursorLine - height + 1
	}
	if offset < 0 {
		offset = 0
	}
	if max := totalLines - height; offset > max {
		offset = max
	}
	return offset
}

// sliceLines returns lines[offset:offset+height], padded with empty
// strings if `lines` is shorter than the window. This anchors the
// footer at a predictable position regardless of body length.
func sliceLines(lines []string, offset, height int) []string {
	if height <= 0 {
		return nil
	}
	if offset < 0 {
		offset = 0
	}
	out := make([]string, height)
	for i := 0; i < height; i++ {
		idx := offset + i
		if idx >= 0 && idx < len(lines) {
			out[i] = lines[idx]
		}
	}
	return out
}
