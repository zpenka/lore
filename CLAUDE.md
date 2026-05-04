# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**lore** is a keyboard-driven TUI (Terminal User Interface) for browsing Claude Code session history. It reads session transcripts from `~/.claude/projects/<encoded-cwd>/*.jsonl` and provides rich navigation, filtering, and search across sessions.

Current status: **v0.5.0 — Phases 1–5 and most of Phase 7 complete.** Implemented:

- Session list (3.1) with relative-time bucketing.
- Inline project (`p`), branch (`b`), and fuzzy (`f`) filters with fuzzy ranking.
- Session detail (3.2) with collapsible tool turns, thinking toggle, copy-prompt, re-run, diff rendering for `Edit` / `Write` tool calls, and turn position indicator (`turn N/M`).
- Full-text search (3.3): SQLite FTS5 index (Phase 5a) with linear-scan fallback.
- Project view (3.4) grouped by branch.
- Re-run (3.5) via `tea.ExecProcess` so the spawned `claude` owns the TTY cleanly. Returns to the session list on exit and surfaces spawn errors.
- Usage stats panel (Phase 5c, `S` key): token counts and estimated cost per session.
- Configurable projects directory (`--dir` flag, `LORE_PROJECTS_DIR` env var).
- Help overlay (`?`) with mode-specific keybindings.
- Per-mode viewport scrolling, mode-specific footers, and one-shot flash messages for no-op keys.

Outstanding: sidechain handling (the only remaining Phase 7 item). The Phase 6 repo split has already happened — this is the standalone `github.com/zpenka/lore` module.

See `DESIGN.md` for the full product vision and phasing roadmap.

## Build & Run

```bash
# Install from source
go install github.com/zpenka/lore/cmd/lore@latest

# Or build locally
go build ./cmd/lore
./lore

# Or run directly
go run ./cmd/lore

# Point lore at a non-default projects dir
./lore --dir /path/to/projects
LORE_PROJECTS_DIR=/path/to/projects ./lore
```

The binary reads from `~/.claude/projects/` (created by Claude Code) by default, scans for `.jsonl` session transcripts, and displays them in a sortable list grouped by recency (today, yesterday, this week, etc.). The projects directory can be overridden via the `--dir` flag (highest precedence) or the `LORE_PROJECTS_DIR` environment variable; resolution lives in `lore.go::resolveProjectsDir`.

The FTS5 search index is cached at `<os.UserCacheDir>/lore/index.db` (e.g. `~/.cache/lore/index.db` on Linux, `~/Library/Caches/lore/index.db` on macOS) and is populated lazily on first search.

## Tests

```bash
# Run all tests in the lore package
go test ./...

# Run tests for a specific file
go test -run TestBucket ./...

# Run one test
go test -run TestModel_Initial

# Run with verbose output
go test -v ./...

# Run only smoke tests (test against real session files)
go test -run TestSmoke ./...
```

Tests use a lightweight fixture pattern (`TestFixture`, `keyMsg()` helpers) for Bubble Tea model testing.

## Architecture

### Core Model: Bubble Tea TUI

