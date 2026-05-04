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

**Current status (v0.4.0)**: Phases 1–4 complete — session list with project/branch filters, session detail with tool expansion and diff rendering, linear-scan search, project view, and re-run. Next up: FTS5 indexed search (5a), list-level fuzzy matching (5b), cost/usage stats (5c), and quality-of-life improvements (7). See `DESIGN.md` for the full vision and roadmap.

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
