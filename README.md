# lore

A keyboard-driven TUI for browsing Claude Code session history.

## Quick start

```bash
go build ./cmd/lore
./lore
```

The tool reads session transcripts from `~/.claude/projects/` and displays them in a sortable, navigable list.

**Current status**: Phases 1–4 complete — session list with project/branch filters, session detail with tool expansion and diff rendering, linear-scan search, project view, and re-run. Phase 5 (SQLite FTS5, list-level fuzzy match, cost stats) is still future work. See `DESIGN.md` for the full vision and roadmap.

## Navigation

Press `?` in any mode for the full keymap. Highlights:

- **List**: `j`/`k` move, `g`/`G` jump, `enter` open, `p`/`b` filter project/branch, `P` project view, `/` search, `q` quit.
- **Detail**: `space` expand a tool turn, `t` toggle thinking blocks, `y` copy the nearest user prompt, `r` re-run that prompt, `esc` back.
- **Search**: type → `enter` to run, `j`/`k` through hits, `enter` to open.
- **Project**: `j`/`k`, `enter` to open, `esc` back. Sessions are grouped by branch.
- **Re-run**: `enter` to spawn `claude` with the chosen prompt and CWD; `esc` to cancel.

## For contributors

See `CLAUDE.md` for:
- Development setup and test commands
- Architecture overview
- TDD requirements (red → green → refactor)
- Agent contract for future work

Full product vision and design roadmap: `DESIGN.md`
