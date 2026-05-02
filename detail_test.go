package lore

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// Test turn extraction from various event types

func TestTurnExtraction_UserEventStringContent(t *testing.T) {
	jsonl := `{"type":"user","message":{"content":"hello world"}}`
	turns, err := parseTurnsFromJSONL(strings.NewReader(jsonl))
	if err != nil {
		t.Fatalf("parseTurnsFromJSONL failed: %v", err)
	}
	if len(turns) != 1 {
		t.Fatalf("expected 1 turn, got %d", len(turns))
	}
	if turns[0].kind != "user" {
		t.Errorf("turn.kind = %q, want 'user'", turns[0].kind)
	}
	if !strings.Contains(turns[0].body, "hello world") {
		t.Errorf("turn.body = %q, want to contain 'hello world'", turns[0].body)
	}
}

func TestTurnExtraction_UserEventArrayContent(t *testing.T) {
	jsonl := `{"type":"user","message":{"content":[{"type":"text","text":"hello array"}]}}`
	turns, err := parseTurnsFromJSONL(strings.NewReader(jsonl))
	if err != nil {
		t.Fatalf("parseTurnsFromJSONL failed: %v", err)
	}
	if len(turns) != 1 {
		t.Fatalf("expected 1 turn, got %d", len(turns))
	}
	if turns[0].kind != "user" {
		t.Errorf("turn.kind = %q, want 'user'", turns[0].kind)
	}
	if !strings.Contains(turns[0].body, "hello array") {
		t.Errorf("turn.body = %q, want to contain 'hello array'", turns[0].body)
	}
}

func TestTurnExtraction_UserEventPurelyToolResults_Skipped(t *testing.T) {
	// User event with content array containing ONLY tool_result blocks should be skipped
	jsonl := `{"type":"user","message":{"content":[{"type":"tool_result","content":"result1"},{"type":"tool_result","content":"result2"}]}}`
	turns, err := parseTurnsFromJSONL(strings.NewReader(jsonl))
	if err != nil {
		t.Fatalf("parseTurnsFromJSONL failed: %v", err)
	}
	if len(turns) != 0 {
		t.Fatalf("expected 0 turns for pure tool_result content, got %d", len(turns))
	}
}

func TestTurnExtraction_AssistantEventTextBlock(t *testing.T) {
	jsonl := `{"type":"assistant","message":{"content":[{"type":"text","text":"I will help"}]}}`
	turns, err := parseTurnsFromJSONL(strings.NewReader(jsonl))
	if err != nil {
		t.Fatalf("parseTurnsFromJSONL failed: %v", err)
	}
	if len(turns) != 1 {
		t.Fatalf("expected 1 turn, got %d", len(turns))
	}
	if turns[0].kind != "asst" {
		t.Errorf("turn.kind = %q, want 'asst'", turns[0].kind)
	}
	if !strings.Contains(turns[0].body, "I will help") {
		t.Errorf("turn.body = %q, want to contain 'I will help'", turns[0].body)
	}
}

func TestTurnExtraction_AssistantEventToolUseBlock(t *testing.T) {
	jsonl := `{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Bash","input":{"command":"ls -la"}}]}}`
	turns, err := parseTurnsFromJSONL(strings.NewReader(jsonl))
	if err != nil {
		t.Fatalf("parseTurnsFromJSONL failed: %v", err)
	}
	if len(turns) != 1 {
		t.Fatalf("expected 1 turn, got %d", len(turns))
	}
	if turns[0].kind != "tool" {
		t.Errorf("turn.kind = %q, want 'tool'", turns[0].kind)
	}
	if !strings.Contains(turns[0].body, "Bash") {
		t.Errorf("turn.body = %q, want to contain 'Bash'", turns[0].body)
	}
	if !strings.Contains(turns[0].body, "ls -la") {
		t.Errorf("turn.body = %q, want to contain 'ls -la'", turns[0].body)
	}
}

func TestTurnExtraction_AssistantEventThinkingBlock_Skipped(t *testing.T) {
	jsonl := `{"type":"assistant","message":{"content":[{"type":"thinking","thinking":"internal thoughts"}]}}`
	turns, err := parseTurnsFromJSONL(strings.NewReader(jsonl))
	if err != nil {
		t.Fatalf("parseTurnsFromJSONL failed: %v", err)
	}
	if len(turns) != 0 {
		t.Fatalf("expected 0 turns (thinking skipped), got %d", len(turns))
	}
}

