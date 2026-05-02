package lore

import (
	"os"
	"os/exec"
)

// rerunClaude executes the claude CLI with the given prompt and working directory.
// It looks up claude on PATH, sets the working directory, and wires stdin/stdout/stderr
// to allow the user to interact with the process directly.
func rerunClaude(prompt, cwd string) error {
	// Look up claude on PATH
	claudePath, err := exec.LookPath("claude")
	if err != nil {
		return err
	}

	// Build the command
	cmd := exec.Command(claudePath, prompt)
	cmd.Dir = cwd
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Run and return the result
	return cmd.Run()
}
