# lore — Decision Narratives

## Theme 1: Conception & Naming (2026-05-01)

A tool to browse Claude Code session transcripts was proposed as a TUI. Several name candidates were considered: lore, recall, yarn, trail, scrollback. "lore" won because accumulated AI history IS your lore — short, evocative. The design was proposed to live under `grit` first then split; in practice it was given its own repo immediately.

Key commits:
- `4eb9a7a` docs: propose lore — a Claude session browser TUI
- `9b47088` lore: phase 1 — scaffold + session-list TUI (TDD)

## Theme 2: TDD Foundation & CI Gate (2026-05-01)

Before writing any features, a CI workflow with an 80% per-package coverage gate was wired. This defined the development contract: every PR must follow red→green→refactor. An initial coverage backfill was needed to clear the bar.

Key commits:
- `e7023a8` ci: add GitHub Actions workflow with 80% per-package coverage gate
- `ee841ee` fix(ci): honest 80% gate; cover Run/defaultProjectsDir without bypass
- `deaaec8` test: backfill lore package coverage to clear 80% bar

## Theme 3: Session Parsing Design (2026-05-01)

A deliberate choice was made to read ONLY the first `user` event from each JSONL file — cheap, scales to large transcripts. The `Session` struct captures ID, Path, CWD, Project, Branch, Slug, Query, and Timestamp. Later, `Query` (first user message) replaced slug as the primary session label because slug is rarely populated in practice.

Key commits:
- `9b47088` lore: phase 1 — scaffold + session-list TUI
- `9ccbadb` feat: query preview in session list, remove dead thinking toggle
- `96fe796` feat: skip system-injected XML in session query extraction

## Theme 4: Navigation & Filtering Evolution (2026-05-01 to 2026-05-06)

Filtering started with exact substring match for `p` (project) and `b` (branch) inline filters. It evolved to fuzzy ranking via `sahilm/fuzzy`, then a third cross-dimensional `f` fuzzy filter was added. The DRY pass in v0.7 collapsed three near-identical `applyFilter` branches into a single `fuzzyFilterSessions` helper.

Key commits:
- `770cf5b` test: add failing tests for project and branch filters
- `05560d8` feat(list): inline project and branch filters
- `116afac` feat: fuzzy ranking for p/b filters
- `3c08f78` test(phase5b): red failing tests for list-level fuzzy filter
- `c3129c2` feat(phase5b): implement list-level fuzzy filter with 'f' key
- `7ce545c` test(red): fuzzyFilterSessions helper for filter DRY
- `7603a9e` refactor(filter): extract fuzzyFilterSessions, DRY applyFilter

## Theme 5: Session Detail View (2026-05-01 to 2026-05-02)

The detail view was built in phases: foundation first, then polish (expand/collapse tool turns, thinking toggle, copy prompt). The thinking toggle was later removed as a dead feature (thinking content is always redacted). Diff rendering for Edit/Write tool calls was added by reusing patterns from grit.

Key commits:
- `b09f398` test: add failing tests for detail view foundation
- `542ff84` feat(detail): session detail view foundation
- `c5f6095` test: add failing tests for detail-view polish
- `a70a68d` feat(detail): expand/collapse tool turns, thinking toggle, copy prompt
- `49fe3bb` test: diff rendering for expanded Edit/Write tool turns
- `31d3c45` feat: diff rendering for expanded Edit/Write tool turns

## Theme 6: Search — Linear Scan to FTS5 (2026-05-02 to 2026-05-04)

Search was built in two passes. v1 used linear scan across JSONL files — simple and retained as a fallback. Phase 5a added a SQLite FTS5 index using `modernc.org/sqlite` (pure-Go, no CGO). The index lives in `os.UserCacheDir()` so it's platform-appropriate. Linear scan remains as a transparent fallback on FTS5 miss/error.

Key commits:
- `67bf8d8` test(search): red commit with failing tests for search v1
- `f4a9c14` feat(search): implement search v1 with linear-scan and model state transitions
- `ae71a2d` test(index): red commit - failing tests for SQLite FTS5 search index
- `09c28f9` feat(index): implement SQLite FTS5 search index
- `a17bd7c` feat(search): wire FTS5 index into search with linear-scan fallback

## Theme 7: Re-run Mode & TTY Handling (2026-05-02 to 2026-05-04)

Re-run mode was implemented to allow relaunching `claude` with a past prompt. The first implementation used `cmd.Run()` synchronously inside the bubbletea Update handler — visually broken because claude fought lore for the terminal. The fix switched to `tea.ExecProcess` which suspends the renderer and hands the TTY cleanly to the child. Later, lore was changed to return to the session list (instead of quitting) when `claude` exits.

