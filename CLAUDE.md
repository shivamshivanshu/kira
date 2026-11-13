# kira

Git-native, terminal-first ticket tracker. Tickets are Markdown files with
YAML frontmatter under `.kira/`, versioned alongside the code they track — no
server, no database. See `README.md` for the full CLI surface.

This repo dogfoods itself: its own tickets live in `.kira/tickets/`, config in
`.kira/config.yaml`. Boards: `KIRA` (default), `CORE`, `TUI`, `CLI`, `DATA`,
`INFRA`, `DX`. Ticket states: `TODO`, `IN_PROGRESS`, `REVIEW`, `DONE`,
`WONT_DO` — a plain enum, any state to any state. `commit.mode: auto` — most `kira`
mutations (create/edit/move/...) auto-commit to `.kira/` immediately; `kira
commit` only exists for `manual` mode.

## Code style: names carry the meaning, not comments

Do not write comments that a good name would make redundant. Long, explicit
identifiers are preferred over short ones plus an explanatory comment — rename
the thing, extract a named function, or introduce a named intermediate value
instead of narrating it. `blockedByTargetTypeMismatch` beats `mismatch` plus a
comment saying what kind of mismatch it is.

A comment is justified only when it carries something no name can:

- a non-obvious *why* (a workaround, a subtle invariant, why the obvious
  approach fails here — see the Gotchas below for the class of thing worth
  writing down),
- a genuinely complex algorithm whose steps do not read off the code.

Never write comments that restate the code, label sections, or repeat the
function name as a sentence. Do not add a doc comment to an unexported
function just because it exists. When editing existing code, do not delete
comments that predate your change — comment cleanup is its own intentional
task, not a side effect.

## Build, test, lint

```sh
go build ./...
go vet ./...
go test -race ./...              # unit + internal/*/test + tests/{contract,e2e,integration,nocolor,perf}
golangci-lint run ./...          # pinned to v2.12.2 in CI — install/verify that exact version locally
```

`tests/e2e` is slow (~4-6 min); everything else is fast. `KIRA_PERF=1` gates
`tests/perf` (also run standalone, non-gating, in `.github/workflows/perf.yml`).

## Package map (`internal/`)

Dependency order, low to high (each layer only depends on layers above it —
see `REVIEW.md` for the full bottom-up file list):

- `datamodel` — domain types (items, config, workflow) and JSON result shapes. Depends only on the `errx`/`ptr`/`setx`/`timex` leaves.
- `id` — two-tier identity: immutable ULIDs + renumberable `KIRA-n` display numbers.
- `codec` — parses/serializes items to/from Markdown-with-frontmatter.
- `storage` — reads/writes item files under `.kira/`, file locking, path resolution.
- `gitx` — git CLI wrapper: commands, staging, branches, trailers, sync.
- `query` — the `list`/filter expression grammar.
- `config` — loads/validates/defaults `.kira/config.yaml` (uses `query` and `storage`).
- `merge` — field-level three-way auto-merge policy for kira item files.
- `reconcile` — deterministic ID-collision repair plans.
- `index` — disposable SQLite cache under `.kira/.cache/` (staleness detection, incremental refresh).
- `treeish` — loads an item set + config from any git tree-ish (for `diff`/`changes`/merge-driver).
- `hooks` — embedded git-hook shim scripts + their pure logic.
- `syncx` — report shape, dirty-tree policy, seams for `sync`.
- `workon` — pure branch-naming/slug/active-pointer logic.
- `showfmt` — item-reference string forms shared by TUI/CLI.
- `warningfmt` — renders `datamodel.Warning` codes as user-facing text shared by CLI/webui.
- `doctor` — read-only consistency checks (id collisions, dangling refs, schema violations).
- `core` — service layer: the one implementation of every mutation and read; everything else composes here.
- `schema` — generates JSON Schema for `--json` output shapes.
- `tui` — terminal UI (tree, board, stats screens); `tui/theme` is the single source of terminal color.
- `webui` — embedded HTTP web dashboard served by `kira server`; mutations go through `core`.
- `cli` — cobra command tree; every command is a thin argv/flag adapter over `core` (launches `tui`/`webui`).
- `seed` — deterministic fixture generation (perf harness, tour/vhs demos).
- Small leaf utilities: `errx` (user-facing errors + suggestions), `ptr`, `statsfmt`, `termx`, `timex`, `clipx` (OSC 52 clipboard), `editorx`, `commandline` (shell-style argv splitting), `testutil` (hermetic git repo test helpers), `setx` (generic dedup/set helpers, zero deps).

`cmd/kira` — the entry point (`os.Exit(cli.Main())`). `tests/` — `contract`,
`e2e`, `integration`, `nocolor`, `perf`, each its own package.

## CI/CD (`.github/workflows/ci.yml`)

- `build-test`: matrix over `ubuntu-latest`/`macos-latest` — `go build`, `go vet`, `go test -race ./...`.
- `lint`: `golangci-lint-action` pinned to `v2.12.2` in `ci.yml` — install that exact version locally, config lives in `.golangci.yml`.
- `tag` (main only, after build-test+lint): computes and pushes the next `vX.Y.Z` patch tag. Uses the default `GITHUB_TOKEN` — pushes made with that token do **not** trigger other workflow runs, so anything that should fire on the new tag must be a job in this same workflow (not a separate `on: push: tags` workflow), consuming the tag via a job output.
- `release` (needs `tag`): checks out the tag, runs `goreleaser/goreleaser-action@v6` (pinned `v2.5.1`) against `.goreleaser.yaml` — builds `cmd/kira` for linux/darwin × amd64/arm64, `CGO_ENABLED=0` + `-trimpath` (static, reproducible), `-ldflags "-s -w -X internal/cli.version=..."`, publishes `.tar.gz` archives + checksums to GitHub Releases.

`.github/workflows/perf.yml` — non-gating perf check: runs the deterministic
`tests/perf` proxies (spawn counts, scaling) against HEAD and fails only on
harness failure or the cold-start tripwire.

## Gotchas hit in practice

- Go has no `-O2`/`-O3` — `gc` always builds at one fixed, always-on
  optimization level. `-gcflags="-N -l"` only *disables* optimization/inlining
  for debugger builds.
- Timestamps: don't render via `time.Time.Local()` — it depends on the host
  machine's OS timezone, which differs between a dev machine and CI runners
  (breaks golden tests non-deterministically). Format using the timestamp's
  own recorded offset instead.
- Any path handed to `git` (pathspecs, `filepath.Rel` against `git rev-parse
  --show-toplevel`) must go through `filepath.EvalSymlinks` first if it might
  not already be resolved — `--show-toplevel` always resolves symlinks (e.g.
  macOS's `/var` -> `/private/var`), so an unresolved path on the other side
  of a `filepath.Rel` call silently produces a pathspec that climbs outside
  the repo.
- Time-relative rendered output (overdue glyphs, `now`'s "in state" age) must
  read the clock through `timex.Now()`, never `time.Now()` directly — tests
  pin it deterministically via the `KIRA_NOW` env var, the same class of bug
  as the timezone gotcha above (real wall-clock drift breaking golden tests).
