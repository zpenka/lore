package lore

import (
	"testing"
)

// TestRerunClaude_ReturnsCmd verifies rerunClaude always returns a non-nil
// tea.Cmd. We don't exercise the happy path (which would shell out to a
// real claude process and conflict with the test runner's TTY); the
// dependency-injection point is exercised via the rerunFn override in
// model_test.go.
func TestRerunClaude_ReturnsCmd(t *testing.T) {
	cmd := rerunClaude("test prompt", "/tmp")
	if cmd == nil {
		t.Fatal("rerunClaude returned a nil tea.Cmd")
	}
	// Calling the Cmd is safe regardless of whether claude is on PATH:
	//   - if it's not on PATH, we return a Cmd that yields a rerunDoneMsg
	//     with the lookup error. Calling it once is a pure function call.
	//   - if it IS on PATH, tea.ExecProcess returns a Cmd that yields an
	//     internal exec-msg which is only acted on by an active bubbletea
	//     Program. Calling it outside a Program does NOT actually exec.
	// Either way, no external process is launched by this test.
	msg := cmd()
	if msg == nil {
		t.Fatal("rerunClaude's Cmd returned nil msg")
	}
}
