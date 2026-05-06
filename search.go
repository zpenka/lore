package lore

import (
	"bufio"
	"encoding/json"
	"os"
	"sort"
	"strings"
)

// SearchHit represents a single session matching the search query.
type SearchHit struct {
	Session  Session // metadata for the session
	HitCount int     // total matching turns/text-blocks
	Snippet  string  // first matching turn's text, truncated to ~80 chars
}

// searchSessions performs a linear-scan full-text search across sessions.
// For each session:
//   - Opens the file at session.Path
//   - Scans line-by-line as JSON events
//   - For "user" events: extracts message text (handles string and array forms)
//     Skips tool-result-only user events
//   - For "assistant" events: iterates message.content[] blocks
//     Matches only "text" blocks, skips "tool_use" and "thinking"
//   - Counts matching blocks and keeps first match as snippet
//
// Returns results sorted by HitCount descending, then by Timestamp descending.
// Empty query returns empty slice.
func searchSessions(sessions []Session, query string) []SearchHit {
	if query == "" {
		return nil
	}

	query = strings.ToLower(query)
	var results []SearchHit

	for _, sess := range sessions {
		hit := searchSession(sess, query)
		if hit != nil {
			results = append(results, *hit)
		}
	}

	// Sort by HitCount descending, then by Timestamp descending
	sort.Slice(results, func(i, j int) bool {
		if results[i].HitCount != results[j].HitCount {
			return results[i].HitCount > results[j].HitCount
		}
		return results[i].Session.Timestamp.After(results[j].Session.Timestamp)
	})

	return results
}

// searchSession searches a single session file and returns a SearchHit if there are matches.
func searchSession(sess Session, query string) *SearchHit {
	f, err := os.Open(sess.Path)
	if err != nil {
		return nil
	}
	defer f.Close()

	var hitCount int
	var firstSnippet string

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64*1024), 16*1024*1024)

	for scanner.Scan() {
		var ev rawEvent
		if err := json.Unmarshal(scanner.Bytes(), &ev); err != nil {
			continue
		}

		switch ev.Type {
		case "user":
			if hits, snippet := matchUserEvent(&ev, query); hits > 0 {
				hitCount += hits
				if firstSnippet == "" {
					firstSnippet = snippet
				}
			}
		case "assistant":
			if hits, snippet := matchAssistantEvent(&ev, query); hits > 0 {
				hitCount += hits
				if firstSnippet == "" {
					firstSnippet = snippet
				}
			}
		}
	}

	if hitCount == 0 {
		return nil
	}

	return &SearchHit{
		Session:  sess,
		HitCount: hitCount,
		Snippet:  firstSnippet,
	}
}

// matchUserEvent extracts text from a user event and counts matches.
// Returns (hitCount, snippet) or (0, "") if no matches.
func matchUserEvent(ev *rawEvent, query string) (int, string) {
	if ev.Message == nil || ev.Message.Content == nil {
		return 0, ""
	}

	content := ev.Message.Content
	var textParts []string
	var allToolResults bool

	// Content can be a string or an array
	switch c := content.(type) {
	case string:
		textParts = append(textParts, c)
		allToolResults = false
	case []interface{}:
		// Array of content blocks
		allToolResults = len(c) > 0
		for _, b := range c {
			blockMap, ok := b.(map[string]interface{})
			if !ok {
				continue
			}
			blockType, ok := blockMap["type"].(string)
			if !ok {
				continue
			}
			if blockType != "tool_result" {
				allToolResults = false
			}
			// Extract text from text blocks
			if blockType == "text" {
				if txt, ok := blockMap["text"].(string); ok {
					textParts = append(textParts, txt)
				}
			}
		}
	default:
		return 0, ""
	}

	// Skip if all blocks are tool_result
	if allToolResults && len(textParts) == 0 {
		return 0, ""
	}

	// Count matches and find snippet
	text := strings.Join(textParts, " ")
	if text == "" {
		return 0, ""
	}

	return countAndSnippet(text, query)
}

// matchAssistantEvent extracts and counts matches from assistant event.
// Returns (hitCount, snippet) or (0, "") if no matches.
func matchAssistantEvent(ev *rawEvent, query string) (int, string) {
	if ev.Message == nil || ev.Message.Content == nil {
		return 0, ""
	}

	content := ev.Message.Content
	blocks, ok := content.([]interface{})
	if !ok {
		return 0, ""
	}

	var hitCount int
	var firstSnippet string

	for _, b := range blocks {
		blockMap, ok := b.(map[string]interface{})
		if !ok {
			continue
		}

		blockType, ok := blockMap["type"].(string)
		if !ok {
			continue
		}

		// Only match on "text" blocks; skip "tool_use" and "thinking"
		if blockType == "text" {
			if text, ok := blockMap["text"].(string); ok && text != "" {
				hits, snippet := countAndSnippet(text, query)
				hitCount += hits
				if firstSnippet == "" && hits > 0 {
					firstSnippet = snippet
				}
			}
		}
	}

	return hitCount, firstSnippet
}

// countAndSnippet counts substring matches (case-insensitive) in text
// and returns a snippet with the match highlighted.
// Truncates snippet to ~80 chars, centering the match if it's past char 40.
func countAndSnippet(text, query string) (int, string) {
	lowerText := strings.ToLower(text)
	count := strings.Count(lowerText, query)

	if count == 0 {
		return 0, ""
	}

	// Find first match position
	matchPos := strings.Index(lowerText, query)
	if matchPos < 0 {
		return count, ""
	}

	// Build snippet: truncate to 80 chars, centering the match if past char 40
	snippet := buildSnippet(text, query, matchPos)

	return count, snippet
}

// buildSnippet creates a snippetMaxLen-char snippet with the match centered if past char 40.
func buildSnippet(text, query string, matchPos int) string {
	if len(text) <= snippetMaxLen {
		return text
	}

	// If match is past char 40, center it
	if matchPos > 40 {
		// Try to start ~20 chars before the match
		start := matchPos - 20
		if start < 0 {
			start = 0
		}
		end := start + snippetMaxLen
		if end > len(text) {
			end = len(text)
			start = end - snippetMaxLen
			if start < 0 {
				start = 0
			}
		}
		snippet := text[start:end]
		if start > 0 {
			snippet = "..." + snippet[3:]
		}
		if end < len(text) {
			snippet = snippet[:len(snippet)-3] + "..."
		}
		return snippet
	}

	// Match is early, just take first snippetMaxLen chars
	snippet := text[:snippetMaxLen]
	if len(text) > snippetMaxLen {
		snippet = snippet[:snippetMaxLen-3] + "..."
	}
	return snippet
}