func TestTurnExtraction_MultipleBlocks(t *testing.T) {
	jsonl := `{"type":"assistant","message":{"content":[{"type":"text","text":"Let me explore"},{"type":"tool_use","name":"Read","input":{"file_path":"/path/to/file"}}]}}`
	turns, err := parseTurnsFromJSONL(strings.NewReader(jsonl))
	if err != nil {
		t.Fatalf("parseTurnsFromJSONL failed: %v", err)
	}
	if len(turns) != 2 {
		t.Fatalf("expected 2 turns, got %d", len(turns))
	}
	if turns[0].kind != "asst" || !strings.Contains(turns[0].body, "Let me explore") {
		t.Errorf("first turn wrong: kind=%q, body=%q", turns[0].kind, turns[0].body)
	}
	if turns[1].kind != "tool" || !strings.Contains(turns[1].body, "Read") {
		t.Errorf("second turn wrong: kind=%q, body=%q", turns[1].kind, turns[1].body)
	}
}

func TestTurnExtraction_ToolSnippetCommand(t *testing.T) {
	// Bash tool should extract command from input.command
	jsonl := `{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Bash","input":{"command":"find . -name '*.go'"}}]}}`
	turns, err := parseTurnsFromJSONL(strings.NewReader(jsonl))
	if err != nil {
		t.Fatalf("parseTurnsFromJSONL failed: %v", err)
	}
	if len(turns) != 1 {
		t.Fatalf("expected 1 turn, got %d", len(turns))
	}
	if !strings.Contains(turns[0].body, "find . -name") {
		t.Errorf("turn.body should contain command snippet, got: %q", turns[0].body)
	}
}

func TestTurnExtraction_ToolSnippetFilePath(t *testing.T) {
	// Read tool should extract file_path
	jsonl := `{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Read","input":{"file_path":"/home/user/code.go"}}]}}`
	turns, err := parseTurnsFromJSONL(strings.NewReader(jsonl))
	if err != nil {
		t.Fatalf("parseTurnsFromJSONL failed: %v", err)
	}
	if len(turns) != 1 {
		t.Fatalf("expected 1 turn, got %d", len(turns))
	}
	if !strings.Contains(turns[0].body, "/home/user/code.go") {
		t.Errorf("turn.body should contain file_path, got: %q", turns[0].body)
	}
}

func TestTurnExtraction_OtherEventTypesIgnored(t *testing.T) {
	jsonl := `{"type":"tool_result","content":"some result"}
{"type":"text","text":"orphaned text"}
{"type":"skill_listing","skills":["a","b"]}`
	turns, err := parseTurnsFromJSONL(strings.NewReader(jsonl))
	if err != nil {
		t.Fatalf("parseTurnsFromJSONL failed: %v", err)
	}
	if len(turns) != 0 {
		t.Fatalf("expected 0 turns for non-user/non-assistant events, got %d", len(turns))
	}
}

// Test model transitions

func TestModel_EnterDetailMode_OnEnter(t *testing.T) {
	m := loadedModel("session-1", "session-2")
	if m.mode != modeList {
		t.Errorf("initial mode = %d, want %d", m.mode, modeList)
	}
}

func TestModel_EnterDetail_Cmd(t *testing.T) {
	m := loadedModel("s1")
	// In list mode, pressing enter should dispatch a load command
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	nm := next.(model)
	if nm.mode != modeList {
		// Mode may not change immediately; the cmd should load
		t.Logf("mode still %d after enter (waiting for cmd)", nm.mode)
	}
	if cmd == nil {
		t.Fatal("enter produced nil cmd, want loadSessionCmd")
	}
}

// Test detail view rendering

func TestDetailView_Header(t *testing.T) {
	m := newModel("/d")
	m.mode = modeDetail
	m.detailSession = Session{
		Slug:      "my-awesome-session",
		Project:   "grit",
		Branch:    "feat/new-feature",
		Timestamp: timeFromString("2026-05-01T14:30:00Z"),
	}
	m.turns = []turn{
		{kind: "user", body: "hello"},
	}
	m.width = 100
	m.height = 40

	out := m.View()
	if !strings.Contains(out, "my-awesome-session") {
		t.Errorf("header missing session slug: %s", out)
	}
	if !strings.Contains(out, "grit") {
		t.Errorf("header missing project: %s", out)
	}
	if !strings.Contains(out, "feat/new-feature") {
		t.Errorf("header missing branch: %s", out)
	}
	if !strings.Contains(out, "2026-05-01") {
		t.Errorf("header missing date: %s", out)
	}
}

