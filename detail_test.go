package lore

import (
	"os"
	"path/filepath"
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

func TestTurnExtraction_AssistantEventThinkingBlock_Parsed(t *testing.T) {
	jsonl := `{"type":"assistant","message":{"content":[{"type":"thinking","thinking":"internal thoughts"}]}}`
	turns, err := parseTurnsFromJSONL(strings.NewReader(jsonl))
	if err != nil {
		t.Fatalf("parseTurnsFromJSONL failed: %v", err)
	}
	if len(turns) != 1 {
		t.Fatalf("expected 1 turn (thinking parsed), got %d", len(turns))
	}
	if turns[0].kind != "thinking" {
		t.Errorf("turn.kind = %q, want 'thinking'", turns[0].kind)
	}
	if !strings.Contains(turns[0].body, "internal thoughts") {
		t.Errorf("turn.body = %q, want to contain 'internal thoughts'", turns[0].body)
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

func TestTurnExtraction_ToolSnippetQuery(t *testing.T) {
	// WebSearch tool should extract query
	jsonl := `{"type":"assistant","message":{"content":[{"type":"tool_use","name":"WebSearch","input":{"query":"golang defer"}}]}}`
	turns, err := parseTurnsFromJSONL(strings.NewReader(jsonl))
	if err != nil {
		t.Fatalf("parseTurnsFromJSONL failed: %v", err)
	}
	if len(turns) != 1 {
		t.Fatalf("expected 1 turn, got %d", len(turns))
	}
	if !strings.Contains(turns[0].body, "golang defer") {
		t.Errorf("turn.body should contain query, got: %q", turns[0].body)
	}
}

func TestTurnExtraction_ToolSnippetDescription(t *testing.T) {
	// Task tool should extract description
	jsonl := `{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Task","input":{"description":"run npm test"}}]}}`
	turns, err := parseTurnsFromJSONL(strings.NewReader(jsonl))
	if err != nil {
		t.Fatalf("parseTurnsFromJSONL failed: %v", err)
	}
	if len(turns) != 1 {
		t.Fatalf("expected 1 turn, got %d", len(turns))
	}
	if !strings.Contains(turns[0].body, "run npm test") {
		t.Errorf("turn.body should contain description, got: %q", turns[0].body)
	}
}

func TestTurnExtraction_ToolSnippetFallback(t *testing.T) {
	// Tool with no preferred fields should marshal input
	jsonl := `{"type":"assistant","message":{"content":[{"type":"tool_use","name":"CustomTool","input":{"foo":"bar","baz":123}}]}}`
	turns, err := parseTurnsFromJSONL(strings.NewReader(jsonl))
	if err != nil {
		t.Fatalf("parseTurnsFromJSONL failed: %v", err)
	}
	if len(turns) != 1 {
		t.Fatalf("expected 1 turn, got %d", len(turns))
	}
	if !strings.Contains(turns[0].body, "CustomTool") {
		t.Errorf("turn.body should contain tool name, got: %q", turns[0].body)
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

// Sidechain tests

func TestTurnExtraction_AgentToolHasToolUseID(t *testing.T) {
	jsonl := `{"type":"assistant","message":{"content":[{"type":"tool_use","id":"toolu_abc123","name":"Agent","input":{"prompt":"do something","description":"test agent"}}]}}`
	turns, err := parseTurnsFromJSONL(strings.NewReader(jsonl))
	if err != nil {
		t.Fatalf("parseTurnsFromJSONL failed: %v", err)
	}
	if len(turns) != 1 {
		t.Fatalf("expected 1 turn, got %d", len(turns))
	}
	if turns[0].toolUseID != "toolu_abc123" {
		t.Errorf("toolUseID = %q, want %q", turns[0].toolUseID, "toolu_abc123")
	}
}

func TestTurnExtraction_NonAgentToolUseIDAlsoCaptured(t *testing.T) {
	jsonl := `{"type":"assistant","message":{"content":[{"type":"tool_use","id":"toolu_xyz","name":"Bash","input":{"command":"ls"}}]}}`
	turns, err := parseTurnsFromJSONL(strings.NewReader(jsonl))
	if err != nil {
		t.Fatalf("parseTurnsFromJSONL failed: %v", err)
	}
	if len(turns) != 1 {
		t.Fatalf("expected 1 turn, got %d", len(turns))
	}
	if turns[0].toolUseID != "toolu_xyz" {
		t.Errorf("toolUseID = %q, want %q", turns[0].toolUseID, "toolu_xyz")
	}
}

func TestParseTurnsFromJSONL_LinksSidechainsByToolUseID(t *testing.T) {
	// Parent JSONL: Agent tool_use followed by a user event with agentId and tool_result
	jsonl := `{"type":"user","message":{"content":"start"},"sessionId":"sid","timestamp":"2026-05-01T10:00:00Z","cwd":"/proj","gitBranch":"main","slug":"test"}
{"type":"assistant","message":{"content":[{"type":"tool_use","id":"toolu_abc","name":"Agent","input":{"prompt":"explore the code","description":"explore"}}]}}
{"type":"user","agentId":"agent123","message":{"content":[{"type":"tool_result","tool_use_id":"toolu_abc","content":"explored successfully"}]}}`

	turns, err := parseTurnsFromJSONL(strings.NewReader(jsonl))
	if err != nil {
		t.Fatalf("parseTurnsFromJSONL failed: %v", err)
	}

	// Find the Agent tool turn
	var agentTurn *turn
	for i := range turns {
		if turns[i].kind == "tool" && strings.Contains(turns[i].body, "Agent") {
			agentTurn = &turns[i]
			break
		}
	}
	if agentTurn == nil {
		t.Fatal("no Agent tool turn found")
	}
	if agentTurn.sidechainID != "agent123" {
		t.Errorf("sidechainID = %q, want %q", agentTurn.sidechainID, "agent123")
	}
}

func TestSidechainsDir(t *testing.T) {
	got := sidechainsDir("/home/user/.claude/projects/-proj/abc-123.jsonl")
	want := "/home/user/.claude/projects/-proj/abc-123/subagents"
	if got != want {
		t.Errorf("sidechainsDir = %q, want %q", got, want)
	}
}

func TestLoadSessionTurns_LinksSidechainPaths(t *testing.T) {
	dir := t.TempDir()

	// Write parent session JSONL
	parentJSONL := `{"type":"user","message":{"content":"start"},"sessionId":"sid","timestamp":"2026-05-01T10:00:00Z","cwd":"/proj","gitBranch":"main","slug":"test"}
{"type":"assistant","message":{"content":[{"type":"tool_use","id":"toolu_abc","name":"Agent","input":{"prompt":"explore the code","description":"explore"}}]}}
{"type":"user","agentId":"agent123","message":{"content":[{"type":"tool_result","tool_use_id":"toolu_abc","content":"explored"}]}}`

	parentPath := filepath.Join(dir, "abc-123.jsonl")
	if err := os.WriteFile(parentPath, []byte(parentJSONL), 0o644); err != nil {
		t.Fatal(err)
	}

	// Write sidechain file
	sidechainDir := filepath.Join(dir, "abc-123", "subagents")
	if err := os.MkdirAll(sidechainDir, 0o755); err != nil {
		t.Fatal(err)
	}
	sidechainJSONL := `{"type":"user","isSidechain":true,"agentId":"agent123","message":{"content":"explore the code"},"sessionId":"sid","timestamp":"2026-05-01T10:00:01Z","cwd":"/proj","gitBranch":"main"}
{"type":"assistant","message":{"content":[{"type":"text","text":"I found some interesting code"}]}}`

	sidechainPath := filepath.Join(sidechainDir, "agent-agent123.jsonl")
	if err := os.WriteFile(sidechainPath, []byte(sidechainJSONL), 0o644); err != nil {
		t.Fatal(err)
	}

	turns, err := loadSessionTurns(parentPath)
	if err != nil {
		t.Fatalf("loadSessionTurns: %v", err)
	}

	var agentTurn *turn
	for i := range turns {
		if turns[i].kind == "tool" && strings.Contains(turns[i].body, "Agent") {
			agentTurn = &turns[i]
			break
		}
	}
	if agentTurn == nil {
		t.Fatal("no Agent tool turn found")
	}
	if agentTurn.sidechainPath != sidechainPath {
		t.Errorf("sidechainPath = %q, want %q", agentTurn.sidechainPath, sidechainPath)
	}
}

func TestModel_DetailMode_SpaceExpandsAgentSidechain(t *testing.T) {
	dir := t.TempDir()

	// Write sidechain file
	sidechainDir := filepath.Join(dir, "s1", "subagents")
	if err := os.MkdirAll(sidechainDir, 0o755); err != nil {
		t.Fatal(err)
	}
	sidechainJSONL := `{"type":"user","isSidechain":true,"agentId":"ag1","message":{"content":"do work"},"sessionId":"s1","timestamp":"2026-05-01T10:00:01Z","cwd":"/proj","gitBranch":"main"}
{"type":"assistant","message":{"content":[{"type":"text","text":"I did the work"}]}}`
	sidechainPath := filepath.Join(sidechainDir, "agent-ag1.jsonl")
	if err := os.WriteFile(sidechainPath, []byte(sidechainJSONL), 0o644); err != nil {
		t.Fatal(err)
	}

	m := newModel("/d")
	m.mode = modeDetail
	m.detailSession = Session{Slug: "test", Project: "p", Branch: "b", Path: filepath.Join(dir, "s1.jsonl"), Timestamp: timeFromString("2026-05-01T14:30:00Z")}
	m.turns = []turn{
		{kind: "user", body: "hello"},
		{kind: "tool", body: "Agent \"do work\"", sidechainID: "ag1", sidechainPath: sidechainPath},
		{kind: "asst", body: "done"},
	}
	m.cursorDetail = 1
	m.expandedTurns = make(map[int]bool)

	next, _ := m.Update(keyMsg(" "))
	nm := next.(model)

	if !nm.expandedTurns[1] {
		t.Error("Agent tool turn should be expanded after space")
	}
	scTurns, ok := nm.sidechainTurns[1]
	if !ok {
		t.Fatal("sidechainTurns[1] not populated after expanding Agent sidechain")
	}
	if len(scTurns) == 0 {
		t.Fatal("sidechainTurns[1] is empty, expected parsed sidechain turns")
	}
	// Should have the user and assistant turns from the sidechain
	hasUser := false
	hasAsst := false
	for _, st := range scTurns {
		if st.kind == "user" && strings.Contains(st.body, "do work") {
			hasUser = true
		}
		if st.kind == "asst" && strings.Contains(st.body, "I did the work") {
			hasAsst = true
		}
	}
	if !hasUser {
		t.Error("sidechain turns missing user turn with 'do work'")
	}
	if !hasAsst {
		t.Error("sidechain turns missing asst turn with 'I did the work'")
	}
}

func TestDetailView_SidechainIndicator(t *testing.T) {
	m := newModel("/d")
	m.mode = modeDetail
	m.detailSession = Session{Slug: "test", Project: "p", Branch: "b", Timestamp: timeFromString("2026-05-01T14:30:00Z")}
	m.turns = []turn{
		{kind: "user", body: "hello"},
		{kind: "tool", body: "Agent \"explore code\"", sidechainID: "ag1", sidechainPath: "/some/path.jsonl"},
		{kind: "asst", body: "done"},
	}
	m.cursorDetail = 0
	m.width = 100
	m.height = 40

	out := m.View()
	if !strings.Contains(out, "⧑") {
		t.Errorf("Agent turn with sidechain should show ⧑ indicator, got:\n%s", out)
	}
}

func TestDetailView_ExpandedSidechainContent(t *testing.T) {
	m := newModel("/d")
	m.mode = modeDetail
	m.detailSession = Session{Slug: "test", Project: "p", Branch: "b", Timestamp: timeFromString("2026-05-01T14:30:00Z")}
	m.turns = []turn{
		{kind: "user", body: "hello"},
		{kind: "tool", body: "Agent \"explore code\"", sidechainID: "ag1", sidechainPath: "/some/path.jsonl"},
	}
	m.cursorDetail = 1
	m.expandedTurns = map[int]bool{1: true}
	m.sidechainTurns = map[int][]turn{
		1: {
			{kind: "user", body: "explore the code"},
			{kind: "asst", body: "Here is what I found"},
			{kind: "tool", body: "Read \"main.go\""},
		},
	}
	m.width = 100
	m.height = 40

	out := m.View()
	if !strings.Contains(out, "explore the code") {
		t.Errorf("expanded sidechain should show user turn, got:\n%s", out)
	}
	if !strings.Contains(out, "Here is what I found") {
		t.Errorf("expanded sidechain should show asst turn, got:\n%s", out)
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

// Feature 1: Expand/collapse tool turns with space

func TestModel_DetailMode_SpaceExpandsToolTurn(t *testing.T) {
	m := newModel("/d")
	m.mode = modeDetail
	m.detailSession = Session{Slug: "test", Project: "p", Branch: "b", Timestamp: timeFromString("2026-05-01T14:30:00Z")}
	m.turns = []turn{
		{kind: "user", body: "hello"},
		{kind: "tool", body: "Bash \"ls -la\""},
		{kind: "asst", body: "done"},
	}
	m.cursorDetail = 1
	m.expandedTurns = make(map[int]bool) // Initialize expansion state

	next, _ := m.Update(keyMsg(" "))
	nm := next.(model)
	if !nm.expandedTurns[1] {
		t.Errorf("after space on tool turn: expandedTurns[1] = false, want true")
	}
	if nm.cursorDetail != 1 {
		t.Errorf("after space: cursorDetail = %d, want 1 (unchanged)", nm.cursorDetail)
	}
}

func TestModel_DetailMode_SpaceTogglesToolTurnExpansion(t *testing.T) {
	m := newModel("/d")
	m.mode = modeDetail
	m.detailSession = Session{Slug: "test", Project: "p", Branch: "b", Timestamp: timeFromString("2026-05-01T14:30:00Z")}
	m.turns = []turn{{kind: "tool", body: "Read \"file.go\""}}
	m.cursorDetail = 0
	m.expandedTurns = map[int]bool{0: true}

	next, _ := m.Update(keyMsg(" "))
	nm := next.(model)
	if nm.expandedTurns[0] {
		t.Errorf("after toggling expanded turn: expandedTurns[0] = true, want false")
	}
}

func TestModel_DetailMode_SpaceOnNonToolTurnNoOp(t *testing.T) {
	m := newModel("/d")
	m.mode = modeDetail
	m.detailSession = Session{Slug: "test", Project: "p", Branch: "b", Timestamp: timeFromString("2026-05-01T14:30:00Z")}
	m.turns = []turn{{kind: "user", body: "hello"}}
	m.cursorDetail = 0
	m.expandedTurns = make(map[int]bool)

	next, _ := m.Update(keyMsg(" "))
	nm := next.(model)
	if nm.expandedTurns[0] {
		t.Errorf("space on non-tool turn should be no-op; expandedTurns[0] = true, want false")
	}
}

func TestDetailView_ExpandedToolRender(t *testing.T) {
	m := newModel("/d")
	m.mode = modeDetail
	m.detailSession = Session{Slug: "test", Project: "p", Branch: "b", Timestamp: timeFromString("2026-05-01T14:30:00Z")}
	// Tool turn with structured input
	m.turns = []turn{
		{kind: "tool", body: "Read \"file.go\"", input: map[string]interface{}{"file_path": "/home/user/file.go"}},
	}
	m.cursorDetail = 0
	m.expandedTurns = map[int]bool{0: true}
	m.width = 100
	m.height = 40

	out := m.View()
	if !strings.Contains(out, "file_path") {
		t.Errorf("expanded tool view should show input fields, got:\n%s", out)
	}
	if !strings.Contains(out, "/home/user/file.go") {
		t.Errorf("expanded tool view should show input values, got:\n%s", out)
	}
}

// Thinking turns are always filtered out (content is redacted in session files).

func TestDetailView_ThinkingTurnsAlwaysHidden(t *testing.T) {
	m := newModel("/d")
	m.mode = modeDetail
	m.detailSession = Session{Slug: "test", Project: "p", Branch: "b", Timestamp: timeFromString("2026-05-01T14:30:00Z")}
	m.turns = []turn{
		{kind: "user", body: "ask"},
		{kind: "thinking", body: "internal reasoning"},
		{kind: "asst", body: "answer"},
	}
	m.cursorDetail = 0
	m.width = 100
	m.height = 40

	out := m.View()
	if strings.Contains(out, "internal reasoning") {
		t.Errorf("thinking turns should always be hidden, got:\n%s", out)
	}
}

// Feature 3: Copy turn with y

func TestModel_DetailMode_YOnFirstTurn_NoPriorUser(t *testing.T) {
	m := newModel("/d")
	m.mode = modeDetail
	m.detailSession = Session{Slug: "test", Project: "p", Branch: "b", Timestamp: timeFromString("2026-05-01T14:30:00Z")}
	m.turns = []turn{
		{kind: "asst", body: "response"},
		{kind: "user", body: "later prompt"},
	}
	m.cursorDetail = 0
	callCount := 0
	m.clipboardFn = func(s string) error {
		callCount++
		return nil
	}

	next, _ := m.Update(keyMsg("y"))
	nm := next.(model)
	if callCount > 0 {
		t.Errorf("after 'y' on asst with no prior user: clipboard called %d times, want 0", callCount)
	}
	if nm.justCopied {
		t.Errorf("after 'y' with no prior user: justCopied = true, want false")
	}
}

func TestModel_DetailMode_YCopiesUserTurnPrompt(t *testing.T) {
	m := newModel("/d")
	m.mode = modeDetail
	m.detailSession = Session{Slug: "test", Project: "p", Branch: "b", Timestamp: timeFromString("2026-05-01T14:30:00Z")}
	m.turns = []turn{
		{kind: "user", body: "the prompt"},
		{kind: "asst", body: "response"},
	}
	m.cursorDetail = 0
	m.clipboardFn = func(s string) error { return nil }
	lastCopied := ""
	m.clipboardFn = func(s string) error {
		lastCopied = s
		return nil
	}

	next, _ := m.Update(keyMsg("y"))
	nm := next.(model)
	if lastCopied != "the prompt" {
		t.Errorf("after 'y' on user turn: copied %q, want 'the prompt'", lastCopied)
	}
	if !nm.justCopied {
		t.Errorf("after 'y': justCopied = false, want true")
	}
}

func TestModel_DetailMode_YCopiesMostRecentUserBeforeTool(t *testing.T) {
	m := newModel("/d")
	m.mode = modeDetail
	m.detailSession = Session{Slug: "test", Project: "p", Branch: "b", Timestamp: timeFromString("2026-05-01T14:30:00Z")}
	m.turns = []turn{
		{kind: "user", body: "initial request"},
		{kind: "asst", body: "thinking..."},
		{kind: "tool", body: "Read file.go"},
		{kind: "asst", body: "here is file"},
	}
	m.cursorDetail = 2 // On tool turn
	copiedText := ""
	m.clipboardFn = func(s string) error {
		copiedText = s
		return nil
	}

	next, _ := m.Update(keyMsg("y"))
	nm := next.(model)
	if copiedText != "initial request" {
		t.Errorf("after 'y' on tool turn: copied %q, want 'initial request'", copiedText)
	}
	if !nm.justCopied {
		t.Errorf("after 'y': justCopied = false, want true")
	}
}

func TestModel_DetailMode_YNoOpIfNoUserTurn(t *testing.T) {
	m := newModel("/d")
	m.mode = modeDetail
	m.detailSession = Session{Slug: "test", Project: "p", Branch: "b", Timestamp: timeFromString("2026-05-01T14:30:00Z")}
	m.turns = []turn{
		{kind: "asst", body: "orphaned response"},
		{kind: "tool", body: "Read file"},
	}
	m.cursorDetail = 1
	callCount := 0
	m.clipboardFn = func(s string) error {
		callCount++
		return nil
	}

	next, _ := m.Update(keyMsg("y"))
	nm := next.(model)
	if callCount > 0 {
		t.Errorf("after 'y' with no prior user turn: clipboard was called %d times, want 0", callCount)
	}
	if nm.justCopied {
		t.Errorf("after 'y' with no user turn: justCopied = true, want false")
	}
}

func TestDetailView_FooterShowsCopiedBriefly(t *testing.T) {
	m := newModel("/d")
	m.mode = modeDetail
	m.detailSession = Session{Slug: "test", Project: "p", Branch: "b", Timestamp: timeFromString("2026-05-01T14:30:00Z")}
	m.turns = []turn{{kind: "user", body: "msg"}}
	m.cursorDetail = 0
	m.justCopied = true

	m.width = 100
	m.height = 40

	out := m.View()
	if !strings.Contains(out, "copied") {
		t.Errorf("footer should show 'copied' when justCopied=true, got:\n%s", out)
	}
}

// Helper tests for model methods

func TestModel_VisibleTurnsFiltersThinking(t *testing.T) {
	m := newModel("/d")
	m.turns = []turn{
		{kind: "user", body: "ask"},
		{kind: "thinking", body: "think"},
		{kind: "asst", body: "answer"},
	}

	visible := m.visibleTurns()
	if len(visible) != 2 {
		t.Errorf("visibleTurns: len = %d, want 2", len(visible))
	}
	if visible[0].kind != "user" || visible[1].kind != "asst" {
		t.Errorf("visibleTurns filtered wrong turns")
	}
}

func TestModel_VisibleIndexToFullIndex(t *testing.T) {
	m := newModel("/d")
	m.turns = []turn{
		{kind: "user", body: "ask"},
		{kind: "thinking", body: "think"},
		{kind: "asst", body: "answer"},
		{kind: "thinking", body: "more"},
		{kind: "tool", body: "Read file"},
	}

	// With thinking hidden, visible is [user, asst, tool] at indices [0, 2, 4]
	testCases := []struct {
		visibleIdx  int
		wantFullIdx int
	}{
		{0, 0}, // visible 0 (user) -> full 0
		{1, 2}, // visible 1 (asst) -> full 2
		{2, 4}, // visible 2 (tool) -> full 4
	}
	for _, tc := range testCases {
		got := m.visibleIndexToFullIndex(tc.visibleIdx)
		if got != tc.wantFullIdx {
			t.Errorf("visibleIndexToFullIndex(%d) = %d, want %d", tc.visibleIdx, got, tc.wantFullIdx)
		}
	}
}

func TestModel_DetailMode_JMovesDownInVisibleList(t *testing.T) {
	m := newModel("/d")
	m.mode = modeDetail
	m.detailSession = Session{Slug: "test", Project: "p", Branch: "b", Timestamp: timeFromString("2026-05-01T14:30:00Z")}
	m.turns = []turn{
		{kind: "user", body: "ask"},
		{kind: "thinking", body: "think"},
		{kind: "asst", body: "answer"},
	}
	m.cursorDetail = 0

	next, _ := m.Update(keyMsg("j"))
	nm := next.(model)
	if nm.cursorDetail != 1 {
		t.Errorf("after 'j' with thinking hidden: cursorDetail = %d, want 1", nm.cursorDetail)
	}
}

func TestModel_DetailMode_KMovesUpInVisibleList(t *testing.T) {
	m := newModel("/d")
	m.mode = modeDetail
	m.detailSession = Session{Slug: "test", Project: "p", Branch: "b", Timestamp: timeFromString("2026-05-01T14:30:00Z")}
	m.turns = []turn{
		{kind: "user", body: "ask"},
		{kind: "thinking", body: "think"},
		{kind: "asst", body: "answer"},
	}
	m.cursorDetail = 1

	next, _ := m.Update(keyMsg("k"))
	nm := next.(model)
	if nm.cursorDetail != 0 {
		t.Errorf("after 'k' with thinking hidden: cursorDetail = %d, want 0", nm.cursorDetail)
	}
}

func TestModel_DetailMode_SpaceOnToolWithInput(t *testing.T) {
	m := newModel("/d")
	m.mode = modeDetail
	m.detailSession = Session{Slug: "test", Project: "p", Branch: "b", Timestamp: timeFromString("2026-05-01T14:30:00Z")}
	m.turns = []turn{
		{kind: "tool", body: "Bash ls", input: map[string]interface{}{"command": "ls -la /home"}},
	}
	m.cursorDetail = 0
	m.expandedTurns = make(map[int]bool)

	next, _ := m.Update(keyMsg(" "))
	nm := next.(model)
	if !nm.expandedTurns[0] {
		t.Errorf("tool turn not expanded")
	}
}

func TestModel_DetailMode_CopyResetsByAnyKey(t *testing.T) {
	m := newModel("/d")
	m.mode = modeDetail
	m.detailSession = Session{Slug: "test", Project: "p", Branch: "b", Timestamp: timeFromString("2026-05-01T14:30:00Z")}
	m.turns = []turn{
		{kind: "user", body: "msg1"},
		{kind: "asst", body: "msg2"},
	}
	m.cursorDetail = 0
	m.justCopied = true

	// Any navigation key should reset justCopied
	next, _ := m.Update(keyMsg("j"))
	nm := next.(model)
	if nm.justCopied {
		t.Errorf("justCopied should be reset by 'j' key")
	}
}

func TestModel_SessionDetailLoaded_Dispatch(t *testing.T) {
	m := newModel("/d")
	m.mode = modeList
	turns := []turn{{kind: "user", body: "test"}}

	next, _ := m.Update(sessionDetailLoadedMsg{turns: turns, err: nil})
	nm := next.(model)
	if nm.mode != modeDetail {
		t.Errorf("after sessionDetailLoadedMsg: mode = %d, want %d", nm.mode, modeDetail)
	}
	if len(nm.turns) != 1 {
		t.Errorf("after sessionDetailLoadedMsg: len(turns) = %d, want 1", len(nm.turns))
	}
	if nm.detailLoading {
		t.Errorf("after sessionDetailLoadedMsg: detailLoading = true, want false")
	}
}

func TestModel_SessionDetailLoaded_WithError(t *testing.T) {
	m := newModel("/d")
	m.mode = modeList
	m.detailLoading = true

	next, _ := m.Update(sessionDetailLoadedMsg{err: errFake("parse failed")})
	nm := next.(model)
	if nm.detailErr == nil {
		t.Errorf("after sessionDetailLoadedMsg with error: detailErr is nil")
	}
	if nm.detailLoading {
		t.Errorf("detailLoading should be false after msg arrives, got true")
	}
}

func TestModel_DetailMode_SpaceResetsJustCopied(t *testing.T) {
	m := newModel("/d")
	m.mode = modeDetail
	m.detailSession = Session{Slug: "test", Project: "p", Branch: "b", Timestamp: timeFromString("2026-05-01T14:30:00Z")}
	m.turns = []turn{{kind: "tool", body: "Read file"}}
	m.cursorDetail = 0
	m.justCopied = true
	m.expandedTurns = make(map[int]bool)

	next, _ := m.Update(keyMsg(" "))
	nm := next.(model)
	if nm.justCopied {
		t.Errorf("justCopied should be reset by space key")
	}
}

func TestModel_Detail_SlashEntersSearch(t *testing.T) {
	m := loadedModel("a")
	m.mode = modeDetail
	m.turns = []turn{{kind: "user", body: "hello"}}
	m.cursorDetail = 0
	m.expandedTurns = make(map[int]bool)

	next, _ := m.Update(keyMsg("/"))
	nm := next.(model)
	if nm.mode != modeSearch {
		t.Errorf("after '/' in detail: mode = %d, want %d (modeSearch)", nm.mode, modeSearch)
	}
	if nm.searchMode != searchModeEntry {
		t.Errorf("after '/' in detail: searchMode = %d, want %d (entry)", nm.searchMode, searchModeEntry)
	}
	if nm.searchQuery != "" {
		t.Errorf("after '/' in detail: searchQuery = %q, want ''", nm.searchQuery)
	}
}

// Helper for tests

func timeFromString(s string) time.Time {
	t, _ := time.Parse(time.RFC3339, s)
	return t
}

// ----- truncate edge branches -----

func TestTruncate_MaxOne(t *testing.T) {
	// max=1: only one rune fits, no room for ellipsis
	got := truncate("abc", 1)
	if got != "a" {
		t.Errorf("truncate(%q, 1) = %q, want %q", "abc", got, "a")
	}
}

func TestTruncate_ExactFit(t *testing.T) {
	// exact fit: no truncation needed, no ellipsis
	got := truncate("ab", 2)
	if got != "ab" {
		t.Errorf("truncate(%q, 2) = %q, want %q", "ab", got, "ab")
	}
}

// FuzzParseTurnsFromJSONL fuzz-tests the JSONL turn parser.
// Seeds cover valid turns and common malformed inputs.
func FuzzParseTurnsFromJSONL(f *testing.F) {
	f.Add(`{"type":"user","message":{"content":"hello"}}`)
	f.Add(`{"type":"assistant","message":{"content":[{"type":"text","text":"hi"}]}}`)
	f.Add(``)
	f.Add(`not json`)
	f.Add("{\"type\":\"tool_use\",\"message\":{\"content\":[]}}\n{\"type\":\"user\"}")
	f.Add(`{"type":"user","message":{"content":["array","of","strings"]}}`)

	f.Fuzz(func(t *testing.T, data string) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("parseTurnsFromJSONL panicked: %v", r)
			}
		}()
		_, _ = parseTurnsFromJSONL(strings.NewReader(data))
	})
}
