# kira — Design

Git-native, local-first, terminal-first project management for software repos. JIRA's data model with git's storage model: tickets are files in your repo, history is `git log`, sync is `git push`.

**Status:** draft for founder review. Produced from a 5-way independent design fan-out + consensus synthesis (2026-07-12).
**Detailed specs:** [docs/design/](docs/design/) · **Execution plan:** [ROADMAP.md](ROADMAP.md)

## 1. Problem

Ticketing for solo/small-team repo work today means either a heavyweight hosted tool (JIRA/GitHub Issues: online-only, context-switch out of the terminal, data lives outside the repo) or nothing (TODO comments, untracked markdown). Neither gives: tickets versioned *with* the code they describe, branch-local ticket state, terminal/nvim-native workflow, or greppable plain-text data. Prior art each solves a slice: `git-bug` (git-native, but object-store opaque — not greppable/hand-editable), `backlog.md` (plain files, but no index, no workflow enforcement, no commit linking), `taskwarrior` (great CLI, not repo-scoped, not git-merged), `jira-cli` (thin client on hosted JIRA). kira composes the missing whole.

## 2. Goals and principles

- **Git is the database.** Canonical state is git-tracked plain text in `.kira/`; diffable, mergeable, PR-reviewable, survives with nothing but a clone.
- **Local-first, offline-always.** No server, no account, no network dependency. **The remote is the git remote**: `.kira/` travels inside the repo, so every git collaboration mechanism — fetch/pull/push, branches, PRs, forks, protected branches — transfers verbatim to tickets. `kira sync [--push]` wraps the pull-rebase→repair→reindex(→push) cycle; multi-user workflows deliberately imitate git's.
- **Terminal/nvim-native.** CLI for scripting, TUI for browsing, nvim plugin as first-class frontend.
- **Don't reinvent the wheel.** Mature deps (cobra, bubbletea) and system tools (`git`, `rg`, `fzf`) over bespoke code; novel effort goes only into what doesn't exist.
- **One write path.** CLI flag, TUI keypress, and nvim `:w` converge on the same core function — validation and commit semantics cannot drift between frontends.
- **Structured data is parsed, not grepped.** External tools (`rg`, `fzf`) accelerate *interactive text search* only; queries over fields (state, labels, parents) always go through parsed frontmatter or the derived index. No `awk`/`sed` plumbing anywhere.

## 3. Constraints and non-goals

Constraints: single static binary; `<~10ms` cold start (invoked per-keystroke from nvim); only hard external dependency is `git`; Linux + macOS CI'd (Windows untested, not deliberately broken); nvim ≥ 0.10.

Non-goals (v1): web UI · real-time sync/notifications/webhooks · hosted service or required daemon · permissions beyond git's own · first-class sprints (field reserved) · custom field schemas · attachments · cross-repo linking · auto-transition-on-merge · Jira import/export · non-git backends · AI ticket authoring.

## 4. Options considered

### 4.1 Implementation language

| | Go (chosen) | Rust | TypeScript/Node | Python |
|---|---|---|---|---|
| Cold start | ~5ms static binary | ~5ms static binary | 50–100ms runtime spin-up | 100ms+ interpreter |
| Distribution | single binary | single binary | needs runtime or bundler (pkg/bun) | venv/pip friction |
| TUI/CLI ecosystem | best-in-class for exactly this category: cobra, bubbletea/lipgloss (gh, k9s, lazygit, glow) | ratatui/clap solid but thinner; slower iteration | ink/oclif exist; TUI depth well below Charm | rich/textual good but startup kills the nvim hot path |
| Verdict | **library support + startup + distribution all align** | no decisive perf need justifies the iteration cost | loses single-binary property | disqualified by startup |

The CLI is invoked synchronously from nvim on cursor actions — startup latency is a hard requirement, which eliminates the runtime-hosted languages regardless of library quality. Git access **shells out to system `git`** (not go-git): behavior parity with the user's real git — hooks, config, trailers — is worth a subprocess.

### 4.2 Canonical storage

| | Markdown + YAML frontmatter files (chosen) | SQLite as canonical | Git object store (git-bug style) |
|---|---|---|---|
| Merge | 3-way text merge, mostly clean by layout design | binary blob: unmergeable | custom CRDT-ish machinery |
| Greppable / hand-editable / PR-reviewable | yes — the point | no | no |
| Query speed at scale | needs derived index past ~1k items | fast natively | needs custom tooling |

**Decision:** git-tracked file-per-item is canonical; SQLite exists only as a **gitignored, always-rebuildable derived index** (`.kira/.cache/`). Deleting the cache loses nothing. This is the hybrid that keeps both properties.

