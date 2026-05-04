package lore

import (
	"bufio"
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

// Index wraps a SQLite database with FTS5 for full-text search over session transcripts.
type Index struct {
	db *sql.DB
}

const schema = `
CREATE VIRTUAL TABLE IF NOT EXISTS sessions_fts USING fts5(
	session_path,
	content,
	tokenize='porter unicode61'
);

CREATE TABLE IF NOT EXISTS session_meta (
	path TEXT PRIMARY KEY,
	mtime_ns INTEGER NOT NULL
);
`

// OpenIndex opens or creates the FTS5 index database in cacheDir/index.db.
func OpenIndex(cacheDir string) (*Index, error) {
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return nil, fmt.Errorf("create cache dir: %w", err)
	}
	dbPath := filepath.Join(cacheDir, "index.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("create schema: %w", err)
	}
	return &Index{db: db}, nil
}

// Close closes the underlying database.
func (idx *Index) Close() error {
	if idx == nil || idx.db == nil {
		return nil
	}
	return idx.db.Close()
}

// Sync walks projectsDir for .jsonl files, indexes new or changed files,
// and removes entries for deleted files.
func (idx *Index) Sync(projectsDir string) error {
	onDisk := make(map[string]int64) // path -> mtime_ns

	err := filepath.WalkDir(projectsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || filepath.Ext(path) != ".jsonl" {
			return nil
		}
		info, serr := d.Info()
		if serr != nil {
			return nil
		}
		onDisk[path] = info.ModTime().UnixNano()
		return nil
	})
	if err != nil {
		return fmt.Errorf("walk: %w", err)
	}

	inIndex := make(map[string]int64)
	rows, err := idx.db.Query("SELECT path, mtime_ns FROM session_meta")
	if err != nil {
		return fmt.Errorf("query meta: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var p string
		var mt int64
		if err := rows.Scan(&p, &mt); err != nil {
			return fmt.Errorf("scan meta: %w", err)
		}
		inIndex[p] = mt
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("rows: %w", err)
	}

	// Index new or changed files
	for path, mtimeNS := range onDisk {
		if existingMtime, ok := inIndex[path]; ok && existingMtime == mtimeNS {
			continue
		}
		if err := idx.indexFile(path, mtimeNS); err != nil {
			continue // skip corrupt files
		}
	}

	// Remove deleted files
	for path := range inIndex {
		if _, ok := onDisk[path]; !ok {
			idx.removeFile(path)
		}
	}

	return nil
}

func (idx *Index) indexFile(path string, mtimeNS int64) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	text, err := extractSessionText(data)
	if err != nil {
		return err
	}

	tx, err := idx.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Remove old entry if it exists
	tx.Exec("DELETE FROM sessions_fts WHERE session_path = ?", path)
	tx.Exec("DELETE FROM session_meta WHERE path = ?", path)

	// Insert new
	if _, err := tx.Exec("INSERT INTO sessions_fts (session_path, content) VALUES (?, ?)", path, text); err != nil {
		return err
	}
	if _, err := tx.Exec("INSERT INTO session_meta (path, mtime_ns) VALUES (?, ?)", path, mtimeNS); err != nil {
		return err
	}
	return tx.Commit()
}

func (idx *Index) removeFile(path string) {
	idx.db.Exec("DELETE FROM sessions_fts WHERE session_path = ?", path)
	idx.db.Exec("DELETE FROM session_meta WHERE path = ?", path)
}

// Search performs an FTS5 query and returns ranked SearchHits.
// Returns nil for empty queries. Falls back gracefully on errors.
func (idx *Index) Search(query string) ([]SearchHit, error) {
	if query == "" {
		return nil, nil
	}

	// Escape FTS5 special characters by quoting each term
	terms := strings.Fields(query)
	for i, t := range terms {
		terms[i] = `"` + strings.ReplaceAll(t, `"`, `""`) + `"`
	}
	ftsQuery := strings.Join(terms, " ")

	rows, err := idx.db.Query(
		"SELECT session_path, snippet(sessions_fts, 1, '', '', '...', 20), rank FROM sessions_fts WHERE content MATCH ? ORDER BY rank",
		ftsQuery,
	)
	if err != nil {
		return nil, fmt.Errorf("fts query: %w", err)
	}
	defer rows.Close()

	var hits []SearchHit
	for rows.Next() {
		var path, snippet string
		var rank float64
		if err := rows.Scan(&path, &snippet, &rank); err != nil {
			continue
		}

		f, err := os.Open(path)
		if err != nil {
			continue
		}
		meta, err := parseSessionMetadata(f)
		f.Close()
		if err != nil {
			continue
		}
		meta.Path = path

		hits = append(hits, SearchHit{
			Session:  meta,
			HitCount: 1, // FTS5 rank is used for ordering, not raw count
			Snippet:  snippet,
		})
	}
	return hits, rows.Err()
}

// extractSessionText reads JSONL data and concatenates all user and assistant
// text content for FTS5 indexing. Skips tool_use and thinking blocks.
func extractSessionText(data []byte) (string, error) {
	var parts []string
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 64*1024), 16*1024*1024)

	for scanner.Scan() {
		var ev rawEvent
		if err := json.Unmarshal(scanner.Bytes(), &ev); err != nil {
			continue
		}
		switch ev.Type {
		case "user":
			if ev.Message == nil || ev.Message.Content == nil {
				continue
			}
			switch c := ev.Message.Content.(type) {
			case string:
				parts = append(parts, c)
			case []interface{}:
				for _, b := range c {
					bm, ok := b.(map[string]interface{})
					if !ok {
						continue
					}
					if bt, _ := bm["type"].(string); bt == "text" {
						if txt, ok := bm["text"].(string); ok {
							parts = append(parts, txt)
						}
					}
				}
			}
		case "assistant":
			if ev.Message == nil || ev.Message.Content == nil {
				continue
			}
			blocks, ok := ev.Message.Content.([]interface{})
			if !ok {
				continue
			}
			for _, b := range blocks {
				bm, ok := b.(map[string]interface{})
				if !ok {
					continue
				}
				if bt, _ := bm["type"].(string); bt == "text" {
					if txt, ok := bm["text"].(string); ok {
						parts = append(parts, txt)
					}
				}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return strings.Join(parts, "\n"), nil
}

// indexCacheDir returns the platform-appropriate cache directory for the index DB.
func indexCacheDir() (string, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("user cache dir: %w", err)
	}
	return filepath.Join(cacheDir, "lore", "index.db"), nil
}
