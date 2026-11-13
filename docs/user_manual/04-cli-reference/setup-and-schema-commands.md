# Setup and schema commands

Global flags and exit codes, then `init`, `config`, `schema`, `doctor`,
`validate`, `version`, `completion`, `server`.

Samples: a repository `acme`, key `ACME`; `$HOME` is `/home/demo`.

## Global flags

Valid on every command, before or after the subcommand.

| Flag | Meaning |
| --- | --- |
| `-C, --C <path>` | Run as if started in `<path>`; the store is found by walking up from there. |
| `--json` | Machine-readable JSON on stdout. Errors go to stderr as `{"error", "hint", "code"}`; on failure stdout is empty. Warnings stay plain text on stderr. Shapes: [`kira schema`](#kira-schema). Not accepted by bare `kira config`; the TUI and server have none. |
| `--no-color` | No ANSI color. Color is already off when stdout is not a terminal; `NO_COLOR` does the same; otherwise `ui.color` decides. |
| `--non-interactive` (hidden) | Never prompt or open a UI, for scripts and agents: `tui` and `server` refuse; bare `kira` prints help and bare `kira board` the static board; `create` implies `--no-edit`; `edit` without field flags, `comment` without `-m`, and `--from-file -` are errors. |

```console
$ cd / && kira -C /src/acme/src/webhooks list
NUMBER  STATE  TYPE    PRIORITY  TITLE
ACME-1  TODO   ticket  P1        Retry failed webhook deliveries
ACME-2  TODO   ticket  P2        Add idempotency keys
$ cd / && kira list; echo "exit=$?"
kira: no .kira/ found in "/" or any parent directory
  hint: run `kira init` to create one here
exit=3
$ kira config set commit.mode bogus --json; echo "exit=$?"
{"error":"config: commit.mode: invalid value \"bogus\", want one of [auto manual]","hint":"","code":1}
exit=1
```

### Exit codes

Same in human and JSON mode; `code` equals the exit status.

| Code | Meaning |
| --- | --- |
| `0` | success |
| `1` | user error: bad flag or value, failed validation |
| `2` | conflict with existing state |
| `3` | environment: no store, unusable config directory |
| `4` | crash; the hint names a crash log |

## `kira init`

```
kira init [--key KEY] [--force]
```

Creates `.kira/` in the current git repository (it does not run `git init`)
and commits it as `kira: init`. `--key` must match `^[A-Z][A-Z0-9]{1,9}$`.
Without it kira derives one from the directory name (uppercased letters and
digits, at most 10, else `KIRA`) — prompting with it as the default on a
terminal.

```console
$ kira init --key acme
kira: project key "acme" must match ^[A-Z][A-Z0-9]{1,9}$
  hint: keys are 2-10 uppercase letters/digits starting with a letter, e.g. ABC
$ kira init
Initialized kira in .kira (project key ACME)
$ tree -a --noreport -I .git .
.
├── .gitattributes
└── .kira
    ├── config.yaml
    ├── .gitignore
    ├── templates
    └── tickets
```

| Path | Content |
| --- | --- |
| `.kira/config.yaml` | `version: 2`, `project`, one default board, empty `labels`/`people`, `commit`/`merge`/`sync`/`estimate`, the `next` filter, commented optional blocks. |
| `.kira/.gitignore` | `.cache/` |
| `.kira/tickets/`, `.kira/templates/` | empty (not tracked until they hold files); templates are optional, see [`create`](item-lifecycle.md#templates) |
| `.gitattributes` | appends `.kira/** text eol=lf` |

`init` installs no hooks ([`hooks install`](git-integration.md#kira-hooks)).
On an existing store it refuses; `--force` rewrites `config.yaml` (local edits
are lost) and commits it.

```console
$ kira init
kira: .kira already exists
  hint: use `--force` to reinitialize over it
```

## `kira config`

Bare `kira config` prints the user config directory (`$XDG_CONFIG_HOME/kira`,
else `$HOME/.config/kira`) and which files exist there.

```console
$ kira config
/home/demo/.config/kira
  present: none
```

Tiers and keys: [Configuration reference](../05-configuration-reference.md).

### `kira config explain`

The effective configuration as rules, one section per area, each tagged with
the tier that set it: `default`, `repo`, or `user` (for `ui`/`workon`/`commit`
when the user file sets them). The tag is per section. `--json`:
`{"sections": [{"name", "provenance", "lines"}]}`.

```console
$ kira config explain
identity (repo)
  project: ACME (acme)
  board: ACME (acme) [default]

commit (default)
  mode: auto
  subject: "update {count} items" (up to 5 ticket numbers listed, then a count)

vocab (default)
  system label: captured (auto-added by `kira create --here`; excluded from vocabulary checks)
  labels: none (advisory)
  people: none (advisory)
  priorities: P0, P1, P2, P3 (advisory)
  subtypes: bug, task, feature, support (advisory)
  resolutions: done, dropped, duplicate, cannot-reproduce (advisory)

ui (default)
  icons: auto (override with the KIRA_ICONS env var: nerd|emoji|text)
  color: auto
  background: dark

workon (default)
  branch_pattern: {key}/{number}-{slug}
  worktree: false (dir ../{repo}-{branch})

merge (default)
  policy: auto

sync (default)
  push: false

git (default)
  landed_ref: (unset)

estimate (default)
  unit: points

filters (repo)
  next: state = TODO AND NOT blocked ORDER BY priority
```

With a user file pinning `ui.icons: text` and `workon.worktree: true`, plus a
repo-only key it ignores:

```console
$ kira config explain | sed -n "/^ui/,/^$/p;/^workon/,/^$/p"
/home/demo/.config/kira/config.yaml: key "labels" is repo-authoritative; ignored
ui (user)
  icons: text (override with the KIRA_ICONS env var: nerd|emoji|text)
  color: auto
  background: dark

workon (user)
  branch_pattern: {key}/{number}-{slug}
  worktree: true (dir ../{repo}-{branch})
```

### `kira config init`

Writes `config.yaml` into the user config directory with every personal key
commented out at its default; never overwrites.

```console
$ kira config init
Created user config in /home/demo/.config/kira
  config.yaml
$ kira config init
/home/demo/.config/kira already exists; leaving it untouched
  present: config.yaml
```

### `kira config filters`

Lists `filters:` sorted by name; `no filters configured` when empty.

```console
$ kira config filters
next  state = TODO AND NOT blocked ORDER BY priority
```

### `kira config set`

```
kira config set <key> <value>
```

Sets one scalar in `.kira/config.yaml` (never the user file), validated like
the loader, rewriting only that line so comments survive. A key absent from
the file (or only commented out) is appended as a new block. Committed under
`commit.mode: auto`. Setting a key to its current value is a no-op (exit 0,
no commit).

```console
$ kira config set ui.nope true
kira: config: unknown key "ui.nope"; valid keys: commit.mode, estimate.unit, git.landed_ref, merge.policy, project.name, sync.push, ui.background, ui.icons, workon.branch_pattern
$ kira config set commit.mode manual
Set commit.mode = manual
$ grep -A1 "^commit:" .kira/config.yaml
commit:
  mode: manual                  # auto | manual
$ git log --oneline -1
68da7ae config set commit.mode
$ kira config set workon.branch_pattern "{key}-{number}"
Set workon.branch_pattern = {key}-{number}
$ grep -n -A1 "^workon:" .kira/config.yaml
44:workon:
45-  branch_pattern: '{key}-{number}'
```

## `kira schema`

Prints the JSON Schema (draft 2020-12) of every `--json` output: a top-level
`oneOf` over the result shapes, named definitions under `$defs`, and
`additionalProperties: false` throughout. Always JSON; `--json` is a no-op.

```console
$ kira schema | jq ".oneOf | length"
46
$ kira schema | jq -r ".oneOf[].\"\$ref\"" | head -5
#/$defs/CreateResult
#/$defs/ListResult
#/$defs/TreeResult
#/$defs/ShowResult
#/$defs/MutationResult
```

## `kira doctor`

Read-only whole-store check. Finding classes: `id-collision`,
`dangling-ref`, `ref-cycle`, `ref-type`, `schema` (unparsable item, unknown
key), `state`, `vocab`, `comment`, `board`, `wip-limit`, `index-freshness`,
`environment`, `stray-file` (e.g. a leftover `.kira/schema/`). Any error exits
1; `--json` uses the same finding shape as `validate`.

```console
$ kira doctor
info    index-freshness index is up to date
ok: 0 errors, 0 warnings, 1 info
see `kira config explain` for the rules these checks enforce
$ kira doctor; echo "exit=$?"
error   dangling-ref    ACME-1 blocked_by: blocked_by references "01HAAAAAAAAAAAAAAAAAAAAAAA", which resolves to no item
info    index-freshness index is up to date
problems found: 1 error, 0 warnings, 1 info
see `kira config explain` for the rules these checks enforce
kira: doctor: 1 problem found
exit=1
$ kira doctor
warning stray-file      .kira/schema: no longer read by kira; delete it
info    index-freshness index is up to date
ok: 0 errors, 1 warning, 1 info
see `kira config explain` for the rules these checks enforce
```

## `kira validate`

```
kira validate <file|id>...
```

Checks complete item files (paths or item references, mixed) against the
item shape, states and vocabularies, writing nothing. Only errors fail; with
`labels.strict: true` vocabulary findings are errors. Use it in CI or a
pre-commit step; `kira doctor` covers cross-item checks.

```console
$ kira validate .kira/tickets/*.md
ok: 0 errors, 0 warnings, 0 info
see `kira config explain` for the rules these checks enforce
$ kira validate ../import/draft.md; echo "exit=$?"
error   state           ACME-2 state: "BLOCKED" is not a state in the ticket workflow
warning vocab           ACME-2 labels: "security" is not in the known labels vocabulary
warning vocab           ACME-2 priority: "P9" is not in the known priority vocabulary
problems found: 1 error, 2 warnings, 0 info
see `kira config explain` for the rules these checks enforce
kira: validate: 1 problem found
exit=1
$ kira validate ../import/odd.md
error   schema          ACME-2: field "severity": unknown frontmatter key
problems found: 1 error, 0 warnings, 0 info
see `kira config explain` for the rules these checks enforce
kira: validate: 1 problem found
$ kira validate ../import/nope.md
kira: "../import/nope.md" is neither a file nor a resolvable ticket id
```

`--json`: `{"ok", "findings": [{"class", "severity", "path", "item_id",
"number", "field", "message"}], "summary": {"error", "warning", "info"}}`.

## `kira version`

`dev` for a source build, the tag for a release. `--json` adds
`json_contract` (bump = output-shape change), `go`, and `commit` (when built
with VCS info).

```console
$ kira version
dev
```

## `kira completion`

```
kira completion bash|zsh|fish|powershell [--no-descriptions]
```

Prints a completion script, e.g. `source <(kira completion zsh)`; install
paths per shell are in `kira completion <shell> --help`. Completion is
dynamic: item numbers (with titles), the item type's states for `move`,
vocabularies and sprint/filter/board keys for the matching flags, and
`config set` keys. Outside a store it completes nothing.

## `kira server`

```
kira server [--listen 127.0.0.1:7432]
```

Alias `serve`. Serves the web dashboard and prints its URL; Ctrl-C stops it.
`--listen` must be a loopback address; port `0` picks a free one.

```console
$ kira server --listen 0.0.0.0:7432
kira: server: listen address must use localhost or a loopback IP
```

The dashboard has tree, board and stats views with an item detail pane
(body, comments, relationships, history, linked-commit patches), query search
and saved filters, and metrics scoped by epic, sprint, date and weeks. It
creates and edits items, moves states, assigns, links, comments, and bulk-edits
a selection (state, board, owner, labels, sprint; per-item results, failures
kept for retry). Settings manage boards, sprints, supported config keys,
commits and sync. Its command bar runs CLI commands with `.` for the selected
item (`move . IN_PROGRESS`); `sync --remote` there takes only a configured
remote name. Keyboard-driven: Ctrl/Cmd+K for quick actions, `?` for the keys.
Theme (light/dark/system) and compact spacing persist per browser, the
auto-refresh toggle per tab. Saving an item that changed on disk since the
page loaded is refused. **Start work** runs `workon` in the current
checkout, never a new worktree. Every action goes through the same core
operations and guards as the CLI. Terminal-only: `init`, hook
install/uninstall, `config init`, worktree creation, file imports, and
anything that opens an editor. Development: [internal/webui](../../../internal/webui/README.md).
