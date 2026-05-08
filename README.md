# lore

A keyboard-driven TUI for browsing Claude Code session history.

## Quick start

```bash
# Install via Homebrew (macOS / Linux)
brew install zpenka/lore/lore

# Or install via Go toolchain
go install github.com/zpenka/lore/cmd/lore@latest

# Or build from source
go build ./cmd/lore
./lore
```

The tool reads session transcripts from `~/.claude/projects/` and displays them in a sortable, navigable list.

**Current status (v0.7.0)**: All planned phases plus the v0.7 cleanup and feature pass complete — session list with query preview (first user message) and project/branch/fuzzy filters, session detail with tool expansion / diff rendering / turn position indicator / sidechain expansion, FTS5-indexed full-text search (with linear-scan fallback), project view, re-run (returns to list on exit), half-page scrolling (`d`/`u`) in all modes, a usage-stats panel showing token counts and estimated cost per session, **session bookmarks** with `★` markers and a bookmark-only filter, and a **timeline activity heatmap** showing 8 weeks of session activity. The render code was unified in v0.7: every mode goes through dedicated `render*Header` / `render*Footer` functions, footer hints and back-nav are consistent across all sub-views, and skipped sessions are surfaced in the list header as "(N skipped)". See `DESIGN.md` for the full vision and roadmap.

## Configuration

`lore` reads sessions from `~/.claude/projects/` by default. Override with either:

- `--dir <path>` flag (highest precedence)
- `LORE_PROJECTS_DIR` environment variable

The FTS5 search index and the bookmarks file (`bookmarks.json`) are cached under the platform-appropriate user cache dir (e.g. `~/.cache/lore/` on Linux, `~/Library/Caches/lore/` on macOS).

## Navigation

Press `?` in any mode for the full keymap. Highlights:

- **List**: `j`/`k` move, `d`/`u` half-page, `g`/`G` jump, `enter` open, `p`/`b` filter project/branch, `f` fuzzy filter, `m` bookmark, `M` bookmark-only filter, `P` project view, `/` search, `S` usage stats, `T` timeline heatmap, `q` quit.
- **Detail**: `d`/`u` half-page, `space` expand a tool turn (Agent turns with sidechains load the sub-conversation inline), `y` copy the nearest user prompt, `r` re-run that prompt, `m` bookmark this session, `/` search, `esc`/`q`/`h`/`←` back.
- **Search**: type → `enter` to run, `j`/`k` or `d`/`u` through hits, `enter` to open, `esc`/`q`/`h`/`←` back.
- **Project**: `j`/`k`, `d`/`u`, `enter` to open, `esc`/`q`/`h`/`←` back. Sessions are grouped by branch.
- **Re-run**: `enter` to spawn `claude` with the chosen prompt and CWD; `esc`/`q`/`h`/`←` to cancel.
- **Stats**: `j`/`k`, `g`/`G` to navigate; `esc`/`q`/`h`/`←` back. Columns: project · branch · model · input/output tokens · estimated cost.
- **Timeline**: `h`/`←` and `l`/`→` move the cursor across days in an 8-week activity heatmap; `enter` filters the list to the highlighted day; `esc`/`q` back.

## For contributors

See `CLAUDE.md` for:
- Development setup and test commands
- Architecture overview
- TDD requirements (red → green → refactor)
- Agent contract for future work

Full product vision and design roadmap: `DESIGN.md`
