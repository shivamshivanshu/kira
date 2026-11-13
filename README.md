# kira

A git-native, local-first, terminal-first ticket tracker. Tickets live as files in
`.kira/` inside your repository, version-controlled alongside your code — no hosted
service, no database, no account. Work from the command line, interactive TUI, or
local web dashboard.

## Quickstart

```sh
kira init                              # create a .kira/ store in the current git repo
kira create ticket --title "Fix flaky merge test"
kira list                              # list tickets and epics
kira board                             # kanban board grouped by workflow state
kira                                   # launch the interactive TUI (default with no command)
kira server                            # serve the dashboard at http://127.0.0.1:7432
```

Open the URL printed by `kira server` and press Ctrl-C in the terminal to stop it.
Use `kira -C /path/to/repo server` to select a repository, or
`kira server --listen 127.0.0.1:8080` to choose another port. `kira serve` is an
alias; `--listen 127.0.0.1:0` chooses an available port. When working from source,
run `go run ./cmd/kira server`.

## Capabilities

Most commands support `--json` for scripting; the interactive TUI and web
dashboard do not emit JSON. Global flags include `--no-color` and `-C <path>`
(run as if invoked from elsewhere); `index` also takes `--quiet`.
Bulk-capable commands accept multiple ids at once.

**Item lifecycle** — `create` (ticket/epic, `--from-file`/stdin, `--here` to
capture under the active ticket), `edit` (editor or `--field`, bulk), `move`
(workflow transitions, bulk), `assign`, `label create/list/add/rm`, `link`
(epic parent, blocked-by, typed links), `comment`, `show`.

**Views & organization** — `list` (native filter flags, `--query` expressions, saved `--filter` queries),
`tree`, `board` (kanban, WIP limits), `find` (regex full-text search),
`now` (active ticket), `stats` (completion, cycle/lead time, throughput), `log` (field history + linked commits), `blame` (per-field
provenance), `diff`/`changes` (vs. a git ref).

**Sprints & boards** — `sprint create/list/activate/close` (with `--move-to`
spillover), multi-board support (`board create/rename/archive/unarchive/move/list`,
renumbering keeps the old number as an alias).

**Git integration** — `init`, `hooks install/uninstall/status/run` (pre-commit,
post-merge, prepare-commit-msg, plus merge-driver registration), a custom git
merge driver that auto-resolves same-field conflicts on kira item files,
`resolve` for manual conflict cleanup, `sync` (pull --rebase, reindex, optional
push), `commit`, `doctor` (id collisions, dangling refs, schema violations),
`validate`, `index` (refresh the derived sqlite cache). Commit trailers
(`Kira-Ticket`, `Kira-Closes`) link and auto-close tickets from commit messages.

**Configuration** — label/people vocab, boards, commit mode
(auto/manual), merge policy, sprints, saved filters, and a personal
overlay (`~/.config/kira/config.yaml` via `config init`) for editor/UI/worktree
preferences that never touches the shared repo config.

**Two-tier identity** — sequential display numbers (`KIRA-42`)
for humans, ULIDs for stable identity across renumbering.

**TUI** — launches by default with no arguments. Tree, board, and stats views
(switch with `1`/`2`/`3`) plus a detail pane; `^o`/`^]` jump, `:` command mode,
`/` filter, `y`/`Y` yank, `r` refresh, `?` help, `q` quit.

**Web dashboard** — `server` serves tree, kanban board, and stats views with an
item detail pane, query search, saved filters, and metrics scoped by epic, sprint,
date, and week count. Inspect descriptions, comments, relationships, history, and
patches for linked or referenced commits. Keyboard navigation, collapsible groups,
an item picker, reference copying, and optional auto refresh support larger boards.

Choose Light, Dark, or System appearance, comfortable or compact spacing, and
collapse the sidebar for more room. Display preferences persist in the browser.
Ctrl/Cmd+K opens searchable quick actions. Subtle transitions respect the system's
reduced-motion preference. State, owner, priority, and label dropdowns combine
with saved filters and Kira's SQL-like query language; the query help includes
one-click examples without exposing raw SQL or database access.

Create and edit items, move workflow states, assign owners, manage relationships,
and add comments through the same core operations as the CLI. Select multiple
items to move states or boards, assign owners, add/remove labels, or set sprints;
results report each item separately and retain failed selections for retry.
Settings provides board and sprint management, supported project configuration
edits, commits, and sync. **Start work** switches the current checkout to the item
branch and marks it active; it never creates another worktree. Workflow guards,
commit policy, and stale-edit checks apply to browser edits.

The command bar also runs supported CLI commands, including `config set`, `sync`,
and `workon`; use `.` for the selected item, for example `show .` or
`move . IN_PROGRESS`. Explicit sync remotes must already be configured by name.
The server binds only to localhost and all actions target the selected checkout.
Store initialization, hook installation/removal, personal config
initialization, creating worktrees, file-based imports, and interactive editor
commands remain terminal-only. The Go binary embeds the dashboard; no separate
frontend build or runtime is needed.

Dashboard state tests run during `go test` when Node is available. To run them
directly: `node --test internal/webui/testdata/*_test.mjs`. See the
[web UI development guide](internal/webui/README.md) for structure and extension
points.

## Documentation

- **[docs/user_manual/](docs/user_manual/README.md)** — the full user manual: getting started, concepts, a complete CLI reference, configuration reference, the TUI guide, recipes, and current limitations.
- `kira --help` and `kira <command> --help` — the command surface.
- `kira schema` — JSON schema for every `--json` output.
- `kira version` — build version.
- `.kira/config.yaml` — the annotated project configuration; `kira config init` scaffolds personal preferences.