Key commits:
- `0fe9ab6` test(rerun): add failing tests for re-run mode
- `1a19c06` feat(rerun): implement re-run mode with claude invocation
- `e4b9568` fix(rerun): use tea.ExecProcess so claude takes the TTY cleanly
- `ff5d620` test(rerun): add tests for returning to list after re-run
- `9100f9a` feat(rerun): return to session list after re-run instead of quitting

## Theme 8: Viewport & UX Polish (2026-05-02)

After dogfooding, three issues were identified: (1) no real scrolling — cursors past screen height were invisible, (2) footers advertised keys that didn't exist, (3) no feedback for no-op key presses. All three were fixed in a single cleanup PR. Text wrapping was also fixed so viewport math accurately reflected multi-line turn bodies.

Key commits:
- `12334af` test(cleanup): viewport, honest footers, no-op flash messages
- `35c7c61` feat(cleanup): viewport scrolling, honest footers, no-op flash messages
- `9a34e0b` test(detail): wrapping for multi-line turn bodies
- `4ad7aeb` fix(detail): wrap multi-line turn bodies for accurate viewport math

## Theme 9: Help Overlay (2026-05-02)

A `?` help overlay was added so users can discover keybindings without reading docs. The overlay is mode-specific — each mode shows its own keys. Later, `?` was mandated to appear in every mode footer as a universal discoverable hint.

Key commits:
- `f573617` test: ? help overlay toggle and dismissal
- `abd67d3` feat: ? help overlay
- `3bb04c0` test: expand help overlay coverage for all modes
- `e64f164` test(red): footer completeness — ? help and missing keys in all modes
- `650f147` feat(green): add ? help hint and missing keys to all mode footers

## Theme 10: Quality of Life — Phase 7 (2026-05-04 to 2026-05-05)

A batch of quality-of-life improvements: sidechain handling (sub-agent transcripts filtered from list, viewable inline), configurable projects dir (`--dir` flag + `LORE_PROJECTS_DIR` env), usage stats panel (S key), turn position indicator in detail header, consistent back-navigation (q/esc/h/← everywhere), query preview surfaced in list and project views.

Key commits:
- `3c351ef` test(red): failing tests for resolveProjectsDir, parseSessionStats, modeStats
- `607f9d2` feat(green): configurable projects dir + usage stats panel
- `90da1c1` feat(render): add turn N/M position indicator to detail view header
- `5f002a9` feat(detail): add h and left-arrow as back-navigation keys
- `669be07` test(red): failing tests for sidechain handling
- `9b07e32` feat(green): sidechain handling -- filter from list, link and render in detail

## Theme 11: v0.7 Cleanup & Chrome Unification (2026-05-05 to 2026-05-06)

A systematic cleanup pass: unified footer rendering (render*Footer functions), consistent header chrome (render*Header functions), layout constants, dead-code removal, missing unit tests, scan warning surfacing. Also fixed XML-injected system prompts polluting session query display.

Key commits:
- `8b8041b` test(red): unified footer rendering across modes
- `27983ed` feat(render): unify footer rendering across modes
- `c7d2603` test(red): consistent render*Header functions across modes
- `8bed9f7` refactor(render): consistent render*Header functions across modes
- `4503ea6` feat(scan): surface skipped-file warnings in list header
- `42fbb27` refactor(render): extract layout constants, align widths, drop dead code

## Theme 12: Session Bookmarks (2026-05-06)

A bookmark feature was added: `m` toggles a star on any session in list or detail mode, persisted to `<cacheDir>/lore/bookmarks.json`. `M` filters to bookmarks-only, composable with the fuzzy filters. Bookmarked sessions show `★` in the list.

Key commits:
- `808e298` test(red): session bookmarks
- `b2cabd0` feat(bookmarks): session bookmarks with persistent storage

## Theme 13: Timeline Activity Heatmap (2026-05-06)

A GitHub-style activity heatmap was added showing session counts over an 8-week × 7-day grid. Navigate with h/l or arrows, enter to filter the session list to a specific day. Implemented in `timeline.go` with `buildHeatmap` and `heatmapBucket` functions.

Key commits:
- `6e97bb2` test(red): timeline activity heatmap
- `ffcc4d4` feat(timeline): activity heatmap mode

## Theme 14: Public Release & Distribution (2026-05-06 to 2026-05-08)

The project was prepared for public release: GoReleaser wired for darwin/linux × amd64/arm64, a Homebrew tap at `zpenka/homebrew-lore`, version promoted from `const` to `var` so ldflags can inject it. README was rewritten for public audience. CI Go version bumped to 1.25 for `modernc.org/sqlite` compatibility.

Key commits:
- `f1dfcd2` fix(ci): bump Go to 1.25 for modernc.org/sqlite compatibility
- `5d67877` release: wire GoReleaser, Homebrew tap, and v0.7.0 version
- `59ccc4c` docs: rewrite README for public release
