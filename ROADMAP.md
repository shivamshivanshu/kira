# kira — Roadmap

Execution plan for [DESIGN.md](DESIGN.md). Each work package (WP) is sized for delegation to a coding agent: **Context** names the design docs that fully specify it, **Verify** names the evidence that closes it. A milestone ships when its gate is green; later milestones never start on a red gate. Dogfooding rule: from M0 exit onward, kira's own work is tracked in kira's own `.kira/`.

Dependency spine: M0 → M1 → M2 → M3; M4 (TUI) needs M2; M5 (nvim) needs the M1 JSON freeze + M2 (history/linked commits feed the floats); M3 is independent of M4/M5 but gates any multi-user recommendation.

## M0 — File model + core CRUD ✅

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

## M1 — Workflow, relationships, search ✅

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

## M1.5 — JIRA parity (fields + query) ✅

Shippable: grooming, sprints, and Definition-of-Done enforcement — the JIRA-parity gaps ([gap analysis](/home/claude_notes/scratchpad/20260712/kira-jira-gap-analysis.md) Tier 1–3) that close without waiting on M2. All additive to the frozen v1 JSON contract; goldens regenerate with new field cases, no existing shape changes. Sits after M1 (needs the query engine + JSON freeze) and before M2 (index makes the new queries fast but the linear-scan fallback already serves them). Dogfooded as kira epic "JIRA parity" / tickets KIRA-3..KIRA-8.

| WP | Scope | Context | Verify |
|---|---|---|---|
| 1.5.1 (KIRA-3) | Backlog ordering: `rank` field (lexicographic, LWW scalar) + config-ordered `priorities:` enum; default sort precedence rank→priority→number; `create`/`edit` `--rank`/`--priority` | [02](docs/design/02-data-model.md), [04](docs/design/04-cli.md) | unit on sort precedence + ranked-priority compare; goldens: rank/priority round-trip + one-line diff |
| 1.5.2 (KIRA-4) | New optional fields: `subtype` (config enum), `due` (RFC3339), `resolution` (item field), typed `links:` (relates/duplicate_of, single-sided); codec + validation + `--subtype`/`--due` flags + `link --relates`/`--duplicate-of`; `show`/`list` JSON additions | [02](docs/design/02-data-model.md), [04](docs/design/04-cli.md) | byte-stability goldens for each field; validation unit (vocab strict/warn, dangling link ref); contract goldens regenerated |
| 1.5.3 (KIRA-5) | Query grammar: `ORDER BY`, `IN (…)`, `IS EMPTY`/`IS NOT EMPTY`; ranked compare on `priority`, date compare on `due`; `sprint=active` sugar | [04](docs/design/04-cli.md) | parser unit (each production); goldens on result ordering and membership/empty predicates |
| 1.5.4 (KIRA-6) | Workflow enforcement: per-transition `require:` (non-null gate) + `set:` (assignments); per-state `wip:` warn-on-move; `resolution` set-on-done via `move --resolution`/`set:`/state tag, cleared on leaving done | [02](docs/design/02-data-model.md), [04](docs/design/04-cli.md) | unit: transition matrix incl. require-fail/force-override, set-applied, resolution set/clear; testscript: WIP warning on stderr + `warnings` in `--json` |
| 1.5.5 (KIRA-7) | Saved filters: config `filters:` map → `list --filter <name>`, `kira filter list` | [02](docs/design/02-data-model.md), [04](docs/design/04-cli.md) | testscript: named filter expands + ANDs extra flags; unknown-name exit 1 |
| 1.5.6 (KIRA-8) | Sprints + scrum stats: `sprint` field activated, config `sprints:` entity, `.cache/active-sprint` pointer; `kira sprint create\|list\|activate\|close` (+`--move-to` spillover); `stats --sprint` burndown + `--velocity` | [02](docs/design/02-data-model.md), [04](docs/design/04-cli.md), [08](docs/design/08-telemetry.md) | testscript: sprint lifecycle + activate/close spillover; unit: burndown/velocity vs hand-computed fixture (incl. squash-degraded `degraded_n`) |

**Gate:** unit + golden + testscript green; contract goldens regenerated additive-only (no existing key changed); kira's own backlog groomed with `rank`/`priorities` and a live sprint.

## M2 — Index + git integration ✅

Shippable: fast at scale; the git-native value prop (history, linked commits, stats) lands.

