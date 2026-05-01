package lore

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// Session is one Claude Code conversation transcript on disk.
type Session struct {
	ID        string    // sessionId from the first user event
	Path      string    // absolute path to the .jsonl file
	CWD       string    // working directory the session was launched from
	Project   string    // basename of CWD
	Branch    string    // gitBranch at session start
	Slug      string    // human-readable session label
	Timestamp time.Time // timestamp of the first user event
}

type rawUserEvent struct {
	Type      string `json:"type"`
	SessionID string `json:"sessionId"`
	Timestamp string `json:"timestamp"`
	CWD       string `json:"cwd"`
	GitBranch string `json:"gitBranch"`
	Slug      string `json:"slug"`
}

// parseSessionMetadata returns metadata extracted from the first "user" event
// in the JSONL stream. Malformed lines and non-user events preceding the first
// user event are tolerated. Returns an error if no user event is found.
func parseSessionMetadata(r io.Reader) (Session, error) {
	s := bufio.NewScanner(r)
	s.Buffer(make([]byte, 64*1024), 16*1024*1024)
	for s.Scan() {
		var ev rawUserEvent
		if err := json.Unmarshal(s.Bytes(), &ev); err != nil {
			continue
		}
		if ev.Type != "user" {
			continue
		}
		ts, err := time.Parse(time.RFC3339Nano, ev.Timestamp)
		if err != nil {
			return Session{}, err
		}
		return Session{
			ID:        ev.SessionID,
			CWD:       ev.CWD,
			Project:   filepath.Base(ev.CWD),
			Branch:    ev.GitBranch,
			Slug:      ev.Slug,
			Timestamp: ts,
		}, nil
	}
	if err := s.Err(); err != nil {
		return Session{}, err
	}
	return Session{}, errors.New("no user event found")
}

// scanSessions walks rootDir for *.jsonl files, parses each one's metadata,
// and returns sessions sorted by timestamp (newest first). Files that can't
// be parsed are skipped silently — a corrupt transcript shouldn't break the
// whole list.
func scanSessions(rootDir string) ([]Session, error) {
	var sessions []Session
	err := filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || filepath.Ext(path) != ".jsonl" {
			return nil
		}
		f, ferr := os.Open(path)
		if ferr != nil {
			return nil
		}
		defer f.Close()
		meta, perr := parseSessionMetadata(f)
		if perr != nil {
			return nil
		}
		meta.Path = path
		sessions = append(sessions, meta)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].Timestamp.After(sessions[j].Timestamp)
	})
	return sessions, nil
}
