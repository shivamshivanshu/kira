# kira

A git-native, local-first, terminal-first ticket tracker. Tickets live as files in
`.kira/` inside your repository, version-controlled alongside your code — no server,
no database, no account. Work from the command line or the interactive TUI.

## Quickstart

```sh
kira init                              # create a .kira/ store in the current git repo
kira create ticket --title "Fix flaky merge test"
kira list                              # list tickets and epics
kira board                             # kanban board grouped by workflow state
kira                                   # launch the interactive TUI (default with no command)
```

## Capabilities

Every command supports `--json` for scripting, plus `--no-color` and
`-C <path>` (run as if invoked from elsewhere); `index` also takes `--quiet`.
Bulk-capable commands accept multiple ids at once.

**Item lifecycle** — `create` (ticket/epic, `--from-file`/stdin, `--here` to
capture under the active ticket), `edit` (editor or `--field`, bulk), `move`
(workflow transitions, bulk), `assign`, `label create/list/add/rm`, `link`
(epic parent, blocked-by, typed links), `comment`, `show`.

**Views & organization** — `list` (filterable, saved `--filter` queries),
`tree`, `board` (kanban, WIP limits), `find` (ripgrep-backed full-text search),
`discover` (fzf picker), `now` (active ticket), `stats` (completion, cycle/lead
time, throughput), `log` (field history + linked commits), `blame` (per-field
provenance), `diff`/`changes` (vs. a git ref).

**Sprints & boards** — `sprint create/list/activate/close` (with `--move-to`
spillover), multi-board support (`board create/rename/archive/move/list`,
renumbering keeps the old number as an alias).

**Git integration** — `init`, `hooks install/uninstall/status/run` (pre-commit,
post-merge, prepare-commit-msg, plus merge-driver registration), a custom git
merge driver that auto-resolves same-field conflicts on kira item files,
`resolve` for manual conflict cleanup, `sync` (pull --rebase, reindex, optional
push), `commit`, `doctor` (id collisions, dangling refs, schema violations),
`validate`, `index` (refresh the derived sqlite cache). Commit trailers
(`Kira-Ticket`, `Kira-Closes`) link and auto-close tickets from commit messages.

**Automation** — config-driven hooks fire on `item.created`,
`item.state_changed`, and `sync.completed`, matched by state/type, run with a
timeout, and receive a versioned JSON payload on stdin (plus env-var mirrors).
`automation trust` hashes and trusts the current hook config; `automation
list` shows defined hooks and trust status.

**Configuration** — per-type workflows (states, categories, WIP limits,
transition graphs with `require`/`set` guards), label/people vocab, priorities,
subtypes, resolutions, commit mode (auto/manual/prompt), merge policy, sprints,
saved filters, and a personal overlay (`~/.config/kira/config.yaml` via `config
init`) for editor/UI/worktree preferences that never touches the shared repo
config.

**Two-tier identity** — sequential or hash-derived display numbers (`KIRA-42`)
for humans, ULIDs for stable identity across renumbering.

**TUI** — launches by default with no arguments. Tree, board, and stats views
(switch with `1`/`2`/`3`) plus a detail pane; `^o`/`^]` jump, `:` command mode,
`/` filter, `y`/`Y` yank, `r` refresh, `?` help, `q` quit.

## Documentation

- `kira --help` and `kira <command> --help` — the command surface.
- `kira schema` / [schema/kira.json](schema/kira.json) — published JSON schema for every `--json` output and the automation hook-stdin payload.
- `kira version` — build version.
- `.kira/config.yaml` — the annotated project configuration; `kira config init` scaffolds personal preferences.
