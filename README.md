# lore

A keyboard-driven TUI for browsing Claude Code session history.

## Quick start

```bash
# Install
go install github.com/zpenka/lore/cmd/lore@latest

# Or build from source
go build ./cmd/lore
./lore
```

The tool reads session transcripts from `~/.claude/projects/` and displays them in a sortable, navigable list.

**Current status (v0.6.0)**: All planned phases complete — session list with project/branch/fuzzy filters, session detail with tool expansion / diff rendering / turn position indicator / sidechain expansion, FTS5-indexed full-text search (with linear-scan fallback), project view, re-run (returns to list on exit), and a usage-stats panel showing token counts and estimated cost per session. See `DESIGN.md` for the full vision and roadmap.

## Configuration

`lore` reads sessions from `~/.claude/projects/` by default. Override with either:

- `--dir <path>` flag (highest precedence)
- `LORE_PROJECTS_DIR` environment variable

The FTS5 search index is cached under the platform-appropriate user cache dir (e.g. `~/.cache/lore/index.db` on Linux).

## Navigation

Press `?` in any mode for the full keymap. Highlights:

- **List**: `j`/`k` move, `g`/`G` jump, `enter` open, `p`/`b` filter project/branch, `f` fuzzy filter, `P` project view, `/` search, `S` usage stats, `q` quit.
- **Detail**: `space` expand a tool turn (Agent turns with sidechains load the sub-conversation inline), `t` toggle thinking blocks, `y` copy the nearest user prompt, `r` re-run that prompt, `esc`/`h`/`←` back.
- **Search**: type → `enter` to run, `j`/`k` through hits, `enter` to open.
- **Project**: `j`/`k`, `enter` to open, `esc` back. Sessions are grouped by branch.
- **Re-run**: `enter` to spawn `claude` with the chosen prompt and CWD; `esc` to cancel.
- **Stats**: `j`/`k`, `g`/`G` to navigate; `esc`/`q` back. Columns: project · branch · model · input/output tokens · estimated cost.

## For contributors

See `CLAUDE.md` for:
- Development setup and test commands
- Architecture overview
- TDD requirements (red → green → refactor)
- Agent contract for future work

Full product vision and design roadmap: `DESIGN.md`
