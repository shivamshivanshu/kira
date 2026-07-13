# Architecture

**Scope:** system shape — process model, language/library choices, the index, concurrency, and the single-write-path guarantee that ties CLI/TUI/nvim together.
Part of the kira design set — see [DESIGN.md](../../DESIGN.md) for decisions and rationale.

## 1. Component diagram

```
 nvim (Lua, thin client)                                    other tools / CI
   │  vim.system({'kira', ..., '--json'})                     │  `kira ... --json`
   ▼                                                          ▼
┌───────────────────────────────────────────────────────────────────┐
│                         kira (one Go binary)                       │
│                                                                     │
│  cmd/ (cobra CLI tree)          tui/ (bubbletea; `kira tui`)       │
│         │                                 │                        │
│         └───────────────┬─────────────────┘                        │
│                          ▼                                         │
│               internal/core  — service layer                      │
│     (Create, Edit, Move, Link, Comment, Query, Log, Stats, …)     │
│     the ONE implementation each frontend calls — §6                │
│                          │                                         │
│            ┌─────────────┴─────────────┐                           │
│            ▼                           ▼                           │
│   internal/storage              internal/index                    │
│   (frontmatter read/write,      (sqlite cache: items, labels,      │
│    atomic rename)                links, commit_links, fts5 — §4)   │
│            │                           │                           │
└────────────┼───────────────────────────┼───────────────────────────┘
             ▼                           ▼
   .kira/tickets/*.md           .kira/.cache/index.db + meta.json
   .kira/config.yaml            (gitignored, derived, rebuildable)
   (canonical, git-tracked)
             │
             ▼
   internal/gitx (os/exec) ─────▶ system `git`  (log, show, interpret-trailers,
                                                  merge-base, status --porcelain)
                                  optional: `rg`, `fzf` via exec.LookPath — §7
```

The nvim plugin and any other external tool are `--json` consumers of the same binary a human runs interactively; there is no separate server process, RPC protocol, or library API. See [06-nvim-plugin.md](06-nvim-plugin.md).

## 2. Language & libraries

| Choice | Why |
|---|---|
| Go | Static binary, no runtime install, <5ms cold start — decisive since the CLI is invoked per-keystroke-adjacent from nvim (`gk`, virtual-text scans) and per-render from the TUI. |
| cobra | Standard noun-verb CLI tree; the stack used by `gh`, matching this tool's UX register. |
| bubbletea + bubbles + lipgloss | The mature Charm TUI stack; same category as `k9s`/`lazygit`, which the target user already runs daily. |
| glamour | Markdown rendering for ticket bodies in TUI/detail views — avoids hand-rolling a markdown renderer. |
| gopkg.in/yaml.v3 | Frontmatter + config parsing; preserves key order needed for the byte-stable writer (see [02-data-model.md](02-data-model.md)). |
| oklog/ulid | ULID generation for immutable item identity. |
| modernc.org/sqlite | cgo-free sqlite driver — keeps the binary static and cross-compilable; no libsqlite3 system dependency. |
| rogpeppe/go-internal/testscript | e2e tests that spawn the real binary against real temp git repos — see [09-testing.md](09-testing.md). |
| system `git` via os/exec | Behavior parity with the user's actual git (hooks, config, trailers, merge). |

Rejected:

| Alternative | Reason rejected |
|---|---|
| Python | Interpreter startup + venv/dependency friction disqualify a CLI invoked on a hot path (per-keystroke from nvim, per-frame-adjacent from the TUI). |
| Rust | Ecosystem for this exact TUI/CLI category is thinner than Go's (bubbletea/cobra have no equally mature Rust equivalent here); slower iteration; no perf requirement decisive enough to justify it. |
| TypeScript/Node | ~50-100ms node startup; distribution needs a bundled runtime or `pkg`-style bundler, losing the single-static-binary property that makes `.kira/` portable and dependency-free. |
| go-git (pure-Go git implementation) | Behavior parity with the user's real git — hooks, config, trailers, merge semantics — demands shelling to the actual `git` binary; a reimplementation risks silent divergence. |

## 3. Process model: stateless CLI, no daemon

No background process. Every invocation is a fresh process that discovers state, does its job, and exits.

