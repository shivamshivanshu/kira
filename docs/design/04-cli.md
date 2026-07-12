# CLI Reference

**Scope:** every `kira` subcommand — flags, files touched, commit behavior, JSON shapes, and the query/find/discover subsystems. The CLI is the system's only API; TUI and nvim call the same command implementations, never a second code path.

Part of the kira design set — see [DESIGN.md](../../DESIGN.md) for decisions and rationale.

## 1. Principles

- **noun-verb cobra tree.** `kira <noun> <verb> [args] [flags]`, e.g. `kira create ticket`, `kira hooks install`.
- **`--json` is the stable extension contract**, present on every read and write command. nvim, scripts, and CI parse only JSON, never human output, never `.kira/tickets/*.md` directly. Golden-file tested (see [09-testing.md](09-testing.md)) — a `--json` shape change is a contract change.
- **Human output is never parsed by tooling.** It is free to change between releases without a deprecation cycle; only `--json` shapes carry that guarantee.
- **Exit codes** *(proposed)*:

  | Code | Meaning |
  |---|---|
  | 0 | success |
  | 1 | user/validation error (bad flag, invalid state transition, malformed frontmatter) |
  | 2 | conflict/consistency error (ID collision, unresolved merge conflict) |
  | 3 | environment error (not a git repo, `.kira/` missing, `$EDITOR` unset when needed, git binary not found) |

- **ID argument resolution** — every `<id>` argument accepts, in this resolution order:
  1. full ULID (26 chars, e.g. `01J8X8Q7RZTN5Y3VXW2A9K4E7F`)
  2. unique ULID prefix (git-short-SHA style; ambiguous prefix → exit 1 listing matches)
  3. display number (`KIRA-142`, project key from config)
  4. alias (a stale number appended by `doctor` reconciliation — resolves forever)

  Resolution is documented once here; every command below reuses it without repeating it.

## 2. Global flags

| Flag | Description |
|---|---|
| `--json` | emit machine-readable JSON on stdout instead of human output |
| `--no-color` | disable ANSI color in human output (auto-disabled when stdout is not a tty) |
| `-C <path>` | run as if invoked from `<path>` *(proposed, cf. `git -C`)* — resolves `.kira/` relative to `<path>` instead of cwd |
| `--quiet` | *(proposed)* suppress non-essential human output (e.g. "staged 1 file" nags); errors still print |

## 3. Command reference

### `kira init`

```
kira init [--key PREFIX] [--force]
```

| Flag | Description |
|---|---|
| `--key` | project key for display IDs, default derived from repo directory name uppercased |
| `--force` | reinitialize over an existing `.kira/` (refuses otherwise) |

Behavior: scaffolds `.kira/{config.yaml, tickets/, templates/, hooks/}`, writes `.kira/.gitignore` (`.cache/`), writes default config and templates, creates an initial commit (`kira: init`). Does not touch the index (none exists yet). Not gated by `commit.mode` — init always commits so the scaffold is never left uncommitted.

```json
{"initialized": true, "path": ".kira", "project_key": "KIRA"}
```

### `kira hooks install`

```
kira hooks install [--with-pre-commit] [--validate]
```

| Flag | Description |
|---|---|
| `--with-pre-commit` | *(proposed)* additionally install the opt-in `pre-commit` hook, which runs `kira validate` on staged `.kira/` paths and blocks the commit on any reject-class violation |
| `--validate` | integrity check only — verifies installed hooks (`post-merge`, `prepare-commit-msg`, and `pre-commit` if present) and the merge-driver registration (`.git/config`/`.git/info/attributes`) are intact; modifies nothing (this is the one meaning of `--validate` across the doc set — installation is always via `--with-pre-commit` or the base command, never `--validate`) |

