# lore — design sketch

A keyboard-driven TUI for browsing your Claude Code session history.

> **Status:** v0.4.0 — Phases 1–4 implemented (list, detail, search v1, project
> view, re-run). The repo split (Phase 6) has happened — this is the standalone
> `github.com/zpenka/lore` module. Next up: Phases 5a–5c (FTS5 index, list-level
> fuzzy match, cost/usage stats) and Phase 7 (quality-of-life).
> See [Phasing](#phasing) for status.
>
> Name landed on `lore`.

---

## Why this exists

If you drive Claude Code all day, you accumulate a transcript of work that is
genuinely valuable — every prompt you've written, every plan you've approved,
every diff Claude has produced. Claude Code already writes a rich JSONL
transcript per session under `~/.claude/projects/<encoded-cwd>/<sid>.jsonl`,
but there is no good way to browse it.

Today, finding past work means one of:

- `grep` over JSONL files — works, but flat, lossy, breaks on multiline content
- scrolling the web UI one session at a time — slow, no cross-session search
- remembering which branch you were on and hoping `git reflog` is enough — it isn't

Common pain points the tool should kill:

- *"What was the prompt that produced that nice refactor last week?"*
- *"Which session touched `auth.go`?"*
- *"What plan did I approve in the dotfiles repo three days ago?"*
- *"I want to re-run that prompt from yesterday but in a different worktree."*
- *"Which skills/MCP servers were active when I wrote that?"*

Audience: an engineer running 5–20 Claude sessions/day across multiple repos.
Goal: do for your Claude history what `grit` does for git history.

---

## Name candidates (historical)

The name `lore` was picked from this short list:

| Name | Why |
|---|---|
| **lore** ← chosen | Your accumulated AI work history is your lore. Short, evocative. |
| **recall** | Action verb — exactly what the tool does. |
| **yarn** | A thread of conversation. Short, tactile. |
| **trail** | Breadcrumbs through past sessions. |
| **scrollback** | Terminal jargon, immediately legible to the target user. |

---

## Repo layout

`lore` lives in its own repo at `github.com/zpenka/lore`. The split happened
early (Phase 6 in the original phasing). For an up-to-date file map of the
current package, see `CLAUDE.md` → "Repo Layout".

The original plan was to nest `lore` under `grit` and split with
`git filter-repo --subdirectory-filter lore` once the design stabilized. In
practice, the standalone repo was created up-front and the only carryover from
grit was the diff-rendering / clipboard / fixture *patterns* — no shared
module code (see [Tech / reuse from grit](#tech--reuse-from-grit)).

---

## Data sources (already on disk)

Everything `lore` needs is already written by Claude Code. Read-only.

| Path | What it is |
|---|---|
| `~/.claude/projects/<encoded-cwd>/<sid>.jsonl` | One file per session. One JSON event per line. |
| `~/.claude/sessions/<n>.json` | Session env / metadata index. |
| `~/.claude/settings.json` | Global settings (model, theme, hooks). |
| `~/.claude/skills/` | User-level skills available to all projects. |

The `<encoded-cwd>` segment encodes the working directory by replacing `/` with
`-`, e.g. `/home/user/grit` → `-home-user-grit`. Decode for display.

### Event types observed in transcripts

Sampled directly from real session files in `~/.claude/projects/-home-user-grit/`:

`user`, `assistant`, `tool_use`, `tool_result`, `text`, `thinking`,
`queue-operation`, `attachment`, `plan_mode`, `tools_changed`,
`skill_listing`, `messages_changed`, `tool_reference`,
`deferred_tools_delta`, `direct`, `message`.

Each `user` event carries the full context needed for list-views:

```jsonc
{
  "type": "user",
  "sessionId": "c95ad6fe-7304-45e6-a726-80780e34099c",
  "timestamp": "2026-05-01T22:37:40.568Z",
  "cwd": "/home/user/grit",
  "gitBranch": "claude/plan-new-cli-tool-8Faqf",
  "slug": "hey-so-i-want-twinkly-aho",
  "permissionMode": "plan",
  "entrypoint": "remote_mobile",
  "version": "2.1.126",
  "parentUuid": null
}
```

The `slug` is gold — a free human-readable label the tool can use as the
session title in any list view.

---

## Feature sketches

Five panels. Each shows a one-line summary, an ASCII mockup, the keys, and
what data backs it.

### 3.1 Session list — the home panel

Most-recent sessions across every project, grouped by relative time.

```
 lore · 142 sessions across 7 projects                   [/] search  [?] help
─────────────────────────────────────────────────────────────────────────────
 today
►  14:22  grit          claude/plan-new-cli-tool   hey-so-i-want-twinkly-aho
   11:08  dotfiles      main                       fix-zsh-prompt-on-login
   09:41  api-server    feat/auth-rotation         add-jwt-refresh-flow
 yesterday
   22:37  grit          claude/auth-fix            why-does-blame-skip-merges
   17:55  api-server    main                       weekly-deps-update
 last week
   ...
─────────────────────────────────────────────────────────────────────────────
 j/k move   enter open   p project filter   b branch filter   r re-run
```

- Sort: most recent first, grouped by relative time bucket.
- Each row: time, project (decoded from path), git branch, slug.
- Cheap implementation: `os.Stat` mtimes for ordering, then read just the
  first `user` event of each `.jsonl` to populate the row.
- `p` opens an inline filter scoped to a single project; `b` to a single branch.

### 3.2 Session detail — replay one transcript

Open a session, walk it turn by turn. Tool calls collapse to one line; expand
on demand.

```
 hey-so-i-want-twinkly-aho · grit · claude/plan-new-cli-tool   2026-05-01
─────────────────────────────────────────────────────────────────────────────
  user │ hey, so i want to make another cli tool that is helpful for...
       │
asst   │ I'll start by exploring grit's structure to understand patterns.
       │ ▸ Agent (Explore) "Explore grit's structure for reuse"     [120 lines]
       │ ▸ Bash "ls -la ~/.claude/"                                  [12 lines]
       │ ▸ Read ~/.claude/projects/.../c95ad6fe.jsonl                [3 lines]
       │
  user │ Got it — Claude session browser, plan doc only...
       │
asst   │ ▸ Write /root/.claude/plans/hey-so-i-want-twinkly-aho.md
       │ Writing the plan now.
─────────────────────────────────────────────────────────────────────────────
 j/k scroll   space expand tool call   t toggle thinking   y copy turn
```

- Tool calls collapsed to one line by default (`▸ Tool "args" [N lines]`).
  `space` expands.
- `thinking` blocks hidden by default (`t` to toggle).
- `y` yanks the selected turn's user prompt to the clipboard — instant
  re-prompt material.
- For `Edit` / `Write` tool calls, the expansion renders the diff using
  grit's `diffLine` styling (added/removed/context/hunk colors).

### 3.3 Search — full-text across all sessions

```
 search: refresh token rotat_                               142 sessions
─────────────────────────────────────────────────────────────────────────────
 api-server  feat/auth-rotation     add-jwt-refresh-flow       3 hits
   "...rotate the refresh token on every use, but keep the access..."
 api-server  main                   weekly-deps-update          1 hit
   "...refresh-token library v3.2 has a CVE..."
 grit        claude/auth-fix        why-does-blame-skip-merges  1 hit
   "...the blame walker doesn't refresh after a rebase..."
─────────────────────────────────────────────────────────────────────────────
 enter open at hit   / new search   p filter project   esc back
```

- v1: linear scan of JSONL files, only `user` and `text`/`assistant` content.
  Plenty fast for a few thousand sessions.
- v2 (later phase): SQLite FTS5 index in `~/.claude/lore/cache.db`, refreshed
  on session-file mtime change.

### 3.4 Project view — sessions grouped by repo

Drill into one project. See its branches, its sessions, and the *effective*
merged config so you don't have to chase three settings files.

```
 grit  ·  /home/user/grit                                  43 sessions
─────────────────────────────────────────────────────────────────────────────
 active branches with sessions
   claude/plan-new-cli-tool        1 session    today
   claude/auth-fix                 4 sessions   yesterday
   main                           12 sessions   last 30 days

 effective config (merged)
   model            claude-opus-4-7        from project settings.json
   permissionMode   plan                   from session
   skills active    init, review, simplify (+ 4 user-level)
   hooks            stop-hook-git-check.sh from user settings.json
   MCP servers      github, gmail, drive
─────────────────────────────────────────────────────────────────────────────
 j/k move   enter session list for project   c open CLAUDE.md   s skills
```

- "Skills active" / "MCP servers" snapshot is pulled from the most recent
  session's `tools_changed` / `skill_listing` events — i.e. the *actual*
  runtime state, not whatever the static config files claim.

### 3.5 Re-run — relaunch with a past prompt

Pick a turn, edit the prompt if needed, fire off `claude` in a new pane.

```
 ▸ enter to launch in new pane                              source session:
                                                            add-jwt-refresh-flow
  prompt to re-run:
 ┌─────────────────────────────────────────────────────────────────────────┐
 │ implement refresh-token rotation in api-server. follow the JWT pattern  │
 │ already in place in middleware/auth.go, and write tests for the rotate  │
 │ window edge cases.                                                      │
 └─────────────────────────────────────────────────────────────────────────┘
  cwd:        /home/user/api-server  (auto-detected from session)
  branch:     [main]  (current HEAD; original was feat/auth-rotation)
  permission: [plan]  default acceptEdits  bypassPermissions
─────────────────────────────────────────────────────────────────────────────
 enter run · e edit prompt · w switch worktree · esc cancel
```

- `lore` shells out to `claude` with the chosen prompt and flags. No API key
  handling — just CLI invocation.
- Bonus: if the source session has a plan file, pre-populate it as a
  starting checklist.

---

## Tech / reuse from grit

| Need | Reuse from grit |
|---|---|
| Bubble Tea + Lipgloss scaffold | `grit.go:1302-1328` — `Run()` + `tea.NewProgram(...).WithAltScreen()` |
| Async fetch → `tea.Msg` pattern | `fetchCommits` / `fetchDiff` / `fetchBlame` shape in `grit.go` |
| Diff rendering for Edit/Write tool calls | `core/types.go` `diffLine` + `lineKind` enum; `parseDiff()` in `engine_parsing.go` |
| Filtering/search primitives | `core/filter.go` — generalize the query/author/since pattern for transcripts |
| Test fixtures | `engine_test_helpers.go` — copy `TestFixture` + `CommitBuilder` shape, port to `SessionFixture` |
| Clipboard | `copyToClipboard()` in `grit.go` (pbcopy / wl-copy / xclip) — verbatim |
| Caching | `engine_cache.go` LRU pattern for parsed-session cache |

New deps likely needed later (not in Phase 0 or 1):

- `github.com/sahilm/fuzzy` — fuzzy match in session list
- `modernc.org/sqlite` — pure-Go SQLite for FTS5 index (v2 search)

Nothing else. Stay lean.

---

## Phasing

| Phase | Deliverable | Status |
|---|---|---|
| 0 | `lore/DESIGN.md` (this file) | ✅ Done |
| 1 | `cmd/lore/main.go`, package skeleton, **session list** (3.1) end-to-end against real `~/.claude/projects/` | ✅ Done |
| 2 | **Session detail** (3.2) with collapsed tool calls and diff rendering | ✅ Done |
| 3 | **Search** (3.3) v1 — linear scan | ✅ Done |
| 4 | **Project view** (3.4) and **re-run** (3.5) | ✅ Done |
| 5a | **SQLite FTS5 search index** — replace linear scan with indexed full-text search | ⏳ Future |
| 5b | **List-level fuzzy matching** — live-filter as-you-type in the session list | ⏳ Future |
| 5c | **Cost/usage stats panel** — token usage and cost aggregated by project/branch/day/model | ⏳ Future |
| 6 | Standalone `github.com/zpenka/lore` repo | ✅ Done |
| 7 | **Quality-of-life** — sidechain handling, re-run UX, configurable projects dir | ⏳ Future |

Beyond the phased work, several quality-of-life items also landed:
inline fuzzy ranking for the `p` / `b` filters, a `?` help overlay with
mode-specific keybindings, per-mode viewport scrolling with edge-snap
offsets, and one-shot flash messages for no-op keys.

### Phase 5a — SQLite FTS5 search index

Replace the linear-scan `searchSessions()` with an FTS5-backed index.

- Add `modernc.org/sqlite` (pure-Go SQLite driver, already planned).
- Cache DB at `~/.cache/lore/index.db` (XDG-friendly, outside `~/.claude/`).
- On launch, diff session file mtimes against the index; re-index only
  changed files. Full rebuild on first run or schema change.
- Search query goes through FTS5 `MATCH`; results scored by `rank`.
- Fallback: if index is missing or corrupt, degrade to linear scan.

### Phase 5b — List-level fuzzy matching

Add a live-filter mode to the session list that fuzzy-matches as the user
types, across slug, project, and branch fields simultaneously.

- Reuse existing `sahilm/fuzzy` dependency.
- New key (likely `f`) enters filter-entry mode in the list. Each
  keystroke re-ranks `visibleSessions` in place.
- Distinct from `p` / `b` which scope to a single dimension.

### Phase 5c — Cost/usage stats

New `modeStats` mode aggregating token usage and estimated cost.

- Parse `assistant` events for token-count fields (need to verify what
  Claude Code actually writes to the JSONL — sample real files first).
- Dimensions: project, branch, day, model.
- Entry point: new key from list mode (e.g. `$` or `S`).
- Open question: are token counts reliably present in the JSONL, or do
  they need to be inferred from message length?

### Phase 7 — Quality-of-life

Smaller improvements identified during the 0.4.0 code review:

- **Sidechain handling.** Sub-agent transcripts (`isSidechain: true`)
  are currently ignored by `parseTurnsFromJSONL`. Inline-collapse them
  under the parent turn in detail view.
- **Re-enter list after re-run.** Instead of quitting lore when `claude`
  exits (`rerunDoneMsg` handler in `model.go`), return to the session
  list and surface any spawn errors.
- **`h` / `←` back-navigation in detail mode.** Vim / less muscle memory
  expects these to go back; currently only `esc` / `q` work.
- **Turn position indicator.** Show "turn N of M" in the detail header.
- **Configurable projects dir.** Support `LORE_PROJECTS_DIR` env var
  and/or `--dir` flag for non-default `~/.claude/projects/` locations.

---

## Open questions

Resolved during Phases 1–4:

- **Final name.** Landed on `lore`.
- **Write access.** Strictly read-only browser. Re-run (3.5) shells out to
  `claude` via `tea.ExecProcess` and mutates no state under `~/.claude/`.
- **Cache strategy** (v1). Raw JSONL read on every launch — fast enough in
  practice. Revisit when Phase 5a lands the SQLite FTS5 index.

Partially resolved:

- **Re-run UX.** Currently lore exits when `claude` returns (see
  `rerunDoneMsg` handler in `model.go:196-199`). The spawn error is
  silently discarded (`_ = msg.err`). Planned fix in Phase 7.

Still open:

- **Sidechain handling.** Sub-agent transcripts (`isSidechain: true`) — fold
  inline under the parent turn, or separate panel? Leaning inline-collapsed
  (Phase 7).
- **Cost/usage stats data availability.** Need to sample real JSONL files to
  confirm what token/cost fields Claude Code actually writes before
  designing Phase 5c.
- **FTS5 index location.** `~/.cache/lore/` follows XDG conventions but
  macOS doesn't have a standard cache dir. Consider `~/Library/Caches/lore/`
  on Darwin, `~/.cache/lore/` elsewhere.
