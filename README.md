# lore

A keyboard-driven TUI for browsing Claude Code session history.

## Quick start

```bash
go build ./cmd/lore
./lore
```

The tool reads session transcripts from `~/.claude/projects/` and displays them in a sortable, navigable list.

**Current status**: Phase 1 (session list view, keyboard navigation). See `DESIGN.md` for the full vision and roadmap.

## Navigation

- `j`/`k`: Move cursor up/down
- `g`/`G`: Jump to top/bottom
- `q`: Quit

## For contributors

See `CLAUDE.md` for:
- Development setup and test commands
- Architecture overview
- TDD requirements (red → green → refactor)
- Agent contract for future work

Full product vision and design roadmap: `DESIGN.md`
