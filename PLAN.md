# lore v0.7 — cleanup & new features plan

> **Created:** 2025-05-05
> **Status:** Draft — not yet started
> **Goal:** Clean up the codebase for consistency, then add two new features.
> Designed for parallel subagent execution where possible.

---

## Part 1: Codebase Cleanup

Six independent cleanup tasks. Each can be tackled by a separate subagent in
its own worktree branch, then merged sequentially.

### 1A — Unify footer rendering

**Problem:** Five modes render footers in four different ways. Search and
rerun build footers inline in their view functions; list, detail, project,
and stats each have their own footer function. The hint text is inconsistent
("back" vs "quit", "q/esc/h/←" vs "q/esc", missing `d/u` in stats).

**Work:**
- Extract a single `renderModeFooter(mode, flash, hints)` helper or
  at minimum ensure every mode uses a dedicated `render*Footer()` function.
- Standardize hint format: `key action` pairs separated by three spaces.
- All sub-views (detail, search results, project, rerun, stats) should
  show `q/esc/h/← back`. List shows `q quit`.
- Add `d/u page` to stats footer (stats supports `d`/`u` scrolling but
  the footer doesn't mention it).
- Flash message display should go through one path, not four.

**Files:** `render.go`, `project.go`
**Tests:** `render_test.go` — add/update footer content assertions.

---

### 1B — DRY the filter logic in `applyFilter`

**Problem:** `model.go::applyFilter()` has three near-identical branches
for project / branch / fuzzy filtering — ~60 lines of duplicated
build-list → fuzzy-rank → map-back logic.

**Work:**
- Extract a generic helper: `fuzzyFilterSessions(text string, candidates func(Session) string, sessions []Session) []Session`.
- Reduce `applyFilter` to three calls to that helper.

**Files:** `model.go`, `filter.go`
**Tests:** `filter_test.go` — existing tests should still pass; add a
unit test for the extracted helper.

---

### 1C — Remove dead code and extract magic numbers

**Problem:**
- `renderRows()` in `render.go` is marked as "retained for tests" but no
  test calls it.
- Hard-coded column widths (48, 12, 20, 26) and line limits (5, 80) are
  scattered across render functions with no named constants.

**Work:**
- Delete `renderRows()` if grep confirms zero callers.
- Extract constants: `fixedCols`, `projectColWidth`, `branchColWidth`,
  `snippetMaxLen`, `rerunMaxLines`, etc.
- Align search result row widths to match list row widths (branch column
  is 26 in search vs 20 in list — pick one).

**Files:** `render.go`
**Tests:** Existing render tests; update any that assert exact row strings.

---

### 1D — Add missing unit tests for untested utilities

**Problem:** Several core functions have zero direct test coverage:
- `extractQuery()`, `stripSystemTags()`, `collapseWhitespace()` — session
  metadata extraction used everywhere.
- `groupByBranch()` — project view's data layer.
- `formatTokenCount()`, `estimateCost()` — stats display.
- `clampOffset()`, `sliceLines()` — viewport primitives.

**Work:**
- Add `viewport_test.go` for `clampOffset` / `sliceLines`.
- Add `stats_test.go` tests for `formatTokenCount` / `estimateCost`.
- Add tests for `extractQuery`, `stripSystemTags`, `collapseWhitespace`
  (in `session_test.go` or a new file).
- Add `project_test.go` for `groupByBranch`.

**Files:** New and existing `*_test.go` files.

---

### 1E — Consistent header chrome

**Problem:** Headers vary across modes — some show session counts, some
show turn position, some show project name. The inconsistency is fine
where it's intentional (each mode has different context), but the
*structure* should be uniform: left-aligned title, right-aligned context,
same style.

**Work:**
- Audit each header: ensure all use `headerStyle`, ensure the divider
  immediately follows, and ensure no mode builds the header inline in its
  view function (rerun currently does).
- Extract `renderRerunHeader()` to match the pattern of the other modes.
- Ensure all headers render at exactly one line height (no wrapping).

**Files:** `render.go`
**Tests:** `render_test.go` — assert header structure for rerun mode.

---

### 1F — Normalize error handling in scan/parse paths

**Problem:** `scanSessions` silently skips unreadable files,
`searchSession` returns nil on open failure, `computeStatsRows` produces
zero-stats rows on error. No visibility into what was lost.

**Work:**
- Add a `warnings []string` field to the model.
- When `scanSessions` skips a file, append to warnings.
- Show warning count in the list footer (e.g. "142 sessions (3 skipped)").
- Don't add logging — keep it TUI-native.

**Files:** `model.go`, `session.go`, `render.go`
**Tests:** `session_test.go` — test with an unreadable fixture file.

---

## Part 2: New Features

Two new features that add genuine value for the target user (5–20 Claude
sessions/day across multiple repos).

### 2A — Session bookmarks

**What:** Let users mark sessions as important so they can find them later
without searching. A bookmarked session gets a `★` indicator in the list
and a dedicated filter to show only bookmarks.

**Why:** Power users accumulate hundreds of sessions. The current tools
(search, project filter, branch filter) are great for "I know roughly
what I'm looking for" but bad for "I want to keep track of the 5 sessions
that represent key decisions." Bookmarks solve the latter.

**Design:**
- Storage: a JSON file at `<cacheDir>/lore/bookmarks.json` — a simple
  `map[string]bool` keyed by session ID. Read-only access to Claude Code's
  files is preserved; bookmarks are lore's own data.
- Keys: `m` (mark) in list and detail modes to toggle a bookmark on the
  current session. Flash message confirms "bookmarked" / "unmarked".
- Filter: `M` in list mode shows only bookmarked sessions (similar UX to
  the `p`/`b` filters but binary).
- Display: bookmarked sessions show `★` before the time column in list,
  search, and project views.

**Files (new):** `bookmark.go`, `bookmark_test.go`
**Files (modified):** `model.go`, `render.go`, `project.go`

**Subagent plan:**
1. Red: tests for `loadBookmarks`, `saveBookmark`, `toggleBookmark`,
   and model-level `m` / `M` key handling.
2. Green: implement `bookmark.go` + model integration.
3. Refactor: extract any shared filter logic.

---

### 2B — Session timeline / activity heatmap

**What:** A new mode (`T` from list) that shows a calendar-style heatmap
of session activity — how many sessions per day over the last 8 weeks,
rendered as a grid of intensity-shaded blocks.

**Why:** Understanding your Claude usage patterns over time is useful for
productivity awareness ("am I over-relying on Claude for certain tasks?")
and for navigating to a specific day's work. Today there's no way to
answer "what did I work on last Tuesday?" without scrolling through the
time-bucketed list.

**Design:**
- Layout: 7 rows (Mon–Sun) × 8 columns (weeks), newest week on the
  right. Each cell is a two-character block (e.g. `██`) colored by
  session count: 0 = dim, 1–2 = light, 3–5 = medium, 6+ = bright.
  Below the grid: a legend row showing the intensity scale.
- Navigation: `h`/`l` or `←`/`→` to move the highlight across days.
  The footer shows the date and session count for the highlighted day.
  `enter` on a day filters the list to that date and returns to list
  mode.
- Data: reuse the already-parsed `sessions` slice — just bucket by
  `time.Time.Truncate(24h)` into a `map[string]int` of date → count.

**Files (new):** `timeline.go`, `timeline_test.go`
**Files (modified):** `model.go` (new mode + `T` key), `detail.go`
(add `modeTimeline` constant), `render.go` (add `renderTimelineView`),
help overlay.

**Subagent plan:**
1. Red: tests for `buildHeatmapData`, `heatmapColor`, timeline
   navigation keys, and enter-to-filter.
2. Green: implement `timeline.go` + model integration + rendering.
3. Refactor: extract date-bucketing if it overlaps with `timeBucket`.

---

## Execution Strategy

These tasks are designed for parallel subagent execution:

```
Parallel batch 1 (cleanup — all independent):
  ├── Agent 1A: footer unification       (worktree: cleanup/footer)
  ├── Agent 1B: DRY filter logic          (worktree: cleanup/filter)
  ├── Agent 1C: dead code + constants     (worktree: cleanup/constants)
  ├── Agent 1D: missing unit tests        (worktree: cleanup/tests)
  ├── Agent 1E: header chrome             (worktree: cleanup/header)
  └── Agent 1F: error handling            (worktree: cleanup/errors)

Sequential merge: 1A → 1B → 1C → 1D → 1E → 1F into main

Parallel batch 2 (features — independent of each other, depend on cleanup):
  ├── Agent 2A: bookmarks                 (worktree: feat/bookmarks)
  └── Agent 2B: timeline heatmap          (worktree: feat/timeline)

Sequential merge: 2A → 2B into main
```

Each agent follows the repo's red → green → refactor contract and opens
a PR. Coverage gate (≥80%) must pass before merge.
