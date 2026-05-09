# lore — v0.8 playbook

A sequenced, agent-friendly task list to drive lore from v0.7 to v0.8 in a
single synchronous Sonnet run. Each task is one PR, scoped tight enough that
it should fit in one focused work block, with red/green/refactor commits per
the agent contract in `CLAUDE.md`.

> **Read this before starting.** The repo has hard rules in `CLAUDE.md`
> ("Test-driven development (required)" and "Agent contract"). Every task
> below assumes you'll follow them: worktree off `main`, red commit first
> with a failing test, green commit with the minimum code to pass, optional
> refactor. CI gates on `go test -race -cover ./...` with ≥80% per-package
> coverage. Don't skip hooks. Don't add deps not listed in `DESIGN.md`.

The tasks are ordered so that earlier work makes later work cheaper. Land
them in order. If a task can't be completed cleanly, stop and surface the
blocker — don't escalate scope.

---

## Operating rules for the executing agent

1. **One task = one PR.** Do not bundle. Open the next PR only after the
   previous one is merged (or, if you're driving the merges, after CI is
   green and the diff is clean).
2. **Red, then green, then optional refactor.** Each phase is its own
   commit with a clear message. The red commit's test must fail when run
   in isolation against the unmodified production code.
3. **No new dependencies.** Anything outside the four listed in `DESIGN.md`
   (`bubbletea`, `lipgloss`, `sahilm/fuzzy`, `modernc.org/sqlite`) is a
   blocker — stop and ask.
4. **Behavior preservation for refactors.** Tasks marked *(refactor)* must
   not change a single key binding, footer string, or rendered byte. The
   existing test suite is your contract.
5. **Update docs as you go.** If you change a key binding or add a flag,
   update `CLAUDE.md` (Keyboard Navigation / Configuration sections) and
   the `?` overlay in `render.go` and the README in the same PR. The
   `footer_completeness_test.go` and `help_test.go` suites enforce some of
   this for you.
6. **PR body format** (per `CLAUDE.md` agent contract):
   `Red commit: <sha>, green commit: <sha>, refactor: <sha or "none">`.

---

## Task 1 — Split `model.go` into per-mode key handlers *(refactor)*

**Goal.** Move the seven `handle*Key` functions out of `model.go` into one
file per mode. `model.go` keeps the `model` struct, `Init`, `Update`, the
top-level `handleKey` dispatch, the offset-clamp helpers, and `applyFilter`.

**Why.** `model.go` is 962 lines. Subsequent feature tasks will touch
specific handlers and the diffs will be much cleaner with one handler per
file. Pure mechanical move; no logic change.

**Files to add.**
- `keys_list.go`     — `handleListKey`, `handleFilterEntryKey`
- `keys_detail.go`   — `handleDetailKey`
- `keys_search.go`   — `handleSearchKey`, `handleSearchEntryKey`, `handleSearchResultsKey`
- `keys_project.go`  — `handleProjectKey`
- `keys_rerun.go`    — `handleRerunKey`
- `keys_stats.go`    — `handleStatsKey`, `computeStatsRows`
- `keys_timeline.go` — `handleTimelineKey`

**Files to shrink.**
- `model.go` — delete what moved, keep dispatch and shared state.

**Red.** Add `internal_split_test.go` with a single test that imports the
package and asserts (via `runtime` reflection or just call-site coverage)
that each of the moved functions is still callable through the model
dispatch — i.e. `model{mode: modeStats}.Update(keyMsg("j"))` returns
without panic for each mode. This will fail to compile *only* if the move
breaks the public surface; design it as a regression net.

**Green.** Move the functions. Run the full suite — every existing test
must pass unchanged.

**Acceptance.**
- `model.go` is under ~400 lines.
- `go test -race -cover ./...` passes; per-package coverage stays ≥80%.
- `gofmt -l .` and `go vet ./...` are clean.

**Out of scope.** Do not rename anything. Do not change function
signatures. Do not "while you're there" tweak any handler's behavior.

---

## Task 2 — Split `render.go` into per-mode renderers *(refactor)*

**Goal.** Same treatment as Task 1, for `render.go` (currently 1029 lines).
Move per-mode `render*View` / `render*Header` / `render*Footer` /
`*BodyLines` triples into one file per mode. `render.go` keeps `View()`
dispatch, the Lipgloss styles, the layout constants, the help overlay, and
the shared body/clamp helpers (`renderBody`, `renderDivider`, `bodyHeight`).

**Files to add.**
- `render_list.go`
- `render_detail.go`
- `render_search.go`
- `render_project.go`   *(merge with the existing `project.go` rendering helpers if it makes sense)*
- `render_rerun.go`
- `render_stats.go`
- `render_timeline.go`