The project uses [Bubble Tea](https://github.com/charmbracelet/bubbletea) for the TUI framework and [Lipgloss](https://github.com/charmbracelet/lipgloss) for styling.

**Main files:**

- `lore.go`: Entry point. `Run()` parses flags (`-v`/`--version`, `--dir`), resolves the projects dir (`--dir` > `LORE_PROJECTS_DIR` > `~/.claude/projects`), and starts the Bubble Tea program.
- `model.go`: The Bubble Tea `model` struct and per-mode key dispatchers (`handleListKey`, `handleDetailKey`, `handleSearchKey`, `handleProjectKey`, `handleRerunKey`, `handleStatsKey`, `handleFilterEntryKey`). Also lazy-opens the FTS5 index on first search.
- `render.go`: `View()`, mode-specific renderers (`renderListView`, `renderDetailView`, `renderSearchView`, `renderStatsView`, etc.), the help overlay, and all Lipgloss styles.
- `session.go`: `Session` struct and `scanSessions()` / `parseSessionMetadata()` — reads only the first `user` event of each `.jsonl`.
- `bucket.go`: `timeBucket()` returns labels like "today", "yesterday", "this week" for relative-time grouping.
- `detail.go`: Mode constants (`modeList`, `modeDetail`, `modeSearch`, `modeProject`, `modeRerun`, `modeStats`), `turn` struct, `parseTurnsFromJSONL()`, and assistant/tool block extraction.
- `search.go`: `searchSessions()` — linear-scan full-text search returning `SearchHit`s ranked by hit count. Used as the fallback path when the FTS5 index is unavailable.
- `index.go`: SQLite FTS5 search index. `OpenIndex(cacheDir)`, `Index.Sync(projectsDir)` for incremental mtime-based reindexing, `Index.Search(query)` for ranked FTS5 lookups, and `extractSessionText()` for indexable content.
- `stats.go`: Usage-stats data layer. `parseSessionStats()` sums `assistant.message.usage` token counts; `estimateCost()` applies a per-million-token pricing table (Opus / Sonnet / Haiku); `formatTokenCount()` adds k/M suffixes.
- `project.go`: `groupByBranch()` and project-view rendering helpers.
- `filter.go`: `fuzzyFilterCandidates()` — wraps `github.com/sahilm/fuzzy` for the `p` / `b` inline filters.
- `clipboard.go`: `copyToClipboard()` — tries pbcopy, wl-copy, then xclip.
- `rerun.go`: `rerunClaude()` returns a `tea.Cmd` that wraps `tea.ExecProcess` so the child `claude` invocation takes over the terminal cleanly.
- `viewport.go`: `clampOffset()` and `sliceLines()` — the two primitives every renderer uses for edge-snap scrolling.
- `wrap.go`: `wrapText()` — soft-wraps multi-line turn bodies so viewport math reflects what's actually rendered.
- `cmd/lore/main.go`: Thin wrapper that calls `lore.Run()`.

### Data Model

**Session**: One complete Claude Code transcript on disk.
- `ID`: UUID from the session's first "user" event.
- `Path`: Absolute path to the `.jsonl` file.
- `CWD`: Working directory the session was launched from.
- `Project`: Basename of CWD (e.g. "grit", "dotfiles").
- `Branch`: Git branch at session start.
- `Slug`: Human-readable session label auto-generated by Claude Code.
- `Timestamp`: Extracted from the first user event.

**Model**: Bubble Tea state machine. The `mode` field switches between `modeList`, `modeDetail`, `modeSearch`, `modeProject`, `modeRerun`, and `modeStats`. Each mode has its own cursor and viewport offset (`listOffset`, `detailOffset`, `searchOffset`, `projectOffset`, `statsOffset`) so navigating away and back preserves position.

Notable per-mode state:
- **List**: `sessions`, `visibleSessions`, `cursor`, `filterMode` / `filterText` / `appliedFilterMode` for the inline `p` / `b` / `f` filters.
- **Detail**: `detailSession`, `turns`, `cursorDetail`, `expandedTurns` (which tool turns are unfolded), `showThinking`, `justCopied`.
- **Search**: `searchMode` (entry vs. results), `searchQuery`, `searchResults`, `searchCursor`. The FTS5 `index` is lazy-opened on the first `enter` press and falls back to linear scan on miss/error.
- **Project**: `projectCWD`, `projectSessions`, `projectCursor`.
- **Rerun**: `rerunPrompt`, `rerunCWD`, plus an injectable `rerunFn` so tests can substitute `tea.ExecProcess`.
- **Stats**: `statsData` (slice of `statsRow`), `statsCursor`, `statsOffset`. Computed by `computeStatsRows` from the in-memory session list when `S` is pressed.

The model also injects `clipboardFn` (default `copyToClipboard`) and `rerunFn` (default `rerunClaude`) so tests can swap real-system effects for fakes. The `projectsDir` field carries the resolved projects path so the FTS5 sync knows where to walk.

### Session File Format

Sessions are read-only JSONL files in `~/.claude/projects/<encoded-cwd>/*.jsonl`. Each line is a JSON event with a `type` field. The tool extracts metadata from the first "user" event and reads no further (cheap, scales to large transcripts).

Example first user event:
```json
{
  "type": "user",
  "sessionId": "c95ad6fe-7304-45e6-a726-80780e34099c",
  "timestamp": "2026-05-01T22:37:40.568Z",
  "cwd": "/home/user/grit",
  "gitBranch": "claude/plan-new-cli-tool",
  "slug": "hey-so-i-want-twinkly-aho"
}
```

## Development Notes

### Dependencies

- `github.com/charmbracelet/bubbletea`: TUI framework.
- `github.com/charmbracelet/lipgloss`: Styling for headers, dividers, selected rows, etc.
- `github.com/sahilm/fuzzy`: Fuzzy ranking for the `p` / `b` / `f` filters.
- `modernc.org/sqlite`: Pure-Go SQLite driver backing the Phase 5a FTS5 search index.

Don't add other dependencies without updating `DESIGN.md` first.

### Keyboard Navigation

The full key map is also surfaced in-app via the `?` overlay. Authoritative reference: `renderHelpOverlay` in `render.go`.

**List mode** (`modeList`):
- `j` / `k`, `↑` / `↓`: Move cursor.
- `g` / `G`: Jump to top / bottom.
- `enter`, `l`, `→`: Open the highlighted session in detail view.
- `p`: Inline project filter (type query, `enter` to apply, `esc` to cancel).
- `b`: Inline branch filter.
- `f`: Fuzzy filter across slug, project, and branch simultaneously.
- `P`: Open the project view scoped to the selected session's CWD.
- `S`: Open the usage stats panel.
- `/`: Enter full-text search.
- `esc`: Clear an applied filter.
- `?`: Show help overlay.
- `q`: Quit.

**Detail mode** (`modeDetail`):
- `j` / `k`, `g` / `G`: Move / jump.
- `space`: Expand or collapse a tool turn (the cursor must be on one).
- `t`: Toggle thinking blocks (hidden by default).
- `y`: Copy the user prompt at-or-before the cursor to the clipboard.
- `r`: Re-run with the current user prompt (enters re-run mode).
- `esc` / `q` / `h` / `←`: Back to list.

**Search mode** (`modeSearch`):
- Entry: type to build query, `enter` to run, `esc` to cancel.
- Results: `j` / `k`, `g` / `G`, `enter` to open, `/` to edit query, `esc` back.

**Project mode** (`modeProject`): `j` / `k`, `g` / `G`, `enter`, `esc` / `q`. Sessions are grouped by branch with the latest branch first.

**Re-run mode** (`modeRerun`): `enter` to spawn `claude` with the chosen prompt and CWD (lore returns to the session list when `claude` exits); `esc` / `q` to cancel and return to detail.

**Stats mode** (`modeStats`): `j` / `k`, `g` / `G` to navigate the per-session table; `esc` / `q` to return to the list. Columns: project · branch · model · input tokens · output tokens · estimated cost. Token counts use `k` / `M` suffixes; cost is computed from a built-in pricing table for Opus / Sonnet / Haiku families and shown as `--` for unknown models.

### Testing Strategy

- **Unit tests**: Test model state transitions, time bucketing, session parsing in isolation.
- **Smoke tests** (`internal_smoke_test.go`): End-to-end against real or fixture session files in `~/.claude/projects/`.
- **Fixtures**: `newTestSession()` and similar helpers in test files.

Tests import `bubbletea` directly to send messages (e.g., `keyMsg()`) to the model's `Update()` method.

### Rendering & View

`View()` dispatches by `m.mode` to the per-mode renderer. Each renderer follows the same shape: header, divider, body lines (sliced through `clampOffset` + `sliceLines` from `viewport.go`), divider, footer. The `?` help overlay short-circuits the entire view.

Styling is done via Lipgloss `NewStyle()` instances defined at the top of `render.go`.

Body math goes through one of `listBodyLines`, `detailBodyLines`, `searchBodyLines`, or `projectBodyLines`. Each returns `(lines []string, cursorLine int)` so the viewport can edge-snap the offset to keep the cursor visible.

### Phasing Notes

| Phase | Status |
|---|---|
| 1 — Session list (3.1) | ✅ Complete |
| 2 — Session detail (3.2) | ✅ Complete (incl. tool expansion, thinking toggle, diff rendering) |
| 3 — Search v1 linear scan (3.3) | ✅ Complete (kept as fallback) |
| 4 — Project view (3.4) and re-run (3.5) | ✅ Complete |
| 5a — SQLite FTS5 search index | ✅ Complete (`index.go`, lazy-opened on first search, falls back to linear scan) |
| 5b — List-level fuzzy matching (`f` key) | ✅ Complete |
| 5c — Cost/usage stats panel | ✅ Complete (`stats.go`, `S` from list mode) |
| 6 — Repo split into `github.com/zpenka/lore` | ✅ Done (this is that repo) |
| 7 — Quality-of-life | 🔶 Partial (back-nav, re-run return-to-list, turn indicator, configurable projects dir done; sidechain handling remaining) |

## Repo Layout

```
lore/
├── CLAUDE.md           # This file
├── DESIGN.md           # Product vision & phasing
├── README.md           # High-level project description
├── go.mod, go.sum
├── lore.go             # Entry point, Run(), resolveProjectsDir()
├── model.go            # Bubble Tea model + per-mode key dispatchers
├── render.go           # View(), per-mode renderers, help overlay, Lipgloss styles
├── session.go          # Session struct, scanSessions(), parseSessionMetadata()
├── detail.go           # Mode constants, turn struct, JSONL → turns parser
├── search.go           # Linear-scan full-text search (fallback path)
├── index.go            # SQLite FTS5 search index (Phase 5a)
├── stats.go            # Token-usage parsing + cost estimation (Phase 5c)
├── project.go          # Branch grouping + project view rendering
├── filter.go           # Fuzzy-ranked p/b/f inline filter
├── clipboard.go        # pbcopy / wl-copy / xclip wrapper
├── rerun.go            # tea.ExecProcess wrapper for the claude child process
├── viewport.go         # clampOffset() + sliceLines() scrolling primitives
├── wrap.go             # wrapText() for multi-line turn bodies
├── bucket.go           # timeBucket(), relative-time grouping
├── *_test.go           # Unit and smoke tests
├── .github/
│   ├── pull_request_template.md
│   └── workflows/ci.yml
└── cmd/lore/
    └── main.go         # Binary wrapper, calls lore.Run()
```

## Common Tasks

### Add a new key binding

1. Identify which mode owns the key. Edit the matching `handle*Key` method in `model.go` (`handleListKey`, `handleDetailKey`, `handleSearchKey`, `handleProjectKey`, `handleRerunKey`).
2. Update the relevant `renderHelpOverlay` block in `render.go` so the `?` overlay matches.
3. If the key affects the footer, update the corresponding `render*Footer` in `render.go` / `project.go`.
4. Test via the matching `TestModel_*` test in `model_test.go` (or the per-feature test file: `search_test.go`, `project_test.go`, `rerun_test.go`, etc.).

### Change a view's display

All rendering lives in `render.go` (plus `project.go::renderProjectView` for project mode). The body math for each mode is in `listBodyLines` / `detailBodyLines` / `searchBodyLines` / `projectBodyLines`. Styles are Lipgloss `NewStyle()` instances at the top of `render.go`.

### Parse a new field from session metadata

1. Add a field to `rawUserEvent` in `session.go` (JSON struct).
2. Update `parseSessionMetadata()` to extract it into the `Session` struct.
3. Update tests in `session_test.go`.

### Test against real session files

Run smoke tests:
```bash
go test -run TestSmoke -v ./...
```

These read actual `.jsonl` files from `~/.claude/projects/`. If you have no real sessions, create fixture files in a temp directory and point the test there.

## Test-driven development (required)

All PRs on this repository must follow **red → green → refactor**:

1. **Red**: Write a failing test first. Commit it with a clear message explaining what behavior you're testing.
2. **Green**: Write the minimal production code to make the test pass. Commit separately.
3. **Refactor** (optional): Improve code style or structure without changing behavior. Tests must remain green.

Do not write production code without a failing test driving it. The CI gate (`.github/workflows/ci.yml`) enforces:

```bash
go test -race -cover ./...
```

Per-package coverage must be ≥80% (CI fails the job if any package other than `cmd/lore` falls below this threshold). The full command to check locally:

```bash
go test -race -coverprofile=coverage.out -covermode=atomic ./...
go tool cover -func=coverage.out
```

Packages excluded from the coverage gate:
- `cmd/lore`: thin wrapper calling `lore.Run()` (integration code).

The gate aggregates per-package statement coverage from `coverage.out` (i.e. statement-weighted, not function-averaged). To debug a failure locally, re-run the awk block in `.github/workflows/ci.yml` against your `coverage.out`.

## Agent contract

Future sub-agents working on this repo must adhere to these rules:

- **Worktree**: Create a worktree, branch from `main`. Do not commit directly to `main`.
- **Red/green/refactor**: One commit per phase. The red commit's tests MUST fail in isolation (verify: `git show HEAD:lore_test.go | go test ./...` after committing the test).
- **Pre-PR checks**: Run locally before opening a PR:
  - `go test -race -cover ./...` (verify ≥80% on touched packages)
  - `go vet ./...` (must be clean)
  - `gofmt -l .` (must be clean; no files should be listed)
- **No shortcuts**: Do not skip hooks (`--no-verify`), do not lower coverage, do not add dependencies not listed in `DESIGN.md`.
- **PR body**: Use `gh pr create` with a body that explicitly names the commits: "Red commit: abc1234, green commit: def5678, refactor (optional): ...".
- **Blocking issues**: If stuck, stop and surface the blocker clearly in the PR or via a GitHub issue. Do not work around constraints.