Behavior: copies/symlinks `.kira/hooks/{post-merge,prepare-commit-msg}` (and `pre-commit` if `--with-pre-commit`) into `.git/hooks/`, chaining onto any existing hook of the same name per the marker-bracketed chain mechanism in [07-git-integration.md §3](07-git-integration.md#3-hooks). Also registers the git merge driver for kira ticket files — writes `merge.kira.driver = kira merge-file %O %A %B` to the repo's **local** `.git/config` and a `.kira/tickets/*.md merge=kira` line to `.git/info/attributes` (local, not tracked `.gitattributes` — every clone must run `hooks install` for itself). Under `merge.policy: auto` (default), the driver auto-resolves same-field conflicts instead of leaving markers — see [`kira resolve`](#kira-resolve). Refuses and reports the conflicting hook rather than clobbering it. Does not commit (git-local, `.git/hooks/`, `.git/config`, and `.git/info/attributes` are never tracked).

### `kira create ticket` / `kira create epic`

```
kira create ticket [--title T] [--type TYPE] [--parent EPIC_ID] [--owner U] [--reporter U]
                    [--label L ...] [--estimate N] [--no-edit] [--from-file PATH] [--print-template] [--json]
kira create epic [...same flags]   # sugar: implies --type epic, template epic.md
```

| Flag | Description |
|---|---|
| `--title` | ticket title; if omitted, prompted for via `$EDITOR` unless `--no-edit` |
| `--type` | *(proposed, reserved)* item subtype for future use (e.g. bug/feature); v1 has exactly two types, selected by the `create ticket`/`create epic` subcommand itself — this flag is accepted but unused in v1, kept for forward compatibility |
| `--parent` | epic ULID/number to set as this item's `epic:` frontmatter field |
| `--owner`, `--reporter` | people fields; validated against `people.known` if `people.strict: true` |
| `--label` | repeatable; validated against `labels.known` if `labels.strict: true` |
| `--estimate` | numeric, unit per `config.yaml` `estimate.unit` |
| `--no-edit` | skip `$EDITOR`; create from flags only (fails if `--title` missing) |
| `--from-file` | read a fully-formed ticket body+frontmatter from `PATH` (the nvim create path — see [Editor flow](#6-editor-flow)) |
| `--print-template` | print the resolved template (`.kira/templates/<type>.md`) to stdout and exit; no ULID minted, no write, no commit — used by the nvim plugin to prefill its own scratch buffer before round-tripping via `--from-file` (see [06-nvim-plugin.md](06-nvim-plugin.md)) |

Behavior: mints a ULID (identity, filename) and the next display number on the current branch (`max(visible)+1`, no shared counter file — see ID scheme in DESIGN.md), applies flags/`$EDITOR`/`--from-file` content, writes `.kira/tickets/<ulid>.md`, `git add`s that one file, commits under `commit.mode: auto` (`kira: create KIRA-142 "Fix race in order-book snapshot merge"`, *proposed message format*). Writes the file directly — does not touch the index (index sees it on next incremental refresh).

```json
{"id": "01J8X8Q7RZTN5Y3VXW2A9K4E7F", "number": "KIRA-142", "type": "ticket",
 "title": "Fix race in order-book snapshot merge", "state": "TODO", "path": ".kira/tickets/01J8X8Q7RZTN5Y3VXW2A9K4E7F.md"}
```

`--print-template --json`:

```json
{"template": "---\ntitle: \ntype: ticket\nowner: \n...\n---\n\n## Description\n\n## Acceptance criteria\n"}
```

### `kira show`

```
kira show <id> [--json]
```

Behavior: read-only. Resolves `<id>`, reads `.kira/tickets/<ulid>.md` **directly** (file is truth, bypasses the index for the item's own fields/body). Linked commits come from the index's `commit_links` table (populated by the incremental trailer scan — see [07-git-integration.md §1](07-git-integration.md#1-commit-linking-convention)). History tail is the last N events from the index's cached event stream (see [08-telemetry.md](08-telemetry.md) §1), not a live `git log` — full history is `kira log <id>`.

```json
{
  "id": "01J8X8Q7RZTN5Y3VXW2A9K4E7F", "number": "KIRA-142", "aliases": [],
  "type": "ticket", "title": "Fix race in order-book snapshot merge",
  "state": "IN_PROGRESS", "category": "doing", "priority": "P1",
  "owner": "shivam", "reporter": "shivam", "labels": ["bug", "orderbook"],
  "epic": "01J8X7B1Q2W3E4R5T6Y7U8I9O0", "blocked_by": ["01J8X9F2M3W7VJQK8N5R6T1B0C"],
  "blocks": [], "estimate": 3,
  "created": "2026-07-10T09:14:00+05:30", "updated": "2026-07-12T11:02:00+05:30",
  "body": "## Description\n\n...", "comments": [{"id": "01J8XA1F...", "author": "shivam", "ts": "...", "text": "..."}],
  "linked_commits": [{"sha": "a1b2c3d", "subject": "fix acquire fence", "author": "shivam", "ts": "..."}],
  "history_tail": [{"ts": "...", "field": "owner", "from": null, "to": "shivam"}]
}
```

### `kira edit`

```
kira edit <id> [--field k=v ...] [--from-file PATH] [--json]
```

| Flag | Description |
|---|---|
| `--field` | repeatable `key=value`; flag-only edit, skips `$EDITOR` |
| `--from-file` | round-trip an edited file (nvim `:w` path) |

No flags at all → opens `$EDITOR` on the current file content (see [Editor flow](#6-editor-flow)). Behavior: writes the one ticket file, commits (`kira: KIRA-142 edit title,labels`, *proposed*, field names comma-joined). Direct file write, no index dependency for the mutation itself.

### `kira move`

```
kira move <id> <state> [--force] [--activate] [--json]
```

| Flag | Description |
|---|---|
| `--force` | bypass the transition adjacency-map check (still an ordinary audited write) |
| `--activate` | additionally set `.cache/active` to this item (drives `prepare-commit-msg` trailer auto-insert) |

Behavior: loads the item's type workflow from `config.yaml`, validates `<state>` is reachable from the current state via `transitions[current]` (exit 1 if not, unless `--force`). Writes `state:` (and `updated:`), commits (`kira: KIRA-142 state TODO -> IN_PROGRESS`). This is the transition the event stream in `08-telemetry.md` is built from.

### `kira assign`

```
kira assign <id> <user> [--reporter] [--json]
```

`--reporter` targets the `reporter:` field instead of `owner:` (default). Validates against `people.known` if strict. Writes + commits (`kira: KIRA-142 assign owner alice`).

### `kira link`

```
kira link <id> [--epic ID] [--blocked-by ID] [--remove] [--json]
```

| Flag | Description |
|---|---|
| `--epic` | set/clear (`--remove`) this item's `epic:` parent pointer |
| `--blocked-by` | add/remove (`--remove`) an entry in `blocked_by[]` |

Single-sided storage: `blocked_by` and `epic` are the only stored edges; `blocks` and "children of epic" are index-derived inverses, never written. `kira unlink` from the founder's original union is pruned — folded into `link --remove`.

### `kira comment`

```
kira comment <id> [-m "text"]
```

No `-m` → opens `$EDITOR`. Appends an anchored HTML-comment-delimited block to the body (append-only; concurrent comments from two branches are disjoint appended regions, merge clean by construction). Commits (`kira: KIRA-142 comment`).

### `kira list`

```
kira list [--type T] [--state S] [--category C] [--owner U] [--label L] [--epic ID] [--tree] [--json]
```

Behavior: filters are ANDed. Reads the index if fresh; pre-M2 or on index failure, falls back to a linear scan of `.kira/tickets/*.md` (same result, slower — the index is a cache, not a dependency). `--tree` groups by epic (same renderer as `query`'s default).

```json
{"items": [{"id": "01J8X8Q7...", "number": "KIRA-142", "title": "...", "type": "ticket",
            "state": "IN_PROGRESS", "category": "doing", "owner": "shivam", "labels": ["bug"], "epic": "01J8X7B1..."}],
 "count": 1}
```

### `kira query`

```
kira query "<expr>" [--tree|--flat] [--json]
```

See [§4 Query grammar](#4-query-expression-grammar). `--tree` (default) groups results by epic in a collapsible-in-TUI tree; `--flat` is a linear list. Index-backed (falls back to linear scan pre-M2, same as `list`). Output shape identical to `list`'s `{"items": [...]}`, plus a `"tree"` key when tree-rendered:

```json
{"items": [...], "tree": [{"epic": "01J8X7B1...", "epic_number": "KIRA-100", "items": ["01J8X8Q7..."]}]}
```

### `kira find`

```
kira find "<pattern>" [-i] [-w] [-C N] [--json]
```

Thin wrapper over `rg` scoped to `.kira/tickets/`, passing through `-i`/`-w`/`-C` (and other rg flags) verbatim. Detects `rg` via `exec.LookPath`; if absent, falls back to a pure-Go regex scan over the same files (slower, no `-C` context — degraded, not broken). This is unstructured text search — see external-tool policy below for why it never substitutes for `query`.

```json
{"matches": [{"id": "01J8X8Q7...", "number": "KIRA-142", "line": 94, "text": "The snapshot merge path drops updates..."}]}
```

### `kira discover`

```
kira discover [--action show|edit] [--fzf]
```

fzf picker over item titles/numbers with a `kira show <id>` preview binding (fzf `--preview`). No `rg`/rg detection here — falls back to a bubbles in-process fuzzy list when `fzf` is absent. No `--json`: this is an interactive selector, not a scriptable read command — its candidate source is the same data `list --json` exposes, so scripts use `list`/`query` directly instead.

### `kira tree`

```
kira tree [<id>] [--json]
```

Explorer-style hierarchy render: all epics with children, or (given `<id>`) that epic's subtree. Index-backed. Traversal keeps a visited-set of item IDs; if an `epic` cycle survives to read time (the `doctor`-flagged case above went unrepaired), traversal stops and reports the cycle instead of looping forever.

```json
{"root": null, "nodes": [{"id": "01J8X7B1...", "number": "KIRA-100", "type": "epic", "title": "...",
                           "children": [{"id": "01J8X8Q7...", "number": "KIRA-142", "type": "ticket", "title": "..."}]}]}
```

### `kira board`

```
kira board [<epic-id>] [--plain] [--json]
```

Kanban view, columns = the item type's configured states in declared order. Launches the TUI board screen by default (tty); `--plain` (or non-tty stdout) renders a static lipgloss table instead — same data path either way, one board implementation shared with the full TUI (see [05-tui.md](05-tui.md)). `--json`:

```json
{"columns": [{"state": "TODO", "category": "todo", "items": [...]}, {"state": "IN_PROGRESS", "category": "doing", "items": [...]}]}
```

### `kira log`

```
kira log <id> [--field NAME] [--since DATE] [--commits] [--json]
```

| Flag | Description |
|---|---|
| `--field` | restrict to one frontmatter field's change events |
| `--since` | only events after this date |
| `--commits` | include linked-commit entries interleaved (default: on) |

Behavior: `git log --follow -p -- .kira/tickets/<ulid>.md` (live shell-out; `--follow` is safe here because the file is never renamed) piped through a frontmatter-aware structural diff to produce field-change events, interleaved chronologically with `commit_links` rows from the index. This is the sole "history" mechanism — nothing is stored beyond git's own object model. See [08-telemetry.md](08-telemetry.md) for how this feeds cycle/lead time.

```json
{"events": [{"ts": "2026-07-11T18:30:00+05:30", "kind": "field_change", "field": "state", "from": "TODO", "to": "IN_PROGRESS", "commit": "a1b2c3d"},
            {"ts": "2026-07-11T18:31:00+05:30", "kind": "linked_commit", "sha": "a1b2c3d", "subject": "fix acquire fence"}]}
```

### `kira stats`

```
kira stats [<epic-id>] [--since DATE] [--weeks N] [--json]
```

Full metric definitions in [08-telemetry.md](08-telemetry.md). Reads the index's cached transition-event stream (populated incrementally during reindex, not a live per-item `git log --follow` — that would be O(items × history) and too slow for `stats` over a large project). Recursive epic-subtree rollup keeps a visited-set and reports (never loops) if an unrepaired `epic` cycle survives to read time — same guard as `kira tree`.

### `kira index`

```
kira index [--full] [--watch]
```

| Flag | Description |
|---|---|
| `--full` | explicit full rebuild (discard and re-derive `.cache/index.db` from scratch) |
| `--watch` | *(proposed, minor)* foreground fs-watch, reindexes on save; convenience only, never required |

Incremental refresh is also the implicit first step of every other command that reads the index: HEAD SHA + working-tree dirty-hash check against `.cache/meta.json`; `git merge-base --is-ancestor <watermark> HEAD` false ⇒ automatic full rescan (rebase detected). Also runs the commit-trailer incremental scan (folds in the founder's `scan-commits`, pruned as a separate command).

### `kira doctor`

```
kira doctor [--fix] [--json]
```

Read-only audit by default: ID collisions (two ULIDs sharing a number), `epic_cycle` (the `epic` parent chain loops back on itself), dangling `epic`/`blocked_by` references, schema/state-machine violations, index staleness, missing `.git/hooks` installation. `--fix` additionally repairs: renumbers the later-created colliding ULID to the next free number, appends its old number to `aliases:`, commits the repair visibly (`kira: doctor renumbered KIRA-143 -> KIRA-151`). This is what the `post-merge` hook invokes automatically.

```json
{"issues": [{"kind": "id_collision", "number": "KIRA-143", "ids": ["01J8...", "01J9..."], "fixed": true, "new_number": "KIRA-151"}],
 "index_stale": false, "hooks_installed": true}
```

### `kira resolve`

```
kira resolve <id> [--field NAME] [--interactive]
```

| Flag | Description |
|---|---|
| `--field` | restrict to one field (with `--interactive`: jump straight to that field's picker) |
| `--interactive` | force the manual field-by-field picker even under `merge.policy: auto` — the escape hatch, and how to redo a specific resolution after the fact |

Kira never shows raw git conflict markers by default — `merge.policy: auto` (default config `merge.policy`) resolves same-field conflicts automatically via the shared merge-policy engine (scalars: last-write-wins by `updated` timestamp, exact tie → the **incoming/remote** side wins, defined independently of git's merge-vs-rebase stage labeling — kira detects whether a rebase is in progress (`.git/rebase-merge`/`rebase-apply` present) and re-maps index stages 2/3 (or driver `%A`/`%B`) to absolute local/remote roles *before* applying the tie-break, since `kira sync`'s `git pull --rebase` inverts ours/theirs relative to a plain merge; lists: three-way set merge, union of adds, removals respected (except `aliases`, union-only — see [09-testing.md §2](09-testing.md#2-unit-tests)); comments: union by comment id; body: `git merge-file` text merge, whole-body LWW fallback on a hunk conflict; `created` immutable, `updated` = max of both sides). This is the same engine the `kira merge-file` git merge driver runs (registered by `kira hooks install`), so a plain `git merge`/`git rebase` and a `kira sync` produce byte-identical output for the same conflict — including which absolute side wins an exact-tie scalar.

No flags → applies the auto policy to every currently-conflicted kira file (reads the three git stages via `git show :1:`/`:2:`/`:3:`, applies the engine, writes back clean single-version YAML, `git add`s the result). `--interactive` → the manual field-aware base/ours/theirs picker, used either because `merge.policy: manual` is configured or as a manual override/re-do of a specific field's auto-resolution. Does not itself run `git commit`/`git rebase --continue` — the user concludes the operation normally. Every auto-resolution is reported (which field, which side won) rather than applied silently; the losing write is never destroyed — it stays reachable via `kira log <id>` because git retains both merge parents.

### `kira sync`

```
kira sync [--push] [--commit|--stash] [--remote <name>]
```

| Flag | Description |
|---|---|
| `--commit` \| `--stash` | dirty-kira-state strategy before pulling: commit staged kira paths (per `commit.mode`) or stash them, rather than refusing on a dirty tree |
| `--push` | push after a clean `doctor` pass; also settable as a standing default via config `sync.push: true` |
| `--remote <name>` | *(proposed)* target a non-default git remote |

Behavior — imitates git's own pull/push separation rather than inventing a new sync model (the remote *is* the git remote, nothing kira-specific): (a) ensure clean kira state — auto-commit staged kira paths per `commit.mode`, or apply the `--commit`/`--stash` strategy on a dirty tree; with **neither** flag given and dirty kira paths that aren't auto-committable under the active `commit.mode` (i.e. `manual`/`prompt` with unstaged or staged-uncommitted changes), `sync` refuses outright with exit 1, pointing at `--commit`/`--stash`, rather than guessing; (b) `git pull --rebase` — under `merge.policy: auto` (default), a same-field conflict in a kira file is resolved automatically by the shared policy engine ([`kira resolve`](#kira-resolve)) and the rebase continues without dropping the user into markers; under `merge.policy: manual`, or for any non-kira-file conflict, the rebase stops normally and points at `kira resolve --interactive`/plain git tooling; (c) `kira doctor --fix`; (d) incremental reindex; (e) report new/changed items, renumbers, auto-merged fields per item, and any conflicts still needing manual attention. `--push` (or `sync.push: true`) pushes only after step (c) passes clean — default stays pull-only, so `sync` never publishes local work unless explicitly asked. Exit code 2 only when a conflict is left unresolved (`merge.policy: manual`, or a non-kira-file conflict). See [07-git-integration.md](07-git-integration.md) "Remote & collaboration model" for the remote/publish model this composes.

```json
{"pulled": true, "pushed": false, "new_items": ["KIRA-150"],
 "renumbers": [{"from": "KIRA-143", "to": "KIRA-151"}],
 "auto_merged": [{"id": "KIRA-142", "field": "state", "winner": "remote"}],
 "conflicts": []}
```

### `kira commit`

```
kira commit [-m "msg"]
```

`commit.mode: manual` only. Commits exactly the paths under `.kira/` that are dirty (never unrelated staged files elsewhere in the repo). Default message if `-m` omitted summarizes the staged kira changes (`kira: N changes staged`, *proposed*).

### `kira config`

```
kira config get <key>
kira config set <key> <value>
kira config edit [--json]
```

`get`/`set` are dotted-path accessors into `.kira/config.yaml` (e.g. `commit.mode`), scriptable, no `$EDITOR`. `edit` opens `$EDITOR` on the full file, validates on save (same parse-validate-retry loop as tickets). All three commit under the active `commit.mode` (config changes are kira-tracked mutations too). `get` on a **non-leaf** key returns the whole subtree as nested JSON mirroring the YAML structure, rather than erroring — e.g. `kira config get workflows --json` returns every type's full workflow block (this is the shape the nvim plugin's `:KiraCreate` field completion consumes, see [06-nvim-plugin.md](06-nvim-plugin.md)):

```json
{"key": "commit.mode", "value": "auto"}
```

```json
// kira config get workflows --json
{
  "key": "workflows",
  "value": {
    "ticket": {
      "states": [
        {"key": "TODO", "category": "todo"},
        {"key": "IN_PROGRESS", "category": "doing"},
        {"key": "REVIEW", "category": "doing"},
        {"key": "DONE", "category": "done"},
        {"key": "WONT_DO", "category": "done", "resolution": "dropped"}
      ],
      "initial": "TODO",
      "transitions": {
        "TODO": ["IN_PROGRESS", "WONT_DO"],
        "IN_PROGRESS": ["REVIEW", "TODO", "WONT_DO"],
        "REVIEW": ["DONE", "IN_PROGRESS"],
        "DONE": [],
        "WONT_DO": []
      }
    },
    "epic": { "states": ["..."], "initial": "PLANNED", "transitions": {"...": "..."} }
  }
}
```

### `kira validate`

```
kira validate <file> [--json]
```

Plumbing: validates one ticket/config file's frontmatter schema + state-machine legality, no write, no commit. Exit 1 on any violation with a message per problem. This is what nvim's `BufWritePost` and the optional pre-commit hook call — never a user-facing workflow command on its own.

```json
{"valid": false, "errors": [{"field": "state", "message": "REVIEW is not reachable from DONE"}]}
```

### `kira merge-file` (plumbing — git merge-driver entry point)

```
kira merge-file <base> <ours> <theirs>
```

Not a user-facing command — this is the executable `kira hooks install` registers as `merge.kira.driver` (git's merge-driver contract: three temp-file paths for the common ancestor / current branch / other branch content). Runs the same merge-policy engine `kira resolve` uses, writes the merged result to `<ours>` in place, exits 0 (clean or auto-resolved merge) or 1 (a conflict remains — only reachable under `merge.policy: manual`, in which case it leaves git's ordinary conflict markers in `<ours>` for `kira resolve --interactive`/manual editing). Called by plain `git merge`/`git rebase` on any `.kira/tickets/*.md` path once the driver is registered — this is what makes a bare `git pull` (outside `kira sync`) produce the same auto-resolved result as `kira sync` does.

### `kira completion`

```
kira completion bash|zsh|fish
```

Standard cobra-generated shell completion script to stdout.

### `kira tui`

```
kira tui [--board]
```

Launches the bubbletea program; `--board` opens directly on the kanban screen instead of the default tree/explorer. Bare `kira` (no subcommand) is equivalent to `kira tui` — see [05-tui.md](05-tui.md).

### `kira serve` (M6 stretch)

Opt-in per-repo warm-cache process speaking line-delimited JSON over stdin/stdout, holding the parsed index in memory to avoid the ~ms-scale reindex cost on every invocation. Never a correctness dependency — restartable with zero data loss (it holds no state that isn't rederivable from git + `.cache/index.db`), and the nvim plugin falls back to spawning the CLI directly if the socket/pipe isn't present. Not in v1 scope; specced here only so `--json` consumers don't need to change when it lands.

### Pruned from the union (do not implement)

| Not implemented | Folded into |
|---|---|
| `kira unlink` | `kira link --remove` |
| `kira scan-commits` | `kira index` (trailer scan is part of incremental refresh) |
| `kira ls` | `kira list` (no alias — one name) |
| `kira history` | `kira log` (duplicate name for the same command) |
| `kira merge-driver` | not a separate command name — the driver entry point is `kira merge-file` (v1, registered by `hooks install`) |

## 4. Query expression grammar

*(proposed — concrete spec)*

```ebnf
query      = or_expr ;
or_expr    = and_expr , { "OR" , and_expr } ;
and_expr   = not_expr , { [ "AND" ] , not_expr } ;   (* adjacency implies AND *)
not_expr   = [ "NOT" ] , primary ;
primary    = "(" , or_expr , ")" | predicate | term ;
predicate  = field , cmp , value ;
field      = "state" | "category" | "owner" | "label" | "type" | "epic" | "priority" | "created" | "updated" ;
cmp        = "=" | "!=" | ">" | ">=" | "<" | "<=" ;
value      = quoted_string | bare_word | date ;
term       = quoted_string | bare_word ;              (* falls through to a title substring match *)
date       = ? RFC3339 date, e.g. 2026-07-01 ? ;
```

`>`/`>=`/`<`/`<=` are only valid on `created`/`updated` (date comparison); other fields accept only `=`/`!=`.

Examples:

```
state=IN_PROGRESS AND owner=shivam
label=bug OR label=perf
category=doing AND NOT owner=alice
epic=KIRA-100 AND created>2026-07-01
race AND priority=P1                    # "race" falls through to a title substring match
```

Default render is the epic-grouped tree (`kira tree`'s renderer); `--flat` gives the linear `list`-style output.

## 5. `find` / `discover` and the external-tool policy

**Founder constraint, enforced structurally:** `rg` and `fzf` are optional accelerators for *interactive, unstructured text search only* — `kira find` and `kira discover`. Every structured lookup (`query`, `list`, `board`, `stats`) parses frontmatter (or reads the index) and never shells out to `rg`/`grep`/`awk`/`sed`. Reason: a text match on the literal string `label` would false-positive inside prose in the ticket body — structured fields must go through the parser, not pattern matching. No `awk`/`sed` invocation exists anywhere in the system.

`find` and `discover` detect their optional binary via `exec.LookPath` at call time (not at startup) and degrade rather than fail:

| Tool | Present | Absent |
|---|---|---|
| `rg` | passthrough exec, full flag support | pure-Go regex scan over `.kira/tickets/*.md`, no `-C` context |
| `fzf` | subprocess picker, `kira show` preview | bubbles in-process fuzzy list, no live preview pane |

`kira doctor` checks both on `PATH` and emits an install hint if missing — a degraded `find`/`discover` is a warning, not a broken build.

## 6. Editor flow

- `$EDITOR` invocation prefills from `.kira/templates/{ticket,epic}.md` for `create`, or the current file content for `edit`/`config edit`.
- **Parse-validate-retry loop** *(proposed)*: on save, kira re-parses the file; if invalid, it reopens `$EDITOR` on the same buffer with an error banner prepended as an HTML comment block (`<!-- kira:error ... -->`) above the frontmatter, and loops until valid or the user aborts (empty save / no change from the error state).
- `--no-edit`: flag-only creation, no `$EDITOR` call — fails fast if required fields (title) are missing rather than falling back to the editor.
- `--from-file PATH`: reads a complete, already-parseable file from `PATH` and validates it exactly as the editor-save path would (the nvim round-trip: nvim writes a scratch buffer, kira validates it as if it were an `$EDITOR` save). No editor process spawned.
- `-` on stdin *(proposed)*: `--from-file -` reads the file body from stdin instead of a path, for programmatic pipelines.

## 7. Machine-interface guarantees

- **JSON schema stability**: additive-only within a major version — new keys may appear, existing keys never change type or disappear, until a major version bump. This is the guarantee the nvim plugin and CI scripts build against.
- All JSON goes to **stdout only**; all diagnostics, progress, and human-readable errors go to **stderr**, even in `--json` mode (a script piping stdout to `jq` never sees a diagnostic corrupt the JSON).
- **Deterministic ordering** *(proposed)*: list-shaped results sort by display number ascending, ties broken by ULID — stable across repeated invocations with no intervening writes, which is what golden-file tests in [09-testing.md](09-testing.md) depend on.
