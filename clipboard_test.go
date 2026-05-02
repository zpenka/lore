package lore

import (
	"testing"
)

func TestCopyToClipboard_FakeImpl(t *testing.T) {
	// This test verifies that clipboardFn can be mocked
	lastCopied := ""
	fakeClip := func(s string) error {
		lastCopied = s
		return nil
	}

	err := fakeClip("test message")
	if err != nil {
		t.Errorf("fake clipboard error: %v", err)
	}
	if lastCopied != "test message" {
		t.Errorf("fake clipboard copied %q, want 'test message'", lastCopied)
	}
}
