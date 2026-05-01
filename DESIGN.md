# lore — design sketch

A keyboard-driven TUI for browsing your Claude Code session history.

> **Status:** design doc only. No code yet. This file lives in the directory the
> tool will eventually occupy so it travels cleanly when we cut a separate repo.
>
> **Working name:** `lore`. Final name still open — see [Name candidates](#name-candidates).

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

## Name candidates

| Name | Why |
|---|---|
| **lore** | Your accumulated AI work history is your lore. Short, evocative. |
| **recall** | Action verb — exactly what the tool does. |
| **yarn** | A thread of conversation. Short, tactile. |
| **trail** | Breadcrumbs through past sessions. |
| **scrollback** | Terminal jargon, immediately legible to the target user. |

The rest of this doc uses `lore` as a placeholder.

---

## Repo layout

```
grit/
├── grit.go, engine_*.go, ...   # existing
├── core/                       # existing shared parsers
├── cmd/grit/                   # existing
└── lore/                       # NEW — this directory
    ├── DESIGN.md               # ← only file in this PR
    ├── go.mod                  # later — module github.com/zpenka/lore
    ├── cmd/lore/main.go        # later
    └── *.go                    # later
```

Why a nested directory with its own `go.mod` (later) rather than
`cmd/lore` + sibling package in grit's module:

- Eventual repo split is one command:
  `git filter-repo --subdirectory-filter lore`. No untangling.
- During shared development, `lore` can pull from `github.com/zpenka/grit/core`
  via a `replace` directive, then swap to a pinned version (or fork the
  parsers) at split time.
- Keeps grit's `go.mod` from accumulating TUI deps that `lore` adds later
  (fuzzy matchers, sqlite for FTS5, etc).

Trade-off: with two `go.mod` files, `go test ./...` from the repo root won't
recurse into `lore/`. Minor — CI just runs both modules.

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

This PR is **Phase 0** — design doc only.

| Phase | Deliverable |
|---|---|
| 0 | `lore/DESIGN.md` (this file) |
| 1 | `cmd/lore/main.go`, package skeleton, **session list** (3.1) end-to-end against real `~/.claude/projects/` |
| 2 | **Session detail** (3.2) with collapsed tool calls and diff rendering |
| 3 | **Search** (3.3) v1 — linear scan |
| 4 | **Project view** (3.4) and **re-run** (3.5) |
| 5 | SQLite FTS5 index, fuzzy matching, cost/usage stats panel |
| 6 | `git filter-repo --subdirectory-filter lore` → `github.com/zpenka/lore` |

---

## Open questions before Phase 1

- **Final name.** lore / recall / yarn / trail / scrollback / something else.
- **Cache strategy.** Read raw JSONL on every launch, or maintain
  `~/.claude/lore/cache.db`? Lean default: raw read for v1, cache only when
  measured slow.
- **Write access.** Default: strictly read-only browser. Re-run (3.5) shells
  out to `claude` rather than mutating any state under `~/.claude/`.
- **Sidechain handling.** Sub-agent transcripts (`isSidechain: true`) — fold
  inline under the parent turn, or separate panel? Probably inline-collapsed.
