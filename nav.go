package lore

// nav advances cursor by the standard list keys (j, k, d, u, g, G, down, up).
// Returns the new cursor. count is len(items); halfPage is the half-page step
// (always >= 1). For unknown keys returns cursor unchanged.
func nav(key string, cursor, count, halfPage int) int {
	if count <= 0 {
		return 0
	}
	if halfPage < 1 {
		halfPage = 1
	}
	switch key {
	case "j", "down":
		if cursor < count-1 {
			cursor++
		}
	case "k", "up":
		if cursor > 0 {
			cursor--
		}
	case "d":
		cursor += halfPage
		if cursor >= count {
			cursor = count - 1
		}
	case "u":
		cursor -= halfPage
		if cursor < 0 {
			cursor = 0
		}
	case "g":
		cursor = 0
	case "G":
		cursor = count - 1
	}
	return cursor
}
