package lore

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
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
	Query     string    // first user message text (preview for list view)
	Timestamp time.Time // timestamp of the first user event
}

type rawUserEvent struct {
	Type      string          `json:"type"`
	SessionID string          `json:"sessionId"`
	Timestamp string          `json:"timestamp"`
	CWD       string          `json:"cwd"`
	GitBranch string          `json:"gitBranch"`
	Slug      string          `json:"slug"`
	Message   json.RawMessage `json:"message"`
}

// parseSessionMetadata returns metadata extracted from the first "user" event
// in the JSONL stream. Malformed lines and non-user events preceding the first
// user event are tolerated. System-injected user events (caveats, slash commands)
// are skipped when looking for the query preview, but metadata (ID, CWD, branch)
// is taken from the first user event. Returns an error if no user event is found.
func parseSessionMetadata(r io.Reader) (Session, error) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 64*1024), 16*1024*1024)

	var sess Session
	found := false

	for sc.Scan() {
		var ev rawUserEvent
		if err := json.Unmarshal(sc.Bytes(), &ev); err != nil {
			continue
		}
		if ev.Type != "user" {
			continue
		}

		if !found {
			ts, err := time.Parse(time.RFC3339Nano, ev.Timestamp)
			if err != nil {
				return Session{}, err
			}
			sess = Session{
				ID:        ev.SessionID,
				CWD:       ev.CWD,
				Project:   filepath.Base(ev.CWD),
				Branch:    ev.GitBranch,
				Slug:      ev.Slug,
				Timestamp: ts,
			}
			found = true
		}

		q := extractQuery(ev.Message)
		if q != "" {
			sess.Query = q
			return sess, nil
		}
	}
	if err := sc.Err(); err != nil {
		return Session{}, err
	}
	if found {
		return sess, nil
	}
	return Session{}, errors.New("no user event found")
}

// scanSessions walks rootDir for *.jsonl files, parses each one's metadata,
// and returns sessions sorted by timestamp (newest first). Files that can't
// be opened or parsed are skipped; the second return value carries one
// short message per skip so the UI can surface the count to the user.
// A corrupt transcript shouldn't break the whole list.
func scanSessions(rootDir string) ([]Session, []string, error) {
	var sessions []Session
	var warnings []string
	err := filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || filepath.Ext(path) != ".jsonl" {
			return nil
		}
		if strings.Contains(path, string(filepath.Separator)+"subagents"+string(filepath.Separator)) {
			return nil
		}
		f, ferr := os.Open(path)
		if ferr != nil {
			warnings = append(warnings, fmt.Sprintf("%s: %v", path, ferr))
			return nil
		}
		defer f.Close()
		meta, perr := parseSessionMetadata(f)
		if perr != nil {
			warnings = append(warnings, fmt.Sprintf("%s: %v", path, perr))
			return nil
		}
		meta.Path = path
		sessions = append(sessions, meta)
		return nil
	})
	if err != nil {
		return nil, warnings, err
	}
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].Timestamp.After(sessions[j].Timestamp)
	})
	return sessions, warnings, nil
}

var systemTagRe = regexp.MustCompile(`(?s)<(local-command-caveat|command-name|command-message|command-args|system-reminder)(?:[^>]*)>.*?</(?:local-command-caveat|command-name|command-message|command-args|system-reminder)>`)

func extractQuery(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var msg struct {
		Content json.RawMessage `json:"content"`
	}
	if err := json.Unmarshal(raw, &msg); err != nil || len(msg.Content) == 0 {
		return ""
	}

	// Content can be a plain string or an array of content blocks.
	var s string
	if err := json.Unmarshal(msg.Content, &s); err == nil {
		return collapseWhitespace(stripSystemTags(s))
	}

	var blocks []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(msg.Content, &blocks); err == nil {
		var parts []string
		for _, b := range blocks {
			if b.Type == "text" && b.Text != "" {
				cleaned := collapseWhitespace(stripSystemTags(b.Text))
				if cleaned != "" {
					parts = append(parts, cleaned)
				}
			}
		}
		return collapseWhitespace(strings.Join(parts, " "))
	}
	return ""
}

func stripSystemTags(s string) string {
	return systemTagRe.ReplaceAllString(s, "")
}

func collapseWhitespace(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "\t", " ")
	// Collapse runs of spaces.
	for strings.Contains(s, "  ") {
		s = strings.ReplaceAll(s, "  ", " ")
	}
	return strings.TrimSpace(s)
}