### 4.3 Per-ticket history

| | Derive from git log (chosen) | Snapshot linked-list in files (original idea) | Append-only event log file |
|---|---|---|---|
| Storage cost | zero | duplicates every revision | grows per edit |
| Correctness | inherits git's authorship/timestamps/graph | drifts on hand-edit; duplicates git's object model badly | second source of truth to reconcile |
| Merge surface | none added | doubled | append conflicts |

**Decision:** no stored history. `kira log ID` = `git log --follow -p` on the item file → structural frontmatter diff → field-level events ("owner: alice→bob"), interleaved with trailer-linked code commits. The commit graph *is* the linked list the original idea wanted.

### 4.4 Process model

**Stateless CLI, no daemon** (unanimous). Workload is parsing hundreds of small files + SQLite reads — milliseconds; a daemon adds lifecycle/IPC/stale-cache-across-checkout bug classes and contradicts local-first. Index freshness is checked per invocation (HEAD SHA + dirty-file hash watermark) with incremental reindex. Opt-in `kira serve` warm-cache process is an M6 stretch, never a correctness dependency.

### 4.5 ID scheme

**Two-tier:** immutable **ULID** identity (filename + all cross-references) + human **display number** `KIRA-n` (max-visible+1 at create; no shared counter file — that would conflict on every concurrent create). Post-merge number collisions are repaired deterministically by a `post-merge` hook running `kira doctor --fix`; the losing item is renumbered and keeps its old number as a permanent alias. Because references key on ULIDs only, a numbering bug can never corrupt a link. Fallback `id.style: hash` (ULID-derived display, zero reconciliation) is specced for teams that prefer no machinery over pretty numbers. Rejected: sequential-only (branch collisions), hash-only (loses KIRA-142 ergonomics).

### 4.6 Strongest arguments against the chosen design

Stated up front: (a) JIRA-style last-writer-wins auto-resolution (the default `merge.policy: auto`) can silently pick a loser when two people move the same field concurrently — accepted because git preserves both parents (the losing write stays recoverable and visible in `kira log`, which is more audit trail than JIRA offers) and every auto-merge is reported; teams that want human arbitration set `merge.policy: manual`; (b) number reconciliation depends on a hook being installed — degraded loudly (every index build warns on duplicates) but not impossible to ignore; (c) auto-commit mode adds noise commits to git history — filterable by the `kira:` prefix, and `manual` mode exists.

## 5. Decision summary

| Axis | Decision | Spec |
|---|---|---|
| Language / libs | Go · cobra · bubbletea/bubbles/lipgloss/glamour · yaml.v3 · oklog/ulid · modernc.org/sqlite; system git/rg/fzf via exec | [01-architecture](docs/design/01-architecture.md) |
| Architecture | stateless CLI, no daemon; lazy incremental index; single write path | [01-architecture](docs/design/01-architecture.md) |
| Data model | unified Item (ticket/epic by `type:`); single-sided edges (parent on child, `blocked_by` stored / `blocks` derived); append-only comment blocks; config-defined state machine with categories | [02-data-model](docs/design/02-data-model.md) |
| Storage & format | `.kira/` file-per-item, markdown + block-style YAML frontmatter, byte-stable writer (one field edit = one-line diff) | [03-storage-and-git](docs/design/03-storage-and-git.md) |
| Conflicts | JIRA-simple by default: `merge.policy: auto` — field-level three-way auto-merge (LWW scalars, set-merge lists, comment union), applied by `kira sync` and a hooks-installed merge driver; users never see conflict markers; `manual` opt-in for git-style arbitration | [03-storage-and-git](docs/design/03-storage-and-git.md) |
| History | derived from `git log --follow`; commit modes auto/manual/prompt (default auto — open question) | [03-storage-and-git](docs/design/03-storage-and-git.md) |
| CLI | noun-verb tree; `--json` on every command is the single machine API | [04-cli](docs/design/04-cli.md) |
| TUI | lazygit-style panel GUI launched by bare `kira`: tree explorer, kanban, detail, stats; single-key actions + modal popups; `:` bar reuses CLI grammar; Nerd Font glyphs with ASCII fallback | [05-tui](docs/design/05-tui.md) |
| nvim | thin Lua client over `kira --json` via `vim.system`; never parses files; `gk` ambient ticket jump; optional mini.hipatterns state highlighting + nvim-web-devicons (pcall-detected, graceful fallback) | [06-nvim-plugin](docs/design/06-nvim-plugin.md) |
| Commit linking | `Kira-Ticket:` git trailer + `git interpret-trailers`; watermarked incremental scan; opt-in hooks for zero-friction linking | [07-git-integration](docs/design/07-git-integration.md) |
| Telemetry | on-demand from index + derived events: completion, cycle/lead time p50/p90, throughput | [08-telemetry](docs/design/08-telemetry.md) |
| Testing | byte-stability goldens, `--json` contract goldens, testscript e2e incl. merge regression matrix, ID fuzz | [09-testing](docs/design/09-testing.md) |

