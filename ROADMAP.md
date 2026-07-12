# kira — Roadmap

Execution plan for [DESIGN.md](DESIGN.md). Each work package (WP) is sized for delegation to a coding agent: **Context** names the design docs that fully specify it, **Verify** names the evidence that closes it. A milestone ships when its gate is green; later milestones never start on a red gate. Dogfooding rule: from M0 exit onward, kira's own work is tracked in kira's own `.kira/`.

Dependency spine: M0 → M1 → M2 → M3; M4 (TUI) needs M2; M5 (nvim) needs the M1 JSON freeze + M2 (history/linked commits feed the floats); M3 is independent of M4/M5 but gates any multi-user recommendation.

## M0 — File model + core CRUD

Shippable: a usable, greppable ticket store. No index, no TUI.

| WP | Scope | Context | Verify |
|---|---|---|---|
| 0.1 | Go module scaffold: cobra skeleton, package layout, golangci-lint, GitHub Actions (linux+macOS, race+vet+lint) | [01](docs/design/01-architecture.md) | CI green on empty command tree |
| 0.2 | Config schema + loader: parse/validate `config.yaml`, defaults, workflow/category model | [02](docs/design/02-data-model.md) | unit tests incl. invalid-config rejection |
| 0.3 | Frontmatter codec: Item parse/serialize, block YAML, fixed key order, in-place line rewrite | [02](docs/design/02-data-model.md), [03](docs/design/03-storage-and-git.md) | byte-stability goldens: one field edit ⇒ one-line diff |
| 0.4 | Identity: ULID mint, `KIRA-n` allocation (max-visible+1), ID resolution (ULID \| prefix \| number \| alias) | [02](docs/design/02-data-model.md) | unit + property tests on allocation/resolution |
| 0.5 | `kira init` (+`.gitignore`, templates, initial commit) and `hooks install` scaffold (scripts land in M3) | [03](docs/design/03-storage-and-git.md), [04](docs/design/04-cli.md) | testscript: init in fresh repo |
| 0.6 | `create ticket|epic` / `show` / `edit` / `list` (linear scan): `$EDITOR` flow with template + validate-retry, `--no-edit`, `--from-file`; commit modes auto/manual/prompt, atomic writes, advisory lock | [03](docs/design/03-storage-and-git.md), [04](docs/design/04-cli.md) | testscript CRUD suite with `EDITOR=true`; auto-mode produces one structured commit per mutation |

**Gate:** unit + golden + testscript green; kira initialized on itself.

## M1 — Workflow, relationships, search

Shippable: a real solo tracker with process discipline.

| WP | Scope | Context | Verify |
|---|---|---|---|
| 1.1 | State machine: transition validation, `move` (+`--force`, `--activate` pointer) | [02](docs/design/02-data-model.md), [04](docs/design/04-cli.md) | unit: full adjacency matrix; testscript |
| 1.2 | `assign`, `link` (epic/blocked-by, `--remove`), `comment` (anchored append blocks) | [02](docs/design/02-data-model.md) | goldens: comment append is pure suffix; link edits touch one file only |
| 1.3 | Label/people vocab enforcement (strict/warn, `--force`) | [02](docs/design/02-data-model.md) | unit |
| 1.4 | Query engine + expression grammar; `list` filters; `tree` render | [04](docs/design/04-cli.md) | unit on parser; goldens on results |
| 1.5 | `find` (rg wrapper, LookPath, pure-Go fallback), `discover` (fzf, bubbles fallback) | [04](docs/design/04-cli.md) | testscript with and without rg/fzf on PATH |
| 1.6 | `--json` on every command + golden contract tests; exit-code policy; stderr/stdout discipline | [04](docs/design/04-cli.md), [09](docs/design/09-testing.md) | golden suite; contract declared frozen |

**Gate:** JSON contract frozen (additive-only after this point) — precondition for all frontend work.

## M2 — Index + git integration

Shippable: fast at scale; the git-native value prop (history, linked commits, stats) lands.

| WP | Scope | Context | Verify |
|---|---|---|---|
| 2.1 | SQLite index: schema, staleness check (HEAD SHA + dirty hash), incremental rebuild, ancestor-check → full rescan, auto-rebuild on corruption, `index --full` | [01](docs/design/01-architecture.md) | testscript: delete `.cache/` ⇒ identical results; stale-detection cases |
| 2.2 | Trailer scan: `interpret-trailers` + lenient fallback, per-ref watermarks, commit_links table | [07](docs/design/07-git-integration.md) | testscript fixture repo with trailers, rebase case |
| 2.3 | `log`: git-derived field-level history interleaved with linked commits | [03](docs/design/03-storage-and-git.md) | goldens on fixture history |
| 2.4 | `stats`: completion/cycle/lead/throughput, `--json` | [08](docs/design/08-telemetry.md) | unit on formulas vs hand-computed fixture |
| 2.5 | `doctor` (collisions, dangling refs, schema/state violations, freshness, missing hooks/binaries) + `validate` plumbing | [02](docs/design/02-data-model.md), [07](docs/design/07-git-integration.md) | testscript: each violation class detected |

