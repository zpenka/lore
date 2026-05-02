package lore

import (
	"bufio"
	"encoding/json"
	"io"
	"strings"
)

// Mode constants for the model state machine
const (
	modeList = iota
	modeDetail
)

// turn represents a single exchange in the detail view.
// kind is "user", "asst" (assistant text), or "tool" (tool use).
type turn struct {
	kind string // "user", "asst", or "tool"
	body string // the rendered text/snippet
}

// rawEvent is the generic JSON event structure from JSONL
type rawEvent struct {
	Type    string          `json:"type"`
	Message *rawMessage     `json:"message,omitempty"`
	Content json.RawMessage `json:"content,omitempty"` // For direct content field if present
}

type rawMessage struct {
	Content interface{} `json:"content"` // Can be string or []interface{}
}

// rawContentBlock represents a single block in message.content array
type rawContentBlock struct {
	Type  string      `json:"type"` // "text", "tool_use", "thinking", "tool_result"
	Text  string      `json:"text,omitempty"`
	Name  string      `json:"name,omitempty"`
	Input interface{} `json:"input,omitempty"`
}

// parseTurnsFromJSONL reads a JSONL stream and extracts turns.
// Returns a slice of turn structs and any error encountered during parsing.
func parseTurnsFromJSONL(r io.Reader) ([]turn, error) {
	var turns []turn
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), 16*1024*1024)

	for scanner.Scan() {
		var ev rawEvent
		if err := json.Unmarshal(scanner.Bytes(), &ev); err != nil {
			// Malformed line; skip
			continue
		}

		// Extract turns based on event type
		switch ev.Type {
		case "user":
			if userTurns := extractUserTurns(&ev); len(userTurns) > 0 {
				turns = append(turns, userTurns...)
			}
		case "assistant":
			if asstTurns := extractAssistantTurns(&ev); len(asstTurns) > 0 {
				turns = append(turns, asstTurns...)
			}
			// All other event types ignored
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return turns, nil
}

// extractUserTurns parses user event and produces zero or more turns.
// Returns empty if the content is purely tool_result blocks.
func extractUserTurns(ev *rawEvent) []turn {
	if ev.Message == nil || ev.Message.Content == nil {
		return nil
	}

	content := ev.Message.Content
	var text string

	// Content can be a string or an array
	switch c := content.(type) {
	case string:
		text = c
	case []interface{}:
		// Array of content blocks
		blocks := c

		// Check if all blocks are tool_result (if so, skip)
		allToolResults := len(blocks) > 0
		var textParts []string
		for _, b := range blocks {
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

		// If all blocks are tool_result, skip this user event
		if allToolResults && len(blocks) > 0 {
			return nil
		}

		text = strings.Join(textParts, " ")
	default:
		return nil
	}

	if text == "" {
		return nil
	}

	return []turn{{kind: "user", body: text}}
}

// extractAssistantTurns parses assistant event and produces turns for each content block.
// Thinking blocks are skipped.
func extractAssistantTurns(ev *rawEvent) []turn {
	if ev.Message == nil || ev.Message.Content == nil {
		return nil
	}

	content := ev.Message.Content
	blocks, ok := content.([]interface{})
	if !ok {
		return nil
	}

	var turns []turn
	for _, b := range blocks {
		blockMap, ok := b.(map[string]interface{})
		if !ok {
			continue
		}

		blockType, ok := blockMap["type"].(string)
		if !ok {
			continue
		}

		switch blockType {
		case "text":
			if text, ok := blockMap["text"].(string); ok && text != "" {
				turns = append(turns, turn{kind: "asst", body: text})
			}
		case "tool_use":
			if toolTurn := extractToolTurn(blockMap); toolTurn != nil {
				turns = append(turns, *toolTurn)
			}
		case "thinking":
			// Skip thinking blocks
		}
	}

	return turns
}

// extractToolTurn produces a turn from a tool_use block.
// Returns nil if extraction fails.
func extractToolTurn(blockMap map[string]interface{}) *turn {
	name, ok := blockMap["name"].(string)
	if !ok {
		return nil
	}

	input, ok := blockMap["input"]
	if !ok {
		input = ""
	}

	snippet := toolSnippet(name, input)
	body := name + " " + snippet

	return &turn{kind: "tool", body: body}
}

// toolSnippet extracts a short snippet from tool input based on the tool name.
// Preference: command (Bash) > file_path (Read/Edit/Write) > query (Grep/WebSearch) > description (Task)
// Falls back to JSON marshal + truncate.
func toolSnippet(name string, input interface{}) string {
	inputMap, ok := input.(map[string]interface{})
	if !ok {
		return ""
	}

	// Try preference fields in order
	if cmd, ok := inputMap["command"].(string); ok && cmd != "" {
		return quote(truncate(cmd, 60))
	}
	if path, ok := inputMap["file_path"].(string); ok && path != "" {
		return quote(truncate(path, 60))
	}
	if query, ok := inputMap["query"].(string); ok && query != "" {
		return quote(truncate(query, 60))
	}
	if desc, ok := inputMap["description"].(string); ok && desc != "" {
		return quote(truncate(desc, 60))
	}

	// Fallback: marshal input and truncate
	data, err := json.Marshal(input)
	if err != nil {
		return ""
	}
	return quote(truncate(string(data), 60))
}

// quote wraps s in double quotes
func quote(s string) string {
	return `"` + s + `"`
}

// truncate limits s to max runes, adding "…" if truncated
func truncate(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	if max <= 1 {
		return string(runes[:max])
	}
	return string(runes[:max-1]) + "…"
}