| WP | Scope | Context | Verify |
|---|---|---|---|
| 2.1 | SQLite index: schema, staleness check (HEAD SHA + dirty hash), incremental rebuild, ancestor-check → full rescan, auto-rebuild on corruption, `index --full` | [01](docs/design/01-architecture.md) | testscript: delete `.cache/` ⇒ identical results; stale-detection cases |
| 2.2 | Trailer scan: `interpret-trailers` + lenient fallback, per-ref watermarks, commit_links table. **Also (KIRA-9): `Kira-Closes` auto-transition** — a `Kira-Closes: KIRA-n` trailer fires the configured done-transition when the commit reaches the landed ref (default `main`), detected on the same watermarked scan (spec in [07](docs/design/07-git-integration.md)) | [07](docs/design/07-git-integration.md) | testscript fixture repo with trailers, rebase case; Kira-Closes fires once at landed ref, idempotent on re-scan |
| 2.3 | `log`: git-derived field-level history interleaved with linked commits | [03](docs/design/03-storage-and-git.md) | goldens on fixture history |
| 2.4 | `stats`: completion/cycle/lead/throughput, `--json` | [08](docs/design/08-telemetry.md) | unit on formulas vs hand-computed fixture |
| 2.5 | `doctor` (collisions, dangling refs, schema/state violations, freshness, missing hooks/binaries) + `validate` plumbing | [02](docs/design/02-data-model.md), [07](docs/design/07-git-integration.md) | testscript: each violation class detected |
| 2.6 | `kira blame <id>`: latest-event-per-field GROUP BY over WP-2.3's event cache; attribution-honesty markers from day one — per-field `source_kind`/`degraded` (squash/merge, `kira: auto-merged` ran-by, doctor/Kira-Closes synthetics) in the shipping `--json` shape; sequenced after WP-2.3's event shape settles | [03](docs/design/03-storage-and-git.md), [08](docs/design/08-telemetry.md), [STRETCH_GOALS](STRETCH_GOALS.md) | goldens on fixture history incl. squash/auto-merge degraded cases; `blame --json` enters the frozen corpus with its markers |

| 2.7 | Perf-budget report (non-gating): same-job A/B — build merge-base and HEAD in one CI job, run interleaved, benchstat median-of-N (N≥10) **ratio** at 2× so runner speed cancels; deterministic proxies alongside: git-subprocess spawn count per command on the 1k fixture + scaling-ratio check across 100/1k/5k fixtures (generated by the seeder machinery grown with a parametric size knob — shared with tour/vhs, tour spec seeds only ~6); at most one absolute smoke ceiling (cold start < 50ms) as an order-of-magnitude tripwire; checked-in numbers survive only as a non-gating trend artifact; promotion to failing gate is WP-7.7 after observed stability | [01](docs/design/01-architecture.md), [STRETCH_GOALS](STRETCH_GOALS.md) | CI job emits A/B ratio + spawn counts + scaling report on fixtures; job red only on harness failure, never on ratio |

**Gate:** index provably disposable; `log`/`stats` correct on fixture repo; CI dogfood `kira doctor` step added.

## M3 — Distributed correctness ✅

Shippable: safe for teams/branches. Gates any multi-user recommendation.