## 6. Verification

- Every milestone gates on named green test suites ([ROADMAP.md](ROADMAP.md)); M3's two-branch merge regression matrix must be green before any multi-user recommendation.
- The `--json` golden files are the frontend contract; nvim work does not start until they are stable (M1 exit).
- Dogfooding: kira tracks its own development in its own `.kira/` from M0 onward; CI runs `kira doctor` against it from M2 (when `doctor` lands).
- Risk table with mitigations lives in the synthesis and is folded per-doc; top three: ID reconciliation bugs (property/fuzz-tested, repairs are visible commits), missing hooks (loud degradation), index staleness (content-hash invalidation + auto-rebuild, cache never authoritative).

## 7. Open questions (founder to ratify)

Defaults below are chosen and specced; each is cheap to flip now, expensive later.

1. **Commit mode default** — `auto` (every mutation = one semantic commit; protects history/telemetry) vs `manual` (state changes ride in your fix commits). Chosen: `auto`.
2. **ID style default** — sequential `KIRA-n` + reconciliation machinery vs zero-machinery hash display (`KIRA-N6T4X2`). Chosen: sequential. Implementation note: hash derivation was underspecified; v1 implements trailing 6 Crockford chars of the ULID (~30 bits — collision-*unlikely*, not impossible, so `doctor` must still check). Pin before hash mode ships.
3. **Label/people strictness** — warn (`strict: false`) vs reject unknown. Chosen: warn.
4. **Comments** — in-body append blocks (greppable single file) vs file-per-comment (maximally conflict-proof). Chosen: in-body.
5. **Filename = bare ULID** — rename-free and reconciliation-safe, but opaque in `ls`. Chosen: bare ULID (browsability comes from `discover`/TUI/nvim).
6. **nvim floor ≥ 0.10** (`vim.system`, zero plugin deps). OK?
7. **Expected scale** (items/contributors/history size) — determines how early the M2 index matters.
8. **Squash-merge shops** — per-mutation kira commits collapse under squash; acceptable, or prefer `manual` default?
9. **Windows** — confirmed untested-not-broken for v1?
10. **Auto-merge policy** — ratify the LWW rules (scalars by `updated` timestamp, tie → theirs; set-merge for lists; whole-body LWW fallback on textual body conflicts). Chosen per your "no merge conflicts, keep it simple" steer; `manual` mode preserved.
11. **nvim plugin is a pure `--json` client** — your original doc had the plugin parsing `.kira`/datamodels directly; reversed so there is one parser/validator (Go), not two drifting (Go + Lua) — see [06](docs/design/06-nvim-plugin.md). Ratify?
12. **Propagation cadence** — tickets ride code branches, so ticket changes made on a feature branch reach teammates at merge cadence; instant JIRA-like propagation means mutating tickets on trunk + `kira sync --push`. A git-bug-style dedicated ticket ref that decouples ticket sync from code review is evaluated-and-deferred to v2 ([07](docs/design/07-git-integration.md)) — acceptable?
13. **`enforce_transitions` default** — the [02](docs/design/02-data-model.md) example sets ticket `true` / epic `false` but no default is documented for workflows that omit the key; YAML map decoding makes omission parse as `false`. Chosen: omission = `false` (mirror the example). Enforcement-on-by-default would need a documented default + tri-state parsing.
14. **`kira init` seed values** — the documented example config carries project-specific values (labels `orderbook`, people `shivam`/`alice`). Chosen: `init` prompts for `project.key` and seeds empty `labels.known`/`people.known`; the doc example stays illustrative, not the literal default.

## 8. Glossary

**item** — ticket or epic (one schema, `type:` discriminates) · **ULID** — immutable identity: filename and the only key used in cross-references · **number** — human display ID `KIRA-n`, reconcilable, aliased forever after renumber · **category** — config-tagged state class (`todo`/`doing`/`done`) that telemetry keys on · **index** — gitignored derived SQLite cache, always rebuildable · **trailer** — git commit trailer `Kira-Ticket: KIRA-n` linking commits to items.
