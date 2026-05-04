package lore

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// SessionStats holds aggregated token usage for a single session.
type SessionStats struct {
	InputTokens      int
	OutputTokens     int
	CacheReadTokens  int
	CacheWriteTokens int
	Model            string
	EstimatedCostUSD float64
}

// statsRow pairs a session with its computed stats for display.
type statsRow struct {
	Session Session
	Stats   SessionStats
}

// rawAssistantUsage mirrors the usage object inside an assistant message.
type rawAssistantUsage struct {
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens"`
}

// rawAssistantMessage is the message object inside an assistant event.
type rawAssistantMessage struct {
	Model   string            `json:"model"`
	Usage   rawAssistantUsage `json:"usage"`
	Content interface{}       `json:"content"`
}

// rawAssistantEvent is the top-level event struct for assistant events.
type rawAssistantEvent struct {
	Type    string               `json:"type"`
	Message *rawAssistantMessage `json:"message,omitempty"`
}

// parseSessionStats reads a JSONL stream and sums token usage across all
// assistant events. The model field is taken from the last assistant event
// that has a non-empty model string.
func parseSessionStats(r io.Reader) (SessionStats, error) {
	var stats SessionStats
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), 16*1024*1024)

	for scanner.Scan() {
		var ev rawAssistantEvent
		if err := json.Unmarshal(scanner.Bytes(), &ev); err != nil {
			// Malformed line — skip
			continue
		}
		if ev.Type != "assistant" || ev.Message == nil {
			continue
		}
		msg := ev.Message
		stats.InputTokens += msg.Usage.InputTokens
		stats.OutputTokens += msg.Usage.OutputTokens
		stats.CacheWriteTokens += msg.Usage.CacheCreationInputTokens
		stats.CacheReadTokens += msg.Usage.CacheReadInputTokens
		if msg.Model != "" {
			stats.Model = msg.Model
		}
	}

	if err := scanner.Err(); err != nil {
		return stats, err
	}
	return stats, nil
}

// modelPricing holds per-million-token rates for a model family.
type modelPricing struct {
	inputPerMTok      float64 // $ per 1M input tokens
	outputPerMTok     float64 // $ per 1M output tokens
	cacheReadFraction float64 // fraction of input price for cache reads
}

// pricingTable maps model name substrings to pricing.
// Matched by checking if the model string contains the key.
var pricingTable = []struct {
	substr  string
	pricing modelPricing
}{
	{"opus", modelPricing{inputPerMTok: 15.0, outputPerMTok: 75.0, cacheReadFraction: 0.1}},
	{"sonnet", modelPricing{inputPerMTok: 3.0, outputPerMTok: 15.0, cacheReadFraction: 0.1}},
	{"haiku", modelPricing{inputPerMTok: 0.80, outputPerMTok: 4.0, cacheReadFraction: 0.1}},
}

// estimateCost returns an estimated USD cost for the given session stats.
// Returns 0 if the model is unknown or empty.
func estimateCost(stats SessionStats) float64 {
	if stats.Model == "" {
		return 0
	}
	lower := strings.ToLower(stats.Model)
	for _, entry := range pricingTable {
		if strings.Contains(lower, entry.substr) {
			p := entry.pricing
			cost := float64(stats.InputTokens)/1_000_000*p.inputPerMTok +
				float64(stats.OutputTokens)/1_000_000*p.outputPerMTok +
				float64(stats.CacheReadTokens)/1_000_000*p.inputPerMTok*p.cacheReadFraction +
				float64(stats.CacheWriteTokens)/1_000_000*p.inputPerMTok
			return cost
		}
	}
	return 0
}

// formatTokenCount formats an integer token count with k/M suffix.
// Values under 1000 are rendered as-is; >= 1000 get "k"; >= 1M get "M".
func formatTokenCount(n int) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	if n < 1_000_000 {
		return fmt.Sprintf("%.1fk", float64(n)/1000)
	}
	return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
}