| WP | Scope | Context | Verify |
|---|---|---|---|
| 3.1 | Hook scripts (tracked) + `hooks install` chaining; post-merge `doctor --fix` ID reconciliation with aliases + visible repair commit | [07](docs/design/07-git-integration.md) | property/fuzz: deterministic, no item lost, ULIDs untouched |
| 3.1.5 | `kira workon <id>`: idempotent per-ticket branch (match existing branches on the configured pattern prefix, switch; else create — never per-slug); `.cache/active` becomes a `{ticket, branch}` pointer honored by `prepare-commit-msg` only on branch match, else branch-pattern inference; slug + hook pattern share one casing config key; alias-aware hook pattern resolution; doing-transition via core (`--no-move`); `--worktree` (per-checkout `.cache` isolates pointers) | [07](docs/design/07-git-integration.md), [STRETCH_GOALS](STRETCH_GOALS.md) | testscript: idempotent re-run, checkout-away produces no mis-trailer, worktree pointer isolation; unit on slug/pattern casing |
| 3.2 | Merge policy engine: field-level three-way auto-merge (LWW scalars by `updated`, set-merge lists, comment union, body text-merge→LWW fallback); `kira merge-file` driver entry point + driver registration in `hooks install`; `resolve` (auto-apply + `--interactive` picker); `merge.policy: auto\|manual` | [03](docs/design/03-storage-and-git.md), [07](docs/design/07-git-integration.md) | property tests: deterministic, losing side always recoverable from parents; sync path and driver path byte-identical |
| 3.2.5 | Shared tree-ish loader + `kira diff <ref>` + `--at` on read verbs (not release-blocking — outside the M3 gate): batched item-set + `config.yaml` load at any tree-ish (`ls-tree` + `cat-file --batch`, one subprocess), ULID-keyed, alias-aware; diff pairs by ULID (default merge-base..ref), reuses WP-3.2's field-class table, renumber/`updated` churn suppressed to one `renumbered X -> Y` line via aliases, body cap (+N/-M); `list`/`show --at` with config from the same tree-ish + ID/alias-skew annotation; mutating verbs reject `--at`; `stats --at` cut from v1; `board --at` lands with WP-4.2 | [03](docs/design/03-storage-and-git.md), [STRETCH_GOALS](STRETCH_GOALS.md) | goldens: diff fixture incl. renumber suppression; testscript: `--at` ref + date forms, mutating verbs reject the flag |
| 3.3 | `sync`: pull --rebase (+dirty handling via commit/stash) → doctor --fix → reindex → report; `--push`/`sync.push` publish side; `--remote` | [07](docs/design/07-git-integration.md) | testscript with remote fixture incl. push path and two-clone round-trip |
| 3.4 | Merge regression matrix: two branches × {different tickets, different fields, same field, concurrent comments, concurrent creates} × {auto, manual} policies × {sync, merge-driver} paths | [09](docs/design/09-testing.md) | matrix green: clean/clean/auto-LWW-reported (manual: surfaced)/clean/renumber+alias |

**Gate:** WP-3.4 matrix green.

## M3.5 — Pre-release CLI polish ✅

