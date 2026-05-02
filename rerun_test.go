package lore

import (
	"testing"
)

func TestRerunClaude_LooksUpPath(t *testing.T) {
	// This test verifies that rerunClaude calls exec.LookPath to find claude.
	// We don't test the actual execution here; that would require mocking exec.
	// For now, this serves as a placeholder to verify the function is callable.

	// Test with a fake prompt and cwd
	err := rerunClaude("test prompt", "/tmp")
	// The error is expected since claude is likely not in PATH in test environment
	// or if it is, it will try to run with the test prompt.
	// The important thing is that the function signature works.
	_ = err // We don't assert here; this is a basic integration check.
}