**Red / Green / Acceptance.** Same shape as Task 1. The existing
`render_test.go`, `render_constants_test.go`, `footer_completeness_test.go`
form your contract — every assertion must pass byte-for-byte.

**Out of scope.** No style tweaks. No header/footer wording changes. No
new helpers. If you find a temptation to refactor a body-lines function,
file it as a follow-up and move on.

---

## Task 3 — Extract a shared cursor-nav helper *(refactor)*

**Goal.** Replace the duplicated `j/k/d/u/g/G` blocks across the five
list-shaped handlers (`list`, `detail`, `search`, `project`, `stats`) with
a single helper:

```go
// nav advances cursor by the standard list keys ("j","k","d","u","g","G",
// "down","up"). Returns the new cursor. count is len(items); halfPage is
// the half-page step (always >= 1). For unknown keys returns cursor
// unchanged.
func nav(key string, cursor, count, halfPage int) int
```

Each handler then collapses its six cases to:

```go
case "j", "k", "d", "u", "g", "G", "down", "up":
    m.cursor = nav(msg.String(), m.cursor, len(m.visibleSessions), halfPage(m))
    m = m.clampListOffsetNow()
```

**Files.**
- `nav.go` — new, with `nav` and `halfPage(m model) int`.
- `nav_test.go` — exhaustive table test for `nav`.
- `keys_*.go` — collapse the duplicated blocks (after Task 1).

**Why.** ~150 lines of repeated code disappear. Future modes (none planned,
but cheap insurance) get correct cursor behavior for free.

**Red.** Write `nav_test.go` first with a table covering: empty list, one
item, mid-list `j`, top `k`, bottom `G`, half-page `d` overshooting end,
half-page `u` underflowing start, unknown key. All tests fail because
`nav` doesn't exist.

**Green.** Implement `nav`, then collapse handlers one mode at a time.
Run the existing per-mode tests after each collapse to catch regressions.

**Acceptance.**
- `nav.go` < 80 lines. `nav_test.go` covers ≥95% of `nav.go`.
- Total LOC across `keys_*.go` files drops by ≥150.
- All existing model tests pass unchanged.

**Out of scope.** Do not touch the timeline handler (different shape:
left/right, date math). Do not change behavior of any existing key.

---

## Task 4 — DRY the lazy FTS5-index open *(refactor)*

**Goal.** Move the six-line block in `handleSearchEntryKey` that opens the
index on first search into a method:

```go
// ensureIndex opens the FTS5 index on first use and runs an initial Sync.
// Best-effort: returns the model unchanged if the index can't be opened.
func (m model) ensureIndex() model
```

The search handler becomes `m = m.ensureIndex()` followed by the existing
search dispatch. Sets up Task 8 (background sync).

**Files.**
- `model.go` (or `index.go`) — add `ensureIndex`.
- `keys_search.go` — call site collapses.
- `model_test.go` (or `index_test.go`) — direct test of `ensureIndex` with
  an injected fake `OpenIndex` if feasible; otherwise an integration test
  via the search handler.

**Red.** Test that calling `ensureIndex` twice opens the index exactly
once (use a counter via package-level swap, or skip and rely on the
existing search test if injection is awkward — flag the trade-off in the
PR).

**Acceptance.** Behavior unchanged. `keys_search.go` shrinks. No new
public API outside the package.

**Out of scope.** Do not move the sync call to a `tea.Cmd` yet — that's
Task 8. Keep the change purely structural here.

---

## Task 5 — `LORE_CACHE_DIR` env var *(small feature)*

**Goal.** Honor `LORE_CACHE_DIR` as an override for the cache location used
by the FTS5 index (`index.db`) and bookmarks (`bookmarks.json`). Resolution
order, mirroring `resolveProjectsDir`:

1. `LORE_CACHE_DIR` env var (if set and non-empty)
2. `os.UserCacheDir()` + `/lore`

**Files.**
- `lore.go` — add `resolveCacheDir() (string, error)`.
- `index.go` — `indexCacheDir` calls the new resolver.
- `bookmark.go` — `bookmarksFile` calls the new resolver.
- `lore_test.go` — env-var precedence test (set/unset, empty string,
  non-existent dir creation).
- `CLAUDE.md` and `README.md` — Configuration section gains a row for the
  new env var.

**Red.** Test `resolveCacheDir` with the env var set, unset, and empty.
Set expectations against the resolved path; create the dir if missing.

**Acceptance.** Existing index/bookmark tests still pass. New env-var
behavior is covered.

**Out of scope.** Don't add a `--cache-dir` flag yet. Env var is enough
for v0.8; add the flag if a user asks.

---

## Task 6 — Resume a session with `R` *(feature)*