Why no daemon:
- Workload is millisecond-scale (frontmatter parse + a bounded sqlite query); a daemon buys no latency win worth its cost.
- A daemon adds a lifecycle to manage (start/stop/crash-recovery), an IPC surface to secure and version, and a new bug class — stale cache after `git checkout`/`git worktree` switches underneath a long-lived process.
- Contradicts local-first: the tool must work correctly from a cold clone with zero setup beyond the binary.

Invocation lifecycle:

1. **Discover `.kira/`** — walk up from cwd (git-style: same algorithm as git's own repo discovery), stop at the first `.kira/` found or at the filesystem root / a `.git` boundary. Fail with a clear error if none found (except `kira init`).
2. **Staleness check** — compare `.cache/meta.json` against current git state; see §4.
3. **Incremental reindex** if stale (bounded by what changed since the last index, never a full rescan unless triggered — §4).
4. **Execute the command** against `internal/core`, which reads/writes ticket files directly and queries the index for anything beyond a single item.
5. **Exit.** No persisted in-memory state between invocations.

## 4. Index design

The index (`.kira/.cache/index.db`, sqlite) is a **derived, gitignored cache** — it is never authoritative and can be deleted with zero data loss; the next invocation rebuilds it from `.kira/tickets/*.md` and git history.

### Schema `(proposed)`

```sql
CREATE TABLE items (
  id       TEXT PRIMARY KEY,   -- ULID, immutable identity
  number   TEXT NOT NULL,      -- KIRA-n, current display number
  type     TEXT NOT NULL,      -- ticket | epic
  title    TEXT NOT NULL,
  state    TEXT NOT NULL,
  category TEXT NOT NULL,      -- todo | doing | done, denormalized from config for telemetry
  priority TEXT,
  owner    TEXT,
  reporter TEXT,
  epic     TEXT REFERENCES items(id),
  estimate REAL,
  created  TEXT NOT NULL,      -- RFC3339
  updated  TEXT NOT NULL,
  path     TEXT NOT NULL,      -- relative path, for re-read
  file_sha TEXT NOT NULL       -- content hash of the file, fine-grained staleness check
);

CREATE TABLE aliases      (item_id TEXT REFERENCES items(id), number TEXT, PRIMARY KEY (item_id, number));
CREATE TABLE labels       (item_id TEXT REFERENCES items(id), label  TEXT, PRIMARY KEY (item_id, label));
CREATE INDEX idx_labels_label ON labels(label);

CREATE TABLE links        (item_id TEXT REFERENCES items(id), kind TEXT, target_id TEXT,
                            PRIMARY KEY (item_id, kind, target_id));   -- kind = 'blocked_by' (blocks is the query-time inverse)

CREATE TABLE commit_links (item_id TEXT REFERENCES items(id), sha TEXT, subject TEXT,
                            author TEXT, ts TEXT, PRIMARY KEY (item_id, sha));

CREATE VIRTUAL TABLE body_fts USING fts5(item_id UNINDEXED, title, body);  -- backs `kira find` fallback / full-text query

CREATE TABLE events (        -- cached view of git-derived field history; fully rebuildable from `git log -p` (no --follow: ULID paths never rename, and --follow bleeds sibling history)
  item_id    TEXT, ts TEXT, field TEXT, old_value TEXT, new_value TEXT, commit_sha TEXT,
  PRIMARY KEY (item_id, commit_sha, field)
);
```

### `.cache/meta.json` contents

```json
{
  "schema_version": 1,
  "last_indexed_head_sha": "a1b2c3d...",
  "trailer_watermarks": { "refs/heads/main": "a1b2c3d..." },
  "dirty_hash": "sha256 of concatenated (path, content) for uncommitted tickets/ changes"
}
```

### Staleness algorithm

1. Read `meta.json`. Missing, unreadable, or `schema_version` mismatch → full rebuild (§below), no error surfaced to the user beyond a log line.
2. `head = git rev-parse HEAD`. If `head == meta.last_indexed_head_sha` and the dirty-hash (below) is unchanged → index is fresh, skip reindex.
3. If `head != meta.last_indexed_head_sha`:
   - `git merge-base --is-ancestor meta.last_indexed_head_sha head` → **incremental**: diff `tickets/` between the watermark and `head`, upsert changed items; scan `git log <watermark>..head` for trailers per the ref watermark (see [07-git-integration.md](07-git-integration.md)).
   - Not an ancestor (history rewritten — rebase/squash) → **full rescan**: truncate and rebuild all tables from `tickets/*.md` + full trailer scan.
4. Dirty working tree: `git status --porcelain -- tickets/`, hash the touched paths' content, compare to `meta.dirty_hash`. Changed → re-read exactly those files into the index (uncommitted changes aren't in git log, so this is a direct file re-read, not a git-log diff).
5. Any sqlite read error or schema mismatch encountered mid-command → drop `index.db`, full rebuild, continue.
6. Rewrite `meta.json` at the end of any index-touching run.

