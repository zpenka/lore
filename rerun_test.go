package lore

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
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

// fakeRerunFn is an injectable rerunFn that immediately yields a rerunDoneMsg
// with the given error, without spawning any real process.
func fakeRerunFn(rerunErr error) func(prompt, cwd string) tea.Cmd {
	return func(prompt, cwd string) tea.Cmd {
		return func() tea.Msg { return rerunDoneMsg{err: rerunErr} }
	}
}

// rerunModel builds a model that is already in modeRerun with a fake rerunFn.
func rerunModel(rerunErr error) model {
	m := loadedModel("a", "b")
	m.mode = modeRerun
	m.rerunPrompt = "do the thing"
	m.rerunCWD = "/some/cwd"
	m.rerunFn = fakeRerunFn(rerunErr)
	return m
}

func TestModel_RerunMode_HReturnsToDetail(t *testing.T) {
	m := rerunModel(nil)
	next, _ := m.Update(keyMsg("h"))
	nm := next.(model)
	if nm.mode != modeDetail {
		t.Errorf("after 'h' in rerun mode: mode = %d, want modeDetail (%d)", nm.mode, modeDetail)
	}
}

func TestModel_RerunMode_LeftReturnsToDetail(t *testing.T) {
	m := rerunModel(nil)
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	nm := next.(model)
	if nm.mode != modeDetail {
		t.Errorf("after 'left' in rerun mode: mode = %d, want modeDetail (%d)", nm.mode, modeDetail)
	}
}

// TestRerunDoneMsg_Success_ReturnsToList checks that a successful re-run
// (err == nil) moves the model back to modeList and returns a non-nil cmd
// (the loadSessionsCmd reload).
func TestRerunDoneMsg_Success_ReturnsToList(t *testing.T) {
	m := rerunModel(nil)

	// Simulate pressing enter to trigger the re-run, then immediately
	// dispatch the rerunDoneMsg that the fake rerunFn yields.
	next, execCmd := m.Update(keyMsg("enter"))
	m = next.(model)
	if execCmd == nil {
		t.Fatal("enter in rerun mode returned nil cmd")
	}
	doneMsg := execCmd()

	next, reloadCmd := m.Update(doneMsg)
	m = next.(model)

	if m.mode != modeList {
		t.Errorf("after successful rerunDoneMsg: mode = %d, want modeList (%d)", m.mode, modeList)
	}
	if reloadCmd == nil {
		t.Error("after successful rerunDoneMsg: expected non-nil reload cmd, got nil")
	}
	// Verify the reload cmd produces a sessionsLoadedMsg (i.e. it is a loadSessionsCmd).
	msg := reloadCmd()
	if _, ok := msg.(sessionsLoadedMsg); !ok {
		t.Errorf("reload cmd produced %T, want sessionsLoadedMsg", msg)
	}
}

// TestRerunDoneMsg_Error_ReturnsToListWithFlash checks that a failed re-run
// (err != nil) moves the model back to modeList and sets a flashMsg containing
// the error text.
func TestRerunDoneMsg_Error_ReturnsToListWithFlash(t *testing.T) {
	rerunErr := fmt.Errorf("claude: exit status 1")
	m := rerunModel(rerunErr)

	next, execCmd := m.Update(keyMsg("enter"))
	m = next.(model)
	if execCmd == nil {
		t.Fatal("enter in rerun mode returned nil cmd")
	}
	doneMsg := execCmd()

	next, reloadCmd := m.Update(doneMsg)
	m = next.(model)

	if m.mode != modeList {
		t.Errorf("after failed rerunDoneMsg: mode = %d, want modeList (%d)", m.mode, modeList)
	}
	if reloadCmd == nil {
		t.Error("after failed rerunDoneMsg: expected non-nil reload cmd, got nil")
	}
	if !strings.Contains(m.flashMsg, "re-run failed") {
		t.Errorf("flashMsg = %q, want it to contain 're-run failed'", m.flashMsg)
	}
	if !strings.Contains(m.flashMsg, rerunErr.Error()) {
		t.Errorf("flashMsg = %q, want it to contain error text %q", m.flashMsg, rerunErr.Error())
	}
}