**Goal.** Add a new key `R` (capital R) to list mode and detail mode that
resumes the selected session via `claude -c <session-id>`. Mirrors the
existing `r` (re-run a single prompt) plumbing.

**Why.** `r` is "re-run this prompt as a new session"; `R` is "continue
this session." Both verbs are first-class daily-driver actions.

**Files.**
- `rerun.go` — add a sibling `resumeClaude(sessionID, cwd string) tea.Cmd`
  that wraps `tea.ExecProcess` for `claude -c <id>` (verify the actual
  Claude CLI flag — `-c`, `--continue`, `--resume`; it's `--resume <id>`
  on recent versions; check what's installed and document the assumption
  in the PR body).
- `model.go` — new injectable hook `resumeFn func(id, cwd string) tea.Cmd`
  alongside `rerunFn`, defaulted to `resumeClaude`.
- `keys_list.go` and `keys_detail.go` — handle `R`.
- `render.go` — help overlay gets a new line (`R resume session`); list and
  detail footers get the hint.
- `model_test.go` — test that pressing `R` invokes `resumeFn` with the
  selected session's ID and CWD; uses a fake `resumeFn` that records the
  call.
- `CLAUDE.md` and `README.md` — Keyboard Navigation update.

**Red.** Write the model test with a fake `resumeFn`, asserting it gets
called with the right args. Run — test fails because `R` is unbound.

**Green.** Wire `R` in both handlers and the help/footer surfaces.

**Acceptance.**
- `footer_completeness_test.go` and `help_test.go` still pass (they're
  table-driven on the help map; update the table in the same PR).
- Manual smoke: open lore, hit `R` on a session, verify Claude attaches
  to the existing transcript (not a fresh session). If the CLI flag is
  wrong, surface that as a blocker in the PR description; don't ship.

**Out of scope.** Don't add a "edit prompt before resume" flow. Don't add
worktree switching from the design doc — that's a future task.

---

## Task 7 — Pricing table → embedded JSON *(refactor + small feature)*

**Goal.** Replace the hardcoded `pricingTable` slice in `stats.go` with an
embedded JSON file (`pricing.json`) loaded via `go:embed`. Add a
`LORE_PRICING_FILE` env var override so users on enterprise rates don't
need a fork.

**Files.**
- `pricing.json` — new, contains the same data as the current `pricingTable`,
  schema:
  ```json
  [
    {"substr": "opus",   "input_per_mtok": 15.0, "output_per_mtok": 75.0, "cache_read_fraction": 0.1},
    {"substr": "sonnet", "input_per_mtok":  3.0, "output_per_mtok": 15.0, "cache_read_fraction": 0.1},
    {"substr": "haiku",  "input_per_mtok":  0.8, "output_per_mtok":  4.0, "cache_read_fraction": 0.1}
  ]
  ```
- `stats.go` — embed via `//go:embed pricing.json`; load on first use;
  honor `LORE_PRICING_FILE` env var override (file path; falls back to
  embedded if missing or malformed, surface a one-time warning via the
  list header).
- `stats_test.go` — test override path: write a temp pricing file, set the
  env var, assert `estimateCost` uses the new rate.
- `CLAUDE.md` and `README.md` — Configuration section gets the new env var.

**Red.** Write the override test; fails because `LORE_PRICING_FILE` is
unread.

**Green.** Implement the loader. Cache the parsed table in a package-level
`sync.Once`-guarded var so we don't re-read on every cost calc.

**Acceptance.** All existing `stats_test.go` cases still pass — same
defaults baked in.

**Out of scope.** Don't add a UI for picking a model. Don't expand the
table to non-Anthropic models.

---

## Task 8 — Background FTS5 sync at startup *(perf)*

**Goal.** Open the FTS5 index and run `Sync` in a `tea.Cmd` dispatched
from `Init()`, instead of lazily on first search. First search becomes
instant for repeat users; first-time users still see the existing
behavior.

**Why.** Today the first `enter` after typing a search query stalls for
hundreds of ms while the index syncs. After Task 4 the open path is
already factored.

**Files.**
- `model.go` — `Init()` returns a batched cmd: `tea.Batch(loadSessionsCmd,
  syncIndexCmd)`. New `indexReadyMsg{idx *Index, err error}` type.
- `keys_search.go` — `ensureIndex` becomes a no-op if the index is
  already set; otherwise falls back to the on-demand path.
- `render_list.go` — list header shows `indexing…` while `m.indexing` is
  true (a new field set when the cmd is in flight, cleared on
  `indexReadyMsg`).
- `model_test.go` — test that `Init` produces both messages; test that
  `indexReadyMsg` populates `m.index`; test header shows `indexing…`
  only when the flag is set.

**Red.** Write the message-flow tests first.

**Green.** Implement.

