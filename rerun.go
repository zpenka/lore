package lore

import (
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"
)

// rerunDoneMsg is dispatched after the spawned claude subprocess exits, or
// fails to launch. It carries any error so the model can surface it (or
// quit, which is what v1 does).
type rerunDoneMsg struct {
	err error
}

// rerunClaude returns a tea.Cmd that runs the claude CLI with the given
// prompt and cwd via tea.ExecProcess. ExecProcess suspends bubbletea's
// renderer and input handling, hands the terminal to the child cleanly,
// and resumes after the child exits. Without it, claude would render over
// lore's alt-screen and inputs would collide.
func rerunClaude(prompt, cwd string) tea.Cmd {
	claudePath, err := exec.LookPath("claude")
	if err != nil {
		return func() tea.Msg { return rerunDoneMsg{err: err} }
	}
	cmd := exec.Command(claudePath, prompt)
	cmd.Dir = cwd
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return rerunDoneMsg{err: err}
	})
}

// resumeClaude returns a tea.Cmd that resumes an existing Claude session by ID
// using `claude --resume <id>`. ExecProcess suspends the renderer and hands
// the terminal to the child process cleanly.
func resumeClaude(sessionID, cwd string) tea.Cmd {
	claudePath, err := exec.LookPath("claude")
	if err != nil {
		return func() tea.Msg { return rerunDoneMsg{err: err} }
	}
	cmd := exec.Command(claudePath, "--resume", sessionID)
	cmd.Dir = cwd
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return rerunDoneMsg{err: err}
	})
}