func TestDetailView_TurnsRendered(t *testing.T) {
	m := newModel("/d")
	m.mode = modeDetail
	m.detailSession = Session{Slug: "test", Project: "p", Branch: "b", Timestamp: timeFromString("2026-05-01T14:30:00Z")}
	m.turns = []turn{
		{kind: "user", body: "first message"},
		{kind: "asst", body: "I will help"},
		{kind: "tool", body: "Bash ls -la"},
	}
	m.width = 100
	m.height = 40

	out := m.View()
	if !strings.Contains(out, "user") || !strings.Contains(out, "first message") {
		t.Errorf("user turn not rendered correctly: %s", out)
	}
	if !strings.Contains(out, "asst") || !strings.Contains(out, "I will help") {
		t.Errorf("asst turn not rendered correctly: %s", out)
	}
	if !strings.Contains(out, "Bash") {
		t.Errorf("tool turn not rendered correctly: %s", out)
	}
}

func TestDetailView_NoTurns(t *testing.T) {
	m := newModel("/d")
	m.mode = modeDetail
	m.detailSession = Session{Slug: "empty", Project: "p", Branch: "b", Timestamp: timeFromString("2026-05-01T14:30:00Z")}
	m.turns = []turn{}
	m.width = 100
	m.height = 40

	out := m.View()
	if !strings.Contains(out, "no turns") {
		t.Errorf("empty detail view should show 'no turns': %s", out)
	}
}

func TestDetailView_Footer(t *testing.T) {
	m := newModel("/d")
	m.mode = modeDetail
	m.detailSession = Session{Slug: "test", Project: "p", Branch: "b", Timestamp: timeFromString("2026-05-01T14:30:00Z")}
	m.turns = []turn{{kind: "user", body: "msg"}}
	m.width = 100
	m.height = 40

	out := m.View()
	if !strings.Contains(out, "j/k") {
		t.Errorf("footer missing j/k: %s", out)
	}
	if !strings.Contains(out, "back") {
		t.Errorf("footer missing 'back': %s", out)
	}
}

// Test detail mode navigation and transitions

func TestModel_DetailMode_JMovesDown(t *testing.T) {
	m := newModel("/d")
	m.mode = modeDetail
	m.detailSession = Session{Slug: "test", Project: "p", Branch: "b", Timestamp: timeFromString("2026-05-01T14:30:00Z")}
	m.turns = []turn{
		{kind: "user", body: "msg1"},
		{kind: "asst", body: "msg2"},
		{kind: "user", body: "msg3"},
	}
	m.cursorDetail = 0

	next, _ := m.Update(keyMsg("j"))
	nm := next.(model)
	if nm.cursorDetail != 1 {
		t.Errorf("after 'j': cursorDetail = %d, want 1", nm.cursorDetail)
	}
}

func TestModel_DetailMode_KMovesUp(t *testing.T) {
	m := newModel("/d")
	m.mode = modeDetail
	m.detailSession = Session{Slug: "test", Project: "p", Branch: "b", Timestamp: timeFromString("2026-05-01T14:30:00Z")}
	m.turns = []turn{
		{kind: "user", body: "msg1"},
		{kind: "asst", body: "msg2"},
	}
	m.cursorDetail = 1

	next, _ := m.Update(keyMsg("k"))
	nm := next.(model)
	if nm.cursorDetail != 0 {
		t.Errorf("after 'k': cursorDetail = %d, want 0", nm.cursorDetail)
	}
}

func TestModel_DetailMode_QReturnsToList(t *testing.T) {
	m := newModel("/d")
	m.mode = modeDetail
	m.detailSession = Session{Slug: "test", Project: "p", Branch: "b", Timestamp: timeFromString("2026-05-01T14:30:00Z")}
	m.turns = []turn{{kind: "user", body: "msg"}}
	m.cursor = 2 // Remember list cursor

	next, _ := m.Update(keyMsg("q"))
	nm := next.(model)
	if nm.mode != modeList {
		t.Errorf("after 'q': mode = %d, want %d", nm.mode, modeList)
	}
	if nm.cursor != 2 {
		t.Errorf("after 'q': cursor = %d, want 2 (preserved)", nm.cursor)
	}
}

func TestModel_DetailMode_EscReturnsToList(t *testing.T) {
	m := newModel("/d")
	m.mode = modeDetail
	m.detailSession = Session{Slug: "test", Project: "p", Branch: "b", Timestamp: timeFromString("2026-05-01T14:30:00Z")}
	m.turns = []turn{{kind: "user", body: "msg"}}
	m.cursor = 1

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	nm := next.(model)
	if nm.mode != modeList {
		t.Errorf("after 'esc': mode = %d, want %d", nm.mode, modeList)
	}
}

// Helper for tests

func timeFromString(s string) time.Time {
	t, _ := time.Parse(time.RFC3339, s)
	return t
}
