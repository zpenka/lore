package lore

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

// Bookmarks are stored as a small JSON object keyed by session ID. Only
// "true" entries are written; toggling a bookmark off removes the key
// rather than persisting an explicit false. The file lives next to the
// FTS5 search index in the user's cache dir.

// bookmarksFile returns the path to the bookmarks JSON, respecting LORE_CACHE_DIR.
func bookmarksFile() (string, error) {
	dir, err := resolveCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "bookmarks.json"), nil
}

// loadBookmarks reads the bookmarks JSON from path. A missing file returns
// an empty map (not an error). Malformed JSON returns an error.
func loadBookmarks(path string) (map[string]bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return map[string]bool{}, nil
		}
		return nil, err
	}
	var raw map[string]bool
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	out := make(map[string]bool, len(raw))
	for k, v := range raw {
		if v {
			out[k] = true
		}
	}
	return out, nil
}

// saveBookmarks writes the bookmarks map to path, creating any missing
// parent directories. Only entries whose value is true are persisted.
func saveBookmarks(path string, bookmarks map[string]bool) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	persist := make(map[string]bool, len(bookmarks))
	for k, v := range bookmarks {
		if v {
			persist[k] = true
		}
	}
	data, err := json.MarshalIndent(persist, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// toggleBookmark flips the entry for sessionID in place and returns the
// new state (true = now bookmarked, false = no longer bookmarked).
func toggleBookmark(bookmarks map[string]bool, sessionID string) bool {
	if bookmarks[sessionID] {
		delete(bookmarks, sessionID)
		return false
	}
	bookmarks[sessionID] = true
	return true
}