**Gate:** index provably disposable; `log`/`stats` correct on fixture repo; CI dogfood `kira doctor` step added.

## M3 — Distributed correctness

Shippable: safe for teams/branches. Gates any multi-user recommendation.

| WP | Scope | Context | Verify |
|---|---|---|---|
| 3.1 | Hook scripts (tracked) + `hooks install` chaining; post-merge `doctor --fix` ID reconciliation with aliases + visible repair commit | [07](docs/design/07-git-integration.md) | property/fuzz: deterministic, no item lost, ULIDs untouched |
| 3.2 | Merge policy engine: field-level three-way auto-merge (LWW scalars by `updated`, set-merge lists, comment union, body text-merge→LWW fallback); `kira merge-file` driver entry point + driver registration in `hooks install`; `resolve` (auto-apply + `--interactive` picker); `merge.policy: auto\|manual` | [03](docs/design/03-storage-and-git.md), [07](docs/design/07-git-integration.md) | property tests: deterministic, losing side always recoverable from parents; sync path and driver path byte-identical |
| 3.3 | `sync`: pull --rebase (+dirty handling via commit/stash) → doctor --fix → reindex → report; `--push`/`sync.push` publish side; `--remote` | [07](docs/design/07-git-integration.md) | testscript with remote fixture incl. push path and two-clone round-trip |
| 3.4 | Merge regression matrix: two branches × {different tickets, different fields, same field, concurrent comments, concurrent creates} × {auto, manual} policies × {sync, merge-driver} paths | [09](docs/design/09-testing.md) | matrix green: clean/clean/auto-LWW-reported (manual: surfaced)/clean/renumber+alias |

**Gate:** WP-3.4 matrix green.

## M4 — TUI

Shippable: the JIRA-replacement daily driver. Needs M2.

| WP | Scope | Context | Verify |
|---|---|---|---|
| 4.1 | bubbletea app shell + tree/explorer screen (netrw keys, preview pane, nav stack) | [05](docs/design/05-tui.md) | teatest keypress + snapshot |
| 4.2 | Kanban board (validated H/L moves through core services; `--plain` static render) | [05](docs/design/05-tui.md) | teatest; board mutation hits same code path as CLI (assert via shared service tests) |
| 4.3 | Ticket detail (glamour body, comments, linked commits → `git show`, history tail) + stats screen (sparklines) | [05](docs/design/05-tui.md) | teatest snapshots |
| 4.4 | `:` command bar (CLI argv grammar), `/` filter, `?` help generated from keymap table | [05](docs/design/05-tui.md) | teatest |
| 4.5 | Icon/visual layer: category→glyph→ASCII table, `ui.icons: auto\|always\|never`, lipgloss theme | [05](docs/design/05-tui.md) | snapshots in both icon modes; alignment preserved in ASCII mode |

**Gate:** teatest suite green incl. nvim `:terminal` smoke run.

## M5 — nvim plugin

Shippable: the full founder vision. Needs M1 JSON freeze + M2.

| WP | Scope | Context | Verify |
|---|---|---|---|
| 5.1 | `cli.lua` async `vim.system` wrapper, error UX, `health.lua`, version handshake | [06](docs/design/06-nvim-plugin.md) | plenary busted vs fixture repo |
| 5.2 | `gk` ambient jump: token scanner, floating view stack, `gp` parent, linked commits → `git show`/diffview | [06](docs/design/06-nvim-plugin.md) | plenary |
| 5.3 | `:KiraCreate`/`:KiraEdit` scratch buffers via `--from-file`, validation diagnostics | [06](docs/design/06-nvim-plugin.md) | plenary incl. invalid-input round-trip |
| 5.4 | `:KiraDiscover`/`:KiraQuery` pickers (telescope, `vim.ui.select` fallback) | [06](docs/design/06-nvim-plugin.md) | plenary both paths |
| 5.5 | Visuals: mini.hipatterns integration + extmark fallback (states from config, never hardcoded), nvim-web-devicons, virtual-text state hints, highlight groups | [06](docs/design/06-nvim-plugin.md) | plenary with/without mini.nvim + devicons installed |
| 5.6 | `:KiraBoard` (`:terminal kira board`, `--plain` fallback) | [06](docs/design/06-nvim-plugin.md) | manual smoke + plenary arg test |

**Gate:** plenary suite green with and without optional deps installed.

## M6 — Stretch / polish

Unordered, data-driven: `kira serve` warm-cache process · `Kira-Closes` auto-transition · sprint entity · shell completions polish · release packaging (static binaries, homebrew tap) · Jira import via `--json`.