Shippable: first-impression CLI quality; all four WPs must land before any public release (errors are the product's real first impression; init output is one-shot per repo, so retrofitting the seed misses existing repos; binaries shipped without the codec fail-safe destroy newer fields forever; dynamic completion rides the `kira completion` scripts).

| WP | Scope | Context | Verify |
|---|---|---|---|
| 3.5.1 | Error contract: three-part stderr errors (what · why · one copy-pasteable next action) via one render helper in `internal/core`, every command's failure path routed through it; in `--json` mode failures emit structured `{error, hint, code}` on **stderr** — stdout stays empty per the [04 §7](docs/design/04-cli.md) ruling; typo-suggestion engine runs on the failure branch only, over vocabulary already in memory; the same change reserves crash exit code 4 in the [04 §1](docs/design/04-cli.md) table (handler lands in WP-4.1 — contract surface reserved before the first tag) | [04](docs/design/04-cli.md) | testscript goldens pin the top-10 first-run errors; `--json` failure case asserts empty stdout |
| 3.5.2 | Self-documenting seed config: `init` writes every unused optional block commented-out with one-line explanation + doc link in `config.yaml`; comment-preserving yaml.v3 `Node`-level `config set` | [02](docs/design/02-data-model.md), [04](docs/design/04-cli.md) | parse-equivalence golden (comments inert) + preservation golden (`config set` changes exactly the target line, keeps all comment blocks) |
| 3.5.3 | Codec unknown-key fail-safe: `Parse` records unknown frontmatter keys; unknown link types downgrade from ParseError to recorded-unknown (reads unblocked); every write path refuses with exit 3 + upgrade hint while unknowns are present — must be in the first tagged binary (pairs with WP-7.3's `min_version` guard) or old binaries silently destroy newer fields forever | [02](docs/design/02-data-model.md), [03](docs/design/03-storage-and-git.md) | property/golden: unknown keys survive parse and are reported; read-with-unknown-link succeeds; write path with unknowns exits 3 |
| 3.5.4 | Dynamic shell completion: cobra `ValidArgsFunction` on every ID/enum position — item IDs with title descriptions, `move` targets from the item's current state via the TUI's adjacency map, `--label`/`--priority`/`--filter` from config vocab; the `__complete` path opens the index **read-only, no staleness check or reindex** (explicit narrow exception to [04](docs/design/04-cli.md)'s implicit-refresh invariant), serves stale over silent-on-stale, never the linear-scan fallback, absent index ⇒ no-suggestions; server-side prefix filter on `toComplete`, ~200-candidate cap, `ShellCompDirectiveNoFileComp` | [04](docs/design/04-cli.md), [STRETCH_GOALS](STRETCH_GOALS.md) | testscript: `__complete` on fixture repo incl. stale-index-serves and absent-index-empty cases; assert no reindex/git subprocess on the completion path |

**Gate:** goldens green; kira's own first-run error paths exercised via testscript.

## M4 — TUI ✅

Shippable: the JIRA-replacement daily driver. Needs M2. If the milestone needs a cut line, crash containment (WP-4.1) and the tour seeder are the two defended items — perf fixtures, vhs tapes, and `kira tour` all hang off the seeder, and the first dogfood panic is when containment pays.

| WP | Scope | Context | Verify |
|---|---|---|---|
| 4.0 | Theme struct: semantic color slots (category/priority/accent/heat), lipgloss AdaptiveColor + `ui.background` escape hatch; the only home for color literals | [05](docs/design/05-tui.md) | goldens under pinned termenv profile+background; no color literal outside the theme package |
| 4.1 | bubbletea app shell + tree/explorer screen (netrw keys, preview pane, nav stack generalized to a bounded jumplist: `Ctrl-o` back / `Ctrl-]` fwd); responsive medium tier (detail pane collapses below the shared min-width constant); epic progress bars in tree rows (cached per refresh); teaching empty state for the zero-item tree; crash containment: bubbletea's default catchPanics path (never reimplemented; v1.3.10 coverage pinned empirically) funneled into main's recover → terminal restore, three fixed lines (one-line panic · crash-log path `.kira/.local/crash-<ts>.log` with `os.TempDir` fallback and print-where-it-went · data-safety statement), exit 4 (reserved in WP-3.5.1); no bare `go func` — one panic-forwarding recover-wrapper mandated for app-spawned goroutines; crash handler itself panic-proof | [05](docs/design/05-tui.md), [STRETCH_GOALS](STRETCH_GOALS.md) | teatest keypress + snapshot; testscript: panic injected **inside a `tea.Cmd`** asserts terminal-restore sequence, crash-log write (incl. tempdir fallback), exit 4 |
| 4.2 | Kanban board (validated H/L moves through core services; `--plain` static render); WIP-limit column tinting (global per-state counts, `n/wip` always shown when configured); teaching empty states for Board (configured column names + hint) and Stats; `board --at <ref>` static render via WP-3.2.5's tree-ish loader | [05](docs/design/05-tui.md) | teatest; board mutation hits same code path as CLI (assert via shared service tests) |
| 4.3 | Ticket detail (glamour body, comments, linked commits → `git show`, history tail) + stats screen (sparklines) + Board detail-peek docked pane (`p`, third mount of the detail renderer, min-width guard) | [05](docs/design/05-tui.md) | teatest snapshots incl. narrow-terminal auto-collapse case |
| 4.4 | `:` command bar (CLI argv grammar), `/` filter, `?` help generated from keymap table | [05](docs/design/05-tui.md) | teatest |
| 4.5 | Icon/visual layer: category→glyph→ASCII table, `ui.icons: auto\|always\|never`, lipgloss theme | [05](docs/design/05-tui.md) | snapshots in both icon modes; alignment preserved in ASCII mode |
| 4.6 | Yank keys: `y` ID, `Y` structured-forms picker; env-aware clipboard chain (OSC 52 always, tmux passthrough; external tool only when `$DISPLAY`/`$WAYLAND_DISPLAY`/darwin supports it); formats via one helper shared with `kira show --format` | [05](docs/design/05-tui.md) | teatest; clipboard-chain matrix across tmux / no-display environments |
| 4.7 | Uninitialized-repo TUI screen: storeless boot mode when `.kira/` is missing on a tty, offering init via explicit typed confirmation that lists exact side effects (creates `.kira/`, makes a commit) through the same core init path; non-tty bare `kira` still exits 3 per the exit-code table | [04](docs/design/04-cli.md), [05](docs/design/05-tui.md) | teatest: uninit screen + typed-confirmation flow; testscript: non-tty invocation keeps exit 3 |

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

Unordered, data-driven: `kira serve` warm-cache process · shell completions polish · Jira import via `--json` · `kira undo` (semantic inverse via core) · visual-mode bulk ops · action palette (`x`) · `ui.theme` variants · staleness/due heat · narrow layout tier · `kira tour` (sandboxed seeded playground; internal seeder may land M4 for demo GIFs) · guided init (`kira init --guided`, contingent on dogfooded first-run config friction) · `kira inbox` (on-sync digest; landed-ref SHA watermark as per-clone user state in gitignored `.kira/.local/`; needs the `people.known` emails identity mapping — 02 amendment) · `kira standup` (author-grouped recap via the `source_kind`-aware event query; strictly after inbox's identity mapping; contingent on dogfooding — may demote to a documented `--json` recipe) · config `aliases:` (CLI-only v1: pre-cobra argv splice, git's rules by reference, `!shell` excluded, shadowing warning; TUI only via a later shared-dispatch hoist) · `list --group-by owner|sprint` (flat-bucket `groups:` JSON reusing the `tree` shape family; `epic`/`state` keys refused with hints to `--tree`/`board --json`). (`Kira-Closes` auto-transition and the sprint entity were promoted to M2.2 / M1.5; release packaging was promoted to M7 — brew tap/nix stay deferred even there.)

## M7 — Publishing

Shippable: kira as an installable public product. Sequenced after M5 (the nvim plugin consumes WP-7.2's handshake) and **before** any M6 stretch promotions — those are post-dogfood/post-release material; numbered M7 only because M6 is the unordered stretch pool. First-tag constraint: WP-7.2/7.3 (with WP-3.5.3) must be aboard the first tagged binary — none of this is retrofittable to already-shipped binaries. v1 install story everywhere (pipeline, README, error hints): curl-binary + `go install` only.

| WP | Scope | Context | Verify |
|---|---|---|---|
| 7.1 | goreleaser pipeline on `v*` tags: linux/darwin × amd64/arm64, `CGO_ENABLED=0` pinned and verified static (`file`/ldd check), archives + `SHA256SUMS`, GitHub artifact attestation, release gated on green CI, ldflags → `internal/version`; smoke step executes each built binary (`kira version` + tiny testscript); install docs for curl-binary + `go install` | [STRETCH_GOALS](STRETCH_GOALS.md) | tag dry-run produces all artifacts; static check + smoke green; release doc states darwin artifacts are cross-compiled untested |
| 7.2 | `kira version` contract handshake: `--json` → `{version, commit, date, json_contract}` from `internal/version`; `json_contract` sourced from the golden-suite constant, never hand-bumped; shape enters the frozen golden corpus in the same change; works with no `.kira/` present | [04](docs/design/04-cli.md), [06](docs/design/06-nvim-plugin.md) | golden corpus includes `version --json`; testscript outside any repo |
| 7.3 | Compat policy doc + `kira.min_version` guard: older binary exits 3 with "repo requires kira ≥ X (you have Y): <install hint>" before any write; light-scan fallback for the one key when the config is otherwise unparseable; `doctor` unknown-key warning; `kira migrate` escape hatch | [03](docs/design/03-storage-and-git.md) | testscript: guard blocks writes, allows reads; unparseable-config fallback case |
| 7.4 | vhs demo tapes: `.tape` scripts under `docs/demo/` against the tour seeder (M4), GIFs published as release assets — never committed to main; regeneration is a required release-checklist step | [STRETCH_GOALS](STRETCH_GOALS.md) | tapes render in the manual release workflow; checklist step documented |
| 7.5 | README as product page: testscript-run quickstart (provisions a local bare origin so `kira sync` is real), install matrix = shipped channels only, principle-level "why not X" table, GIF-less fallback layout; last inside the milestone, after install paths are real | [STRETCH_GOALS](STRETCH_GOALS.md) | quickstart testscript green incl. sync against bare origin; CI link check |
| 7.6 | Remaining topic-help guides (`workflow`/`config`/`hooks`) — parked here by the batch-3 onboarding ruling (`query`/`sync` gate the public release earlier) | [04](docs/design/04-cli.md) | guides render via `kira help <topic>`; listed under `kira help` |
| 7.7 | Perf-gate promotion: WP-2.7's non-gating A/B report flips to build-failing (benchstat ratio > 2× or spawn-count/scaling-proxy regression) only after weeks of observed stability; "verified every commit" trust signal added to README/docs | [STRETCH_GOALS](STRETCH_GOALS.md) | gate red on an injected-regression fixture; stability criterion documented in the release doc |

**Gate:** a tagged release installable via both v1 channels on a clean machine; first-tag constraint items verified aboard.