**Acceptance.** Existing search tests pass without modification (the
fallback path still works when the background sync hasn't completed).
Header shows `indexing…` only during the first sync, not on every key
press.

**Out of scope.** No periodic re-sync. No "watch for new files" — the
mtime-based sync on next startup catches everything. No progress
percentage; the boolean flag is enough.

---

## Task 9 — Search query syntax for `project:` and `branch:` *(feature)*

**Goal.** Parse search queries like `project:lore branch:main refresh
token` into structured filters. The non-prefixed terms hit FTS5; the
prefixed terms post-filter the result set against `Session.Project` and
`Session.Branch`.

**Why.** Two daily-driver searches today require multiple keystrokes
(filter then search, or search then visually scan). One query string nails
both.

**Files.**
- `search.go` — new `parseSearchQuery(q string) (text string, filters
  searchFilters)` where `searchFilters` is a small struct
  (`project`, `branch` strings; empty == no filter). Linear-scan
  `searchSessions` honors the filters.
- `index.go` — `Index.Search` takes the parsed filters and post-filters
  by joining `session_path` against the parsed `Session.Project` /
  `Session.Branch`. (Cheaper than a schema change for v0.8; revisit if
  search latency suffers.)
- `keys_search.go` — call `parseSearchQuery` before dispatching to FTS5
  vs. linear scan.
- `search_test.go` — table tests for the parser (single prefix, multiple
  prefixes, prefix at end, prefix with quoted value, no prefix); end-to-end
  test that `project:lore foo` returns only lore sessions matching `foo`.
- `render.go` — help overlay gains a one-line note about the syntax.
  README mention too.

**Red.** Parser table tests, then end-to-end test.

**Green.** Implement parser; thread filters through both search paths.

**Acceptance.** All existing search tests still pass (prefix-free queries
behave identically).

**Out of scope.** No `cwd:` or `since:` filters yet — propose them as
v0.9 if users ask. No quoted-string parsing beyond the trivial case (do
the simple thing; flag if a user hits the limit).

---

## Task 10 — Fuzz the JSONL parsers *(quality)*

**Goal.** Add `go test -fuzz` targets for `parseSessionMetadata` and
`parseTurnsFromJSONL`. They consume third-party data (transcripts written
by Claude Code) and have unsafe-looking type assertions in a few places.

**Files.**
- `session_test.go` — `FuzzParseSessionMetadata` seeded with a few real
  fixtures.
- `detail_test.go` — `FuzzParseTurnsFromJSONL` seeded similarly.
- `.github/workflows/ci.yml` — add a step that runs each fuzz target for
  30 seconds (`go test -fuzz=FuzzParseSessionMetadata -fuzztime=30s
  -run=^$`). Keep it as a separate non-blocking job initially so a flaky
  fuzz crash doesn't gate merges; promote to required once it's been
  green for a week.

**Red.** Add the fuzz targets; CI runs them. Any panic surfaces as a
failure. (If they pass immediately, that's fine — the value here is
ongoing protection, not a one-time catch.)

**Green.** Fix anything the fuzzer finds. If nothing, the PR is just the
test additions and the CI step.

**Acceptance.** CI runs fuzz for 30s × 2 targets per push. No regressions
in unit-test coverage.

**Out of scope.** Don't fuzz `extractSessionText` or `parseSessionStats`
in the same PR; if the JSONL parser fuzzers find nothing in a week, add
those next.

---

## Done criteria for v0.8

When all ten tasks are merged:

- `model.go` and `render.go` are each under ~400 lines.
- The `nav` helper backs every list-shaped mode.
- `LORE_CACHE_DIR` and `LORE_PRICING_FILE` join `LORE_PROJECTS_DIR` as
  first-class env-var overrides.
- `R` resumes sessions; the search bar accepts `project:` and `branch:`
  prefixes.
- The FTS5 index syncs in the background at startup; the list header
  shows `indexing…` while it does.
- `pricing.json` is the single source of truth for cost rates.
- The JSONL parsers are protected by ongoing fuzz coverage.

Tag `v0.8.0`, update `CLAUDE.md`'s "Project Overview" and `DESIGN.md`'s
phasing table, and ship.

---

## Explicitly deferred to v0.9 (do not start here)

These came up while planning but are out of scope for the playbook above —
they each require product judgment that's better made after v0.8 ships:

- Per-session annotations / notes (needs UX design).
- Search-result hit jumping into the matching turn (FTS5 schema migration).
- Cost aggregation header and group-by-project in stats panel.
- Cost-shaded timeline heatmap.
- Side-by-side session compare.
- Streaming session scan with incremental render.
- Stats result caching in SQLite.
- Shell completions, man page, `--json` machine output, README GIF.

Surface user demand for these in GitHub issues first; pick the next
playbook from there.
