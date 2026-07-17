# kira

Git-native, terminal-first ticket tracker. Tickets are Markdown files with
YAML frontmatter under `.kira/`, versioned alongside the code they track — no
server, no database. See `README.md` for the full CLI surface.

This repo dogfoods itself: its own tickets live in `.kira/tickets/`, config in
`.kira/config.yaml`. Boards: `KIRA` (default), `CORE`, `TUI`, `CLI`, `DATA`,
`INFRA`, `DX`. Ticket states: `TODO -> IN_PROGRESS -> REVIEW -> DONE` (or
`WONT_DO`), enforced transition graph. `commit.mode: auto` — most `kira`
mutations (create/edit/move/...) auto-commit to `.kira/` immediately; `kira
commit` only exists for `manual`/`prompt` modes.

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

- `datamodel` — domain types (items, config, workflow) and JSON result shapes. Zero deps.
- `id` — two-tier identity: immutable ULIDs + renumberable `KIRA-n` display numbers.
- `codec` — parses/serializes items to/from Markdown-with-frontmatter.
- `storage` — reads/writes item files under `.kira/`, file locking, path resolution.
- `gitx` — git CLI wrapper: commands, staging, branches, trailers, sync.
- `query` — the `list`/filter expression grammar.
- `merge` — field-level three-way auto-merge policy for kira item files.
- `reconcile` — deterministic ID-collision repair plans.
- `index` — disposable SQLite cache under `.kira/.cache/` (staleness detection, incremental refresh).
- `treeish` — loads an item set + config from any git tree-ish (for `diff`/`changes`/merge-driver).
- `hooks` — tracked git-hook shim scripts + their pure logic.
- `automation` — user-defined observer scripts firing post-commit (distinct from `hooks`, which is git-hook territory).
- `workon` — pure branch-naming/slug/active-pointer logic.
- `showfmt` — item-reference string forms shared by TUI/CLI.
- `doctor` — read-only consistency checks (id collisions, dangling refs, schema violations).
- `syncx` — report shape, dirty-tree policy, seams for `sync`.
- `core` — service layer: the one implementation of every mutation and read; everything else composes here.
- `cli` — cobra command tree; every command is a thin argv/flag adapter over `core`.
- `tui` — terminal UI (tree, board, stats screens); `tui/theme` is the single source of terminal color.
- `config` — loads/validates/defaults `.kira/config.yaml`.
- `schema` — generates JSON Schema for `--json` output shapes.
- `seed` — deterministic fixture generation (perf harness, tour/vhs demos).
- Small leaf utilities: `errx` (user-facing errors + suggestions), `ptr`, `rgx`, `fzfx` (fzf picker), `statsfmt`, `termx`, `timex`, `clipx` (OSC 52 clipboard), `editorx`, `testutil` (hermetic git repo test helpers).

`cmd/kira` — the entry point (`os.Exit(cli.Main())`). `schema/kira.json` —
generated (not hand-edited) output-shape schema. `tests/` — `contract`,
`e2e`, `integration`, `nocolor`, `perf`, each its own package.

## CI/CD (`.github/workflows/ci.yml`)

- `build-test`: matrix over `ubuntu-latest`/`macos-latest` — `go build`, `go vet`, `go test -race ./...`.
- `lint`: `golangci-lint-action` pinned to `v2.12.2` in `ci.yml` — install that exact version locally, config lives in `.golangci.yml`.
- `tag` (main only, after build-test+lint): computes and pushes the next `vX.Y.Z` patch tag. Uses the default `GITHUB_TOKEN` — pushes made with that token do **not** trigger other workflow runs, so anything that should fire on the new tag must be a job in this same workflow (not a separate `on: push: tags` workflow), consuming the tag via a job output.
- `release` (needs `tag`): checks out the tag, runs `goreleaser/goreleaser-action@v6` (pinned `v2.5.1`) against `.goreleaser.yaml` — builds `cmd/kira` for linux/darwin × amd64/arm64, `CGO_ENABLED=0` + `-trimpath` (static, reproducible), `-ldflags "-s -w -X internal/cli.version=..."`, publishes `.tar.gz` archives + checksums to GitHub Releases.

`.github/workflows/perf.yml` — non-gating A/B perf trend report (same-runner,
interleaved base/HEAD builds via `benchstat`), never fails the build on ratio,
only on harness failure or cold-start tripwire.

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
