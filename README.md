# lore

A keyboard-driven TUI for browsing Claude Code session history.

## Install

```bash
brew install zpenka/lore/lore
```

Requires [Claude Code](https://claude.ai/code) — lore reads the session transcripts it writes to `~/.claude/projects/`.

**Alternative (Go toolchain):**
```bash
go install github.com/zpenka/lore/cmd/lore@latest
```

## What it does

`lore` gives you a fast, keyboard-driven interface to browse, search, and re-run your Claude Code sessions:

- **Session list** with relative-time bucketing, project/branch/fuzzy filters, and bookmark support
- **Session detail** with collapsible tool turns, diff rendering for edits, and sidechain expansion for Agent turns
- **Full-text search** backed by SQLite FTS5 (linear-scan fallback)
- **Project view** grouped by branch
- **Re-run** — replay any user prompt in the original working directory
- **Usage stats** — token counts and estimated cost per session
- **Timeline heatmap** — 8-week activity grid; click a day to filter the list

## Navigation

Press `?` in any mode for the full keymap. Highlights:

- **List**: `j`/`k` move, `d`/`u` half-page, `g`/`G` jump, `enter` open, `p`/`b` filter project/branch, `f` fuzzy filter, `m` bookmark, `M` bookmark-only filter, `P` project view, `/` search, `S` usage stats, `T` timeline heatmap, `q` quit.
- **Detail**: `d`/`u` half-page, `space` expand a tool turn (Agent turns with sidechains load the sub-conversation inline), `y` copy the nearest user prompt, `r` re-run that prompt, `m` bookmark this session, `/` search, `esc`/`q`/`h`/`←` back.
- **Search**: type → `enter` to run, `j`/`k` or `d`/`u` through hits, `enter` to open, `esc`/`q`/`h`/`←` back.
- **Project**: `j`/`k`, `d`/`u`, `enter` to open, `esc`/`q`/`h`/`←` back. Sessions are grouped by branch.
- **Re-run**: `enter` to spawn `claude` with the chosen prompt and CWD; `esc`/`q`/`h`/`←` to cancel.
- **Stats**: `j`/`k`, `g`/`G` to navigate; `esc`/`q`/`h`/`←` back. Columns: project · branch · model · input/output tokens · estimated cost.
- **Timeline**: `h`/`←` and `l`/`→` move the cursor across days in an 8-week activity heatmap; `enter` filters the list to the highlighted day; `esc`/`q` back.

## Configuration

`lore` reads sessions from `~/.claude/projects/` by default. Override with either:

- `--dir <path>` flag (highest precedence)
- `LORE_PROJECTS_DIR` environment variable

The FTS5 search index and bookmarks file are cached under the platform user cache dir (`~/.cache/lore/` on Linux, `~/Library/Caches/lore/` on macOS).

## For contributors

See `CLAUDE.md` for development setup, architecture, TDD requirements, and the agent contract.

Full product vision and roadmap: `DESIGN.md`
