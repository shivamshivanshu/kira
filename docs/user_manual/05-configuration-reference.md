# Configuration Reference

Every key of `.kira/config.yaml` (repo) and `~/.config/kira/config.yaml`
(user). Commands that read and write config — `config explain|init|filters|set`
— are in the [CLI reference](04-cli-reference/setup-and-schema-commands.md#kira-config).

## Two config layers

| | Repo | User |
| --- | --- | --- |
| Path | `.kira/config.yaml` | `$XDG_CONFIG_HOME/kira/config.yaml`, else `~/.config/kira/config.yaml` |
| Committed | yes | no |
| Keys | all | `ui`, `workon`, `commit.subject` |
| Collision | **wins** | loses |

Any other top-level key in the user file is ignored with a stderr warning per
key (the command continues):

```
~/.config/kira/config.yaml: key "labels" is repo-authoritative; ignored
```

Exception: `ui.editor` is read **only** from the user file; in the repo file
it is ignored with
`.kira/config.yaml: ui.editor is personal; set it in ~/.config/kira/config.yaml; ignored`.

Keys kira no longer reads (`workflows`, `priorities`, `subtypes`,
`resolutions`, `resolutions_dropped`, `fields`, `automation`, `id`,
`sync.dirty`, `workon.casing`, `commit.trailer` and the other removed
`commit.*` link keys) are silently ignored.

### Errors and warnings

An invalid value in the repo file fails every command:

```
kira: config: ui.icons: invalid value "bogus", want one of [auto nerd emoji text]
```

In the user file the section holding it is ignored with a warning:
`~/.config/kira/config.yaml: config: ui.icons: invalid value "bogus", want one of [auto nerd emoji text]; ignored`.

These only warn and drop the value: an unknown `ui.list.columns` entry, an
unknown or non-hex `ui.theme` slot, `ui.tui.split` outside `(0,1)`, a
multi-line `commit.subject` in the user file.

## `version`

```yaml
version: 2
```

Config format version. `kira init` writes `2`. A `version: 1` repo config no
longer loads (historical trees still read for `--at`, `diff`, `changes`):

```
kira: config: version 1 is no longer supported
  hint: set version: 2 and add boards: [{ key: KIRA, name: config, default: true }]
```

Any other version: `version: unsupported version 3 (this kira understands 1..2)`.

## `project`

| Key | Type | Notes |
| --- | --- | --- |
| `key` | string | Key of the initial board, written by `kira init`. |
| `name` | string | Display name; `kira config set project.name`. |

## Item types and vocabularies

Not configurable: types, states, WIP limits, and the `priority`/`subtype`/
`resolution` lists are built in ([Concepts](03-concepts.md#workflow-states-and-categories)).
An unknown type is refused:

```
kira: "bug" is not a known type
  hint: known types: epic, ticket
```

## `labels`, `people`

```yaml
labels:
  known: [perf, backend]
  strict: false
people:
  known:
    - jdoe
    - { name: asmith, git: ["Alice Smith", "alice@example.com"] }
  strict: false
```

| Key | Type | Default |
| --- | --- | --- |
| `known` | list | `[]` |
| `strict` | bool | `false` |

A person is a name or `{name, git: [aliases]}`; a git `user.name` or
`user.email` listed in `git` resolves to that name (for `@me` and comment
authors). `labels.strict` also governs the built-in
vocabularies. Semantics: [Concepts](03-concepts.md#vocabularies).
`kira label create` appends to `labels.known`.

## `filters`

```yaml
filters:
  next: state = TODO AND NOT blocked ORDER BY priority
```

Map name → [query expression](04-cli-reference/views-and-queries.md#query-expressions),
run with `list --filter`, `board --filter`, or as TUI board chips. Parsed at
load, so a bad query fails every command:

```
kira: config: filters.broken: query: expected a value after = at position 8
```

## `boards`

```yaml
boards:
  - { key: KIRA, name: kira, default: true }
  - { key: CORE, name: Core store and domain, description: "Storage layer" }
```

| Key | Type | Required |
| --- | --- | --- |
| `key` | `^[A-Z][A-Z0-9]{1,9}$`, unique case-insensitively | yes |
| `name` | non-empty string | yes |
| `description` | string | no |
| `default` | bool, at most one board, not an archived one | no |
| `archived` | bool | no |

At least one board is required:

```
kira: config: boards: at least one board is required
  hint: add boards: [{ key: KIRA, name: config, default: true }]
```

Managed with [`kira board`](04-cli-reference/boards-and-sprints.md).

## `sprints`

```yaml
sprints:
  - { key: 2026-S46, name: Docs push, start: "2026-11-09", end: "2026-11-20" }
```

All four keys required; unique keys, non-empty names, `YYYY-MM-DD` dates,
`start` before `end`. Normally written by `kira sprint create`.

## `commit`

| Key | Type | Default | Tier |
| --- | --- | --- | --- |
| `mode` | `auto` \| `manual` | `auto` | repo |
| `subject` | single-line string | `update {count} items` | user |

`auto` commits each mutation immediately; `manual` stages only, for
[`kira commit`](04-cli-reference/git-integration.md#kira-commit). `subject`
replaces the subject of multi-item `kira commit` commits. Trailers and link
markers are fixed:
[Commit linking](04-cli-reference/git-integration.md#commit-linking).

## `merge`

| Key | Type | Default |
| --- | --- | --- |
| `policy` | `auto` \| `manual` | `auto` |

`auto`: the merge driver and `kira sync` merge item files per field and
arbitrate same-field edits. `manual`: item conflicts get ordinary markers and
`kira sync` aborts its rebase on any conflict.

## `sync`

| Key | Type | Default |
| --- | --- | --- |
| `push` | bool | `false` |

`true` makes `kira sync` push after a clean run.

## `workon`

| Key | Type | Default |
| --- | --- | --- |
| `branch_pattern` | template; tokens `{key}` `{number}` `{slug}` | `{key}/{number}-{slug}` |
| `worktree` | bool | `false` |
| `worktree_dir` | template; tokens `{repo}` `{branch}` `{key}` `{number}` | `../{repo}-{branch}` |

Rendered lowercase with `-` separators: `KIRA-1 Report an unresolvable ref
once` → `kira/kira-1-report-an-unresolvable-ref-once`. The pattern must
contain `{number}`. `worktree: true` makes `kira workon` default to
`--worktree`. `worktree_dir` is relative to the repo or absolute; `~` is
rejected (not expanded). Usually set in the user file.

```
kira: config: workon.branch_pattern: must contain {number}, got "{key}/{slug}"
kira: config: workon.worktree_dir: "~/wt/{branch}" begins with ~, which is not expanded
  hint: use an absolute or repo-relative path
```

## `ui`

Personal; prefer the user file.

| Key | Type | Default |
| --- | --- | --- |
| `icons` | `auto` \| `nerd` \| `emoji` \| `text` | `auto` |
| `background` | `dark` \| `light` | `dark` |
| `color` | `auto` \| `always` \| `never` | `auto` |
| `editor` | shell command (user file only) | empty: `$VISUAL`, then `$EDITOR` |
| `auto_tui` | bool | `true` |
| `list.columns` | list | `[number, state, type, priority, title]` |
| `tui.split` | float in `(0,1)` | `0.5` |
| `tui.refresh` | duration, `"0"` or ≥ `1s` | `"0"` (off) |
| `theme.<slot>` | `#RGB` / `#RRGGBB` | built-in palette |

- `icons`: glyph set; `auto` probes the terminal. Boxes or `?` mean no Nerd
  Font — use `emoji` or `text`. `KIRA_ICONS` overrides per invocation.
- `background`: palette the TUI and board assume.
- `color`: `auto` colors only on a terminal; `--no-color`/`NO_COLOR` win.
- `editor`: run via `sh -c`, so flags and quoting work (`code --wait`).
- `auto_tui`: bare `kira` and bare `kira board` open the TUI on a terminal;
  `false` makes them print help and the static board. Never under `--json` or
  [`--non-interactive`](04-cli-reference/setup-and-schema-commands.md#global-flags).
- `list.columns`: `number`, `title`, `type`, `subtype`, `state`, `category`,
  `priority`, `owner`, `labels`, `parent`, `parent_number`, `resolution`,
  `due`, `id`. Unknown names warn and are dropped:
  `kira: warning: ui.list.columns: unknown column "assignee"; ignored`.
- `tui.split`: tree pane width fraction. `tui.refresh`: TUI re-read interval;
  `500ms` fails with `ui.tui.refresh: 500ms is below the 1s minimum`.
- `theme` slots: `accent` (selection, focus), `dim` (secondary text), `border`,
  `todo`/`doing`/`done` (categories), `warm` (near a WIP limit, due soon),
  `hot` (over a limit, overdue).

## `git`

| Key | Type | Default |
| --- | --- | --- |
| `landed_ref` | git ref | `origin/HEAD`'s target, else `main` |

The ref whose history means "landed": `Kira-Closes` trailers close items only
once their commit is reachable from it
([`index --closes`](04-cli-reference/git-integration.md#--closes)).

## `estimate`

| Key | Type | Default |
| --- | --- | --- |
| `unit` | `points` \| `hours` | `points` |

Label for the `estimate` field; no conversion.

## Scaffold

What `kira init --key KIRA` writes (here in a directory named `config`):

```yaml
version: 2

project:
  key: KIRA
  name: config

boards:
  - { key: KIRA, name: config, default: true }

labels:
  known: []                   # add project labels; strict rejects unknown ones (--force overrides)
  # known: [bug, feature, perf, tech-debt, infra]   # example vocabulary
  strict: false

people:
  known: []                   # add known people; strict rejects unknown assignees
  # known: [{ name: jdoe }, { name: asmith }]        # example roster
  strict: false

commit:
  mode: auto                  # auto | manual

merge:
  policy: auto                # auto | manual: auto-resolve same-field kira conflicts vs leave markers

sync:
  push: false                 # true = kira sync publishes after a clean pull; default is pull-only

# workon: branch and worktree naming (built-in defaults shown; uncomment a key to pin it for this repo)
#   branch_pattern: "{key}/{number}-{slug}"   # tokens: {key} {number} {slug}

# ui: TUI and board appearance (personal presentation; prefer ~/.config/kira/config.yaml so pins are not shared across clones)
#   icons: auto               # auto | nerd | emoji | text
#   background: dark          # dark | light

estimate:
  unit: points                # points | hours

# --- optional blocks: uncomment and edit to enable (defaults shown are inert) ---

# filters: named saved queries, run via kira list --filter <name>.
filters:
  next: state = TODO AND NOT blocked ORDER BY priority

# sprints: sprint entities, created by kira sprint create; items key into this list.
#sprints:

# git: relate kira items to the commits and branches that touch them.
#   landed_ref — ref (e.g. origin/main) whose history marks work as merged/landed
#git: {}
```

`kira config init` writes the user-file equivalent, every key commented out
at its default.