Single-item *writes* (`edit`, `move`, `assign`, `link`, `comment`) **bypass the index entirely** — they read/write the ticket file directly, no index dependency for the mutation itself. `kira show` reads the item's own frontmatter/body directly from the file, but fills `blocks`, `linked_commits`, and `history_tail` from the index — those fields degrade gracefully (empty + a stderr note) when the index is absent/stale, rather than blocking the read. Beyond that, the index backs multi-item operations: `list`, `query`, `find` (fts5 fallback), `board`, `stats`, `log` (commit_links / events).

## 5. Concurrency

- **Advisory lock** `.cache/lock` (flock-based, proposed: 2s acquire timeout, then fail loudly rather than hang — concurrent `kira` invocations against one repo are expected to be rare for a single-operator local tool) guards: (a) any index write, (b) any ticket-file mutation.
- **Ticket file writes**: temp-file + atomic rename (`os.Rename` within the same directory) — a reader never observes a partially-written file.
- **sqlite WAL mode** `(proposed)` — allows the TUI to hold a long-lived read connection (e.g., a live kanban board) while a concurrent `kira move` from another terminal writes without blocking the reader.

## 6. Single-write-path guarantee

Every mutation — a CLI command, a TUI keypress, an nvim `:w` on a scratch buffer — calls the same `internal/core` Go function. There is exactly one implementation of `Move`, `Edit`, `Link`, `Comment`, etc.; validation and commit semantics structurally cannot drift between frontends, because there is only one code path to drift from.

### Package layout

```
kira/
  cmd/kira/         main package: thin entry point; command logic lives in internal/cli
  internal/
    cli/            cobra command tree, flag parsing, --json marshal of core results
    core/           service layer: one function per verb, called by cli/ (and tui/ later)
    datamodel/      Item, Config, workflow/state-machine types shared by storage + core
    codec/          frontmatter parse/serialize + comment blocks (yaml.v3, byte-stable)
    config/         config load, defaults, sprint append
    id/             ULID generation, ref resolution, sort keys
    query/          query-language lexer, parser, eval
    storage/        ticket file I/O: discovery, store lock, atomic write
    gitx/           os/exec git wrappers: log, commit, stage, rev-parse
    rgx/ fzfx/      optional rg / fzf accelerators (exec.LookPath, pure-Go fallback)
    termx/ editorx/ terminal prompts + $EDITOR invocation
    errx/           error classification → exit codes
  tests/            contract goldens, e2e testscripts, integration
  go.mod
```

Still design-only: `index/` (§4) and `tui/` (05-tui.md).

## 7. External-tool policy

- **Hard requirement: `git` only.** Every other external tool is an optional accelerator.
- `rg`, `fzf` detected via `exec.LookPath` at command start; absent → pure-Go fallback (regex/bufio scan for `find`-style pattern search; `bubbles` list with fuzzy filtering for `discover`). `kira doctor` reports which optional tools are present/missing with install hints.
- **Structured queries never shell out to `rg`/`fzf`** — `list`/`query`/`board`/`stats` read parsed frontmatter or the index. Shelling out is reserved for interactive free-text search (`kira find`, `kira discover`) where `rg`/`fzf` genuinely outperform a hand-rolled scanner.
- No `awk`/`sed` dependency anywhere in the tool.

## 8. Machine interface

`--json` on every read/write command is the **sole** machine-consumable interface — no second protocol, no library API, no socket. This forces the CLI itself to be complete: anything the TUI or nvim plugin needs is something `--json` must expose, so there is never a feature reachable only through an internal API. Golden-file tests pin every `--json` shape (see [09-testing.md](09-testing.md)).

**M6 stretch — `kira serve`:** an opt-in, per-repo warm-cache process (stdin/stdout line-JSON) that holds the parsed index in memory to skip the staleness check on every call. It is a cache-holder only, never a correctness dependency — the nvim plugin falls back to spawning the CLI directly if `serve` isn't running or isn't reachable. Out of scope until M6.
