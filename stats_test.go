package lore

import (
	"os"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// ----- parseSessionStats tests -----

func TestParseSessionStats_Empty(t *testing.T) {
	r := strings.NewReader("")
	stats, err := parseSessionStats(r)
	if err != nil {
		t.Fatalf("parseSessionStats empty: %v", err)
	}
	if stats.InputTokens != 0 || stats.OutputTokens != 0 {
		t.Errorf("empty: got %+v, want all zeros", stats)
	}
}

func TestParseSessionStats_SingleAssistant(t *testing.T) {
	jsonl := `{"type":"user","sessionId":"s1","timestamp":"2026-05-01T10:00:00Z","cwd":"/test","gitBranch":"main","slug":"test","message":{"content":"hello"}}
{"type":"assistant","sessionId":"s1","timestamp":"2026-05-01T10:00:01Z","message":{"id":"m1","model":"claude-opus-4-6","content":[{"type":"text","text":"hi"}],"usage":{"input_tokens":100,"output_tokens":50,"cache_creation_input_tokens":20,"cache_read_input_tokens":10}}}
`
	stats, err := parseSessionStats(strings.NewReader(jsonl))
	if err != nil {
		t.Fatalf("parseSessionStats: %v", err)
	}
	if stats.InputTokens != 100 {
		t.Errorf("InputTokens = %d, want 100", stats.InputTokens)
	}
	if stats.OutputTokens != 50 {
		t.Errorf("OutputTokens = %d, want 50", stats.OutputTokens)
	}
	if stats.CacheWriteTokens != 20 {
		t.Errorf("CacheWriteTokens = %d, want 20", stats.CacheWriteTokens)
	}
	if stats.CacheReadTokens != 10 {
		t.Errorf("CacheReadTokens = %d, want 10", stats.CacheReadTokens)
	}
	if stats.Model != "claude-opus-4-6" {
		t.Errorf("Model = %q, want 'claude-opus-4-6'", stats.Model)
	}
}

func TestParseSessionStats_MultipleAssistants_Sums(t *testing.T) {
	jsonl := `{"type":"assistant","sessionId":"s1","timestamp":"2026-05-01T10:00:01Z","message":{"model":"claude-sonnet-4-6","content":[],"usage":{"input_tokens":100,"output_tokens":50,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}}
{"type":"assistant","sessionId":"s1","timestamp":"2026-05-01T10:00:02Z","message":{"model":"claude-sonnet-4-6","content":[],"usage":{"input_tokens":200,"output_tokens":75,"cache_creation_input_tokens":10,"cache_read_input_tokens":5}}}
`
	stats, err := parseSessionStats(strings.NewReader(jsonl))
	if err != nil {
		t.Fatalf("parseSessionStats: %v", err)
	}
	if stats.InputTokens != 300 {
		t.Errorf("InputTokens = %d, want 300", stats.InputTokens)
	}
	if stats.OutputTokens != 125 {
		t.Errorf("OutputTokens = %d, want 125", stats.OutputTokens)
	}
	if stats.CacheWriteTokens != 10 {
		t.Errorf("CacheWriteTokens = %d, want 10", stats.CacheWriteTokens)
	}
	if stats.CacheReadTokens != 5 {
		t.Errorf("CacheReadTokens = %d, want 5", stats.CacheReadTokens)
	}
}

func TestParseSessionStats_SkipsNonAssistant(t *testing.T) {
	// Only user events — should produce zero stats
	jsonl := `{"type":"user","sessionId":"s1","timestamp":"2026-05-01T10:00:00Z","cwd":"/test","gitBranch":"main","slug":"test","message":{"content":"hello"}}
`
	stats, err := parseSessionStats(strings.NewReader(jsonl))
	if err != nil {
		t.Fatalf("parseSessionStats: %v", err)
	}
	if stats.InputTokens != 0 || stats.OutputTokens != 0 {
		t.Errorf("user-only: got non-zero stats: %+v", stats)
	}
}

func TestParseSessionStats_MalformedLineSkipped(t *testing.T) {
	jsonl := `not-json
{"type":"assistant","sessionId":"s1","timestamp":"2026-05-01T10:00:01Z","message":{"model":"claude-haiku-4","content":[],"usage":{"input_tokens":50,"output_tokens":25,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}}
`
	stats, err := parseSessionStats(strings.NewReader(jsonl))
	if err != nil {
		t.Fatalf("parseSessionStats: %v", err)
	}
	if stats.InputTokens != 50 {
		t.Errorf("InputTokens = %d, want 50", stats.InputTokens)
	}
}

// ----- estimateCost tests -----

func TestEstimateCost_Opus(t *testing.T) {
	stats := SessionStats{
		Model:        "claude-opus-4-6",
		InputTokens:  1_000_000, // 1M tokens → $15
		OutputTokens: 1_000_000, // 1M tokens → $75
	}
	cost := estimateCost(stats)
	want := 90.0
	if cost < want*0.99 || cost > want*1.01 {
		t.Errorf("Opus cost = %.4f, want ~%.4f", cost, want)
	}
}

func TestEstimateCost_Sonnet(t *testing.T) {
	stats := SessionStats{
		Model:        "claude-sonnet-4-6",
		InputTokens:  1_000_000, // 1M → $3
		OutputTokens: 1_000_000, // 1M → $15
	}
	cost := estimateCost(stats)
	want := 18.0
	if cost < want*0.99 || cost > want*1.01 {
		t.Errorf("Sonnet cost = %.4f, want ~%.4f", cost, want)
	}
}

func TestEstimateCost_Haiku(t *testing.T) {
	stats := SessionStats{
		Model:        "claude-haiku-4",
		InputTokens:  1_000_000, // 1M → $0.80
		OutputTokens: 1_000_000, // 1M → $4
	}
	cost := estimateCost(stats)
	want := 4.8
	if cost < want*0.99 || cost > want*1.01 {
		t.Errorf("Haiku cost = %.4f, want ~%.4f", cost, want)
	}
}

func TestEstimateCost_CacheReadDiscount(t *testing.T) {
	// Cache reads should cost 10% of input price
	stats := SessionStats{
		Model:           "claude-sonnet-4-6",
		InputTokens:     0,
		OutputTokens:    0,
		CacheReadTokens: 1_000_000, // 1M cache reads → $3 * 0.1 = $0.30
	}
	cost := estimateCost(stats)
	want := 0.30
	if cost < want*0.99 || cost > want*1.01 {
		t.Errorf("cache read cost = %.4f, want ~%.4f", cost, want)
	}
}

func TestEstimateCost_UnknownModel_Zero(t *testing.T) {
	stats := SessionStats{
		Model:        "unknown-model-xyz",
		InputTokens:  1000,
		OutputTokens: 500,
	}
	cost := estimateCost(stats)
	if cost != 0 {
		t.Errorf("unknown model cost = %f, want 0", cost)
	}
}

func TestEstimateCost_EmptyModel_Zero(t *testing.T) {
	stats := SessionStats{
		Model:        "",
		InputTokens:  1000,
		OutputTokens: 500,
	}
	cost := estimateCost(stats)
	if cost != 0 {
		t.Errorf("empty model cost = %f, want 0", cost)
	}
}

// ----- formatTokenCount tests -----

func TestModel_StatsMode_HReturnsToList(t *testing.T) {
	m := loadedModel("a")
	m.mode = modeStats
	m.statsData = []statsRow{{Session: Session{ID: "a"}}}
	m.statsCursor = 0

	next, _ := m.Update(keyMsg("h"))
	nm := next.(model)
	if nm.mode != modeList {
		t.Errorf("after 'h' in stats mode: mode = %d, want modeList (%d)", nm.mode, modeList)
	}
}

func TestModel_StatsMode_LeftReturnsToList(t *testing.T) {
	m := loadedModel("a")
	m.mode = modeStats
	m.statsData = []statsRow{{Session: Session{ID: "a"}}}
	m.statsCursor = 0

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	nm := next.(model)
	if nm.mode != modeList {
		t.Errorf("after 'left' in stats mode: mode = %d, want modeList (%d)", nm.mode, modeList)
	}
}

func TestFormatTokenCount(t *testing.T) {
	tests := []struct {
		n    int
		want string
	}{
		{0, "0"},
		{999, "999"},
		{1000, "1.0k"},
		{1500, "1.5k"},
		{12300, "12.3k"},
		{999999, "1000.0k"},
		{1_000_000, "1.0M"},
		{2_500_000, "2.5M"},
	}
	for _, tt := range tests {
		got := formatTokenCount(tt.n)
		if got != tt.want {
			t.Errorf("formatTokenCount(%d) = %q, want %q", tt.n, got, tt.want)
		}
	}
}

// ----- computeStatsRows tests -----

func TestComputeStatsRows_SumsTokensFromFile(t *testing.T) {
	jsonl := `{"type":"assistant","message":{"model":"claude-sonnet-4-6","content":[],"usage":{"input_tokens":100,"output_tokens":50,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}}
{"type":"assistant","message":{"model":"claude-sonnet-4-6","content":[],"usage":{"input_tokens":200,"output_tokens":75,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}}
`
	tmp := t.TempDir()
	fpath := tmp + "/sess.jsonl"
	if err := os.WriteFile(fpath, []byte(jsonl), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	rows := computeStatsRows([]Session{{Path: fpath}})
	if len(rows) != 1 {
		t.Fatalf("computeStatsRows returned %d rows, want 1", len(rows))
	}
	if rows[0].Stats.InputTokens != 300 {
		t.Errorf("InputTokens = %d, want 300", rows[0].Stats.InputTokens)
	}
	if rows[0].Stats.OutputTokens != 125 {
		t.Errorf("OutputTokens = %d, want 125", rows[0].Stats.OutputTokens)
	}
	if rows[0].Stats.EstimatedCostUSD <= 0 {
		t.Errorf("EstimatedCostUSD = %f, want > 0", rows[0].Stats.EstimatedCostUSD)
	}
}

func TestComputeStatsRows_MissingFileProducesEmptyStats(t *testing.T) {
	rows := computeStatsRows([]Session{{Path: "/nonexistent/path/sess.jsonl"}})
	if len(rows) != 1 {
		t.Fatalf("computeStatsRows returned %d rows, want 1", len(rows))
	}
	if rows[0].Stats.InputTokens != 0 {
		t.Errorf("InputTokens = %d, want 0 for missing file", rows[0].Stats.InputTokens)
	}
	if rows[0].Stats.OutputTokens != 0 {
		t.Errorf("OutputTokens = %d, want 0 for missing file", rows[0].Stats.OutputTokens)
	}
}

// ----- LORE_PRICING_FILE override tests -----

func TestEstimateCost_PricingFileOverride(t *testing.T) {
	// Write a temp pricing file with a custom rate (opus at $1/mtok in, $2/mtok out)
	pricingJSON := `[{"substr":"opus","input_per_mtok":1.0,"output_per_mtok":2.0,"cache_read_fraction":0.0}]`
	tmp := t.TempDir()
	pf := tmp + "/custom_pricing.json"
	if err := os.WriteFile(pf, []byte(pricingJSON), 0o644); err != nil {
		t.Fatalf("write pricing file: %v", err)
	}
	t.Setenv("LORE_PRICING_FILE", pf)
	resetPricingOnce()

	stats := SessionStats{
		Model:        "claude-opus-4",
		InputTokens:  1_000_000,
		OutputTokens: 1_000_000,
	}
	got := estimateCost(stats)
	want := 3.0 // $1 in + $2 out
	if got != want {
		t.Errorf("estimateCost with override = %.4f, want %.4f", got, want)
	}
}
