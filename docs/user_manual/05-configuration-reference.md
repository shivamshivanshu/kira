# Configuration Reference

Everything kira knows about your project lives in one committed file:
`.kira/config.yaml`. It defines your boards, your vocabularies, how
commits link to tickets, and how `kira sync` behaves. It is read on every
command, validated strictly, and versioned alongside your code — so a change to
it arrives with the commit that needed it, and code review covers it.

This page documents every key: its shape, its default, and what actually
changes when you set it.

---

## Two config layers

kira reads configuration from two places.

| | Repo config | User config |
|---|---|---|
| Path | `.kira/config.yaml` | `$XDG_CONFIG_HOME/kira/config.yaml`, else `~/.config/kira/config.yaml` |
| Committed? | Yes — shared with everyone | No — personal, per machine |
| Scope | Everything | `ui`, `workon`, and `commit.subject` only |
| On collision | **Wins** | Ignored |

The repo file is authoritative. The user file is a narrow personal overlay for
presentation and local habits — things that should not be forced on your
teammates, like which icon set your terminal can render or whether `kira workon`
creates a worktree.

Only three top-level keys are honoured in the user file:

- `ui:` — the whole UI section (see [`ui`](#ui) below)
- `workon:` — the whole workon section (see [`workon`](#workon) below)
- `commit:` — **only** its `subject` sub-key, which replaces the auto-generated
  subject line for multi-item commits

Any other top-level key is ignored with a warning naming the file:

```
~/.config/kira/config.yaml: key "priorities" is repo-authoritative; ignored
~/.config/kira/config.yaml: key "project" is repo-authoritative; ignored
```

The warning is per key, on stderr, and does not stop the command — the rest of
your user file still applies.

Where both files set the same key, the repo wins. If your user file pins
`ui.icons: text` and the repo config sets `ui.icons: emoji`, you get `emoji`.
This is deliberate: the repo file is the shared contract, and a personal file
should never silently diverge a clone from what its owners configured.

The one exception runs the other way. `ui.editor` is personal by definition, so
a repo config that sets it is ignored with a warning:

```
.kira/config.yaml: ui.editor is personal; set it in ~/.config/kira/config.yaml; ignored
```

### Working with the two layers

```sh
kira config init      # scaffold ~/.config/kira/config.yaml, fully commented out
kira config explain   # print the effective rules, tagged (default) / (repo) / (user)
kira config filters   # list the named saved queries
kira config set KEY VALUE
```

`kira config init` writes the user config with every built-in default present
but commented out, so you uncomment the one line you want to change. It never
overwrites: if the file already exists it reports what it found and writes
nothing.

`kira config explain` is the fastest way to answer "why is kira doing this?".
Each section carries a provenance tag saying where its values came from:

```
ui (repo)
  icons: auto (override with the KIRA_ICONS env var: nerd|emoji|text)
  ...

merge (default)
  policy: auto
```

The tag is per section, not per key — a section shows `(repo)` if the repo file
touched it at all.

`kira config set` edits a single scalar in the repo file by splicing that one
line, leaving every comment and every other line byte-identical. It accepts a
fixed list of keys:

```
commit.mode            estimate.unit      git.landed_ref     merge.policy
project.name           sync.push          ui.background      ui.icons
workon.branch_pattern
```

Everything else is edited by hand. For the full flags and output of these
commands, see
[Setup and schema commands](04-cli-reference/setup-and-schema-commands.md).

### Validation: errors vs warnings

Most mistakes are **hard errors** that fail the command outright:

```
kira: config: ui.icons: invalid value "bogus", want one of [auto nerd emoji text]
```

A small set of UI issues are **warnings** — the value is dropped and kira
continues. These are: an unknown `ui.list.columns` entry, an unknown or
non-hex `ui.theme` slot, and a `ui.tui.split` outside `(0,1)`.

---

## `version`

```yaml
version: 2
```

The schema version of *this config file format* — not the version of the kira
binary, and not anything about your tickets. `kira init` scaffolds
`version: 2` together with a [`boards`](#boards) list holding the project's
default board. A `version: 1` config, from before boards existed, no longer
loads:

```
kira: config: version 1 is no longer supported
  hint: set version: 2 and add boards: [{ key: KIRA, name: kira, default: true }]
```

Older commits whose config is still `version: 1` stay readable, so
`kira diff`, `kira changes` and `list --at` reach back past the upgrade.
Any other version is rejected:

```
kira: config: version: unsupported version 3 (this kira understands 1..2)
```

---

## `project`

```yaml
project:
  key: KIRA
  name: kira
```

| Key | Type | Default |
|---|---|---|
| `key` | string | `KIRA` |
| `name` | string | `kira` |

`project.key` is the prefix for display numbers (`KIRA-12`). `project.name` is
the human label shown in the TUI and board headers.

`kira init` uses these two values for the first, default
[`boards`](#boards) entry. From then on the `boards` list alone decides board
resolution.

`project.name` is settable via `kira config set project.name "..."`.
`project.key` is not — it is baked into every existing display number.

---

## Item types and states are built in

Item types, their states and the item vocabularies are part of kira itself,
not configuration. There are two types, `ticket` and `epic`; referring to any
other is an error:

```
kira: "bug" is not a known type
  hint: known types: epic, ticket
```

Their states, categories, advisory WIP caps and the subtype, priority and
resolution vocabularies are listed in
[Concepts](03-concepts.md#workflow-states-and-categories).

Older configs may still carry `workflows`, `priorities`, `subtypes`,
`resolutions`, `resolutions_dropped` or `fields`. kira no longer reads these
keys and ignores them; delete them at your convenience.

---

## Vocabularies: `labels`, `people`

Two keys that constrain what values a field may take, each with `known` plus
`strict`:

```yaml
labels:
  known: [captured]
  strict: false

people:
  known: []
  strict: false
```

### `strict`

`strict: true` rejects any value outside the vocabulary; `--force` accepts it
anyway. `strict: false` is advisory — the value is accepted with a warning, and
`kira doctor` reports it at a lower severity.

The built-in priority, subtype and resolution vocabularies follow
**`labels.strict`**: writing `labels: {strict: true}` turns strictness on for
all of them at once.

`kira config explain` prints the effective answer per vocabulary, so you never
have to work it out:

```
vocab (repo)
  labels: captured (advisory)
  priorities: P0, P1, P2, P3 (advisory)
  subtypes: bug, task, feature, support (advisory)
  resolutions: done, dropped, duplicate, cannot-reproduce (advisory)
```

### `people` entries

A person may be written as a bare name, or as a mapping that maps git
identities to a canonical name:

```yaml
people:
  known:
    - jdoe
    - { name: asmith, git: ["Alice Smith", "alice@example.com"] }
  strict: false
```

The `git:` aliases let kira attribute a commit to `asmith` regardless of which
name or address the commit was authored under. Every entry needs a `name`.

### The `captured` label

`captured` is a system label kira adds itself to items created with `kira create
--here`. It is exempt from vocabulary checks, so you never need to declare it —
though this repo lists it anyway for documentation value.

---

## `filters`

```yaml
filters:
  backlog: "state = TODO ORDER BY priority"
  core: "label = core AND state != DONE"
```

| Type | Default |
|---|---|
| map name → query string | `{}` |

Named saved queries. Run one with `kira list --filter backlog`; list them all
with `kira config filters`. They also drive board chips in the TUI.

Every filter is parsed at config load, so a typo in a saved query fails
immediately with the position of the problem — you find out when you write it,
not three weeks later when someone runs it.

The query language supports field comparisons (`state = TODO`, `priority > P2`,
`due < 2026-07-12`), boolean composition with `AND` / `OR` / `NOT`, emptiness
tests (`blocked_by IS NOT EMPTY`), and `ORDER BY`. For the full grammar and the
list of queryable fields, see
[Views and queries](04-cli-reference/views-and-queries.md).

---

## `boards`

```yaml
boards:
  - { key: KIRA, name: kira, default: true }
  - { key: CORE, name: Core store and domain }
  - { key: TUI, name: Terminal UI }
  - { key: CLI, name: Command surface }
  - { key: DATA, name: Index query and schema }
  - { key: INFRA, name: CI CD and build }
  - { key: DX, name: Tooling and hygiene }
```

| Key | Type | Required | Default |
|---|---|---|---|
| `key` | string matching `^[A-Z][A-Z0-9]{1,9}$` | yes | — |
| `name` | string | yes | — |
| `description` | string | no | empty |
| `default` | bool | no | `false` |
| `archived` | bool | no | `false` |

Boards partition items into independent numbering streams — an item on the
`CORE` board is `CORE-46`, not `KIRA-46`.

Rules enforced at load: keys are 2–10 characters, uppercase alphanumeric,
starting with a letter; keys are unique case-insensitively; `name` is
non-empty; and at most one board may be `default`.

`archived: true` keeps a board's items readable and its numbers resolvable
while removing it from pickers and from the default board list. An archived
board cannot serve as the default.

`kira init` writes the list with one default board keyed by `project.key`.
The list must hold at least one board; a config without one does not load:

```
kira: config: boards: at least one board is required
  hint: add boards: [{ key: KIRA, name: kira, default: true }]
```

See [Boards and sprints](04-cli-reference/boards-and-sprints.md) for working
with them day to day.

---

## `sprints`

```yaml
sprints:
  - { key: 2026-S14, name: Parity + restructure, start: "2026-07-13", end: "2026-07-20" }
```

| Key | Type | Required |
|---|---|---|
| `key` | string | yes |
| `name` | string | yes |
| `start` | date, `YYYY-MM-DD` | yes |
| `end` | date, `YYYY-MM-DD` | yes |

An item's `sprint` frontmatter field is a key into this list. Keys must be
unique, names non-empty, both dates well-formed, and `start` strictly before
`end`.

Quote the dates. Bare `2026-07-13` parses as a YAML date rather than a string
in some situations; the quoted form is unambiguous. Sprints are normally
managed with `kira sprint create` rather than edited by hand.

---

## `commit`

```yaml
commit:
  mode: auto                  # auto | manual
```

| Key | Type | Default |
|---|---|---|
| `mode` | `auto` \| `manual` | `auto` |

**`mode`** decides what happens to `.kira/` after a mutation.

- `auto` — kira stages and commits the change immediately. Most commands
  (`create`, `edit`, `move`, `label`, `link`, …) each produce their own commit.
  This is the default and what this repo uses.
- `manual` — changes are written and staged but not committed; you run `kira
  commit` when you want to bundle them.

Subjects kira generates carry no prefix: `KIRA-46 state TODO -> DONE`.

### Commit linking

The trailers are fixed. `Kira-Ticket: KIRA-46` links a commit to an item; a
single commit's trailer may name up to 20 numbers. `Kira-Closes: KIRA-46`
*closes* the item when its commit lands on [`git.landed_ref`](#git), moving it
to its type's first `done`-category state.

kira also *recognizes* a ticket in a commit that was written by hand — by you,
or by a tool that knows nothing about kira. This matters because `kira show`
lists the commits that touched an item, and that list is built by scanning
history.

The distinction is strength of association. A **link** is a claim that this
commit is work on that ticket. A **reference** is a passing mention. They are
stored separately and displayed differently.

| Marker | Kind | Matches |
|---|---|---|
| trailer | link | a `Kira-Ticket:` git trailer naming the ticket |
| subject | link | a `[[KEY-n]]` marker in the commit subject |
| leading number | link | a bare `KEY-n` token leading the subject |
| bare | reference | a `KEY-n` token anywhere in the commit body |

### `commit.subject` — the user-file key

The one `commit` sub-key honoured in the *user* config is `subject`, which
replaces the auto-generated subject for multi-item commits:

```yaml
# ~/.config/kira/config.yaml
commit:
  subject: "chore(tickets): batch update"
```

By default that subject is `update {count} items` (with up to five
numbers listed before falling back to a count). It must be a single line.

---

## `merge`

```yaml
merge:
  policy: auto     # auto | manual
```

| Type | Default |
|---|---|
| `auto` \| `manual` | `auto` |

How conflicts inside `.kira/` item files are resolved.

**`auto`** engages kira's field-level three-way merge. Because an item file is
structured YAML frontmatter, kira can merge per field rather than per line: if
one side changed `owner` and the other changed `state`, both changes survive
with no conflict at all. This runs in two places — as the git merge driver
registered for `.kira/**`, and inside `kira sync`, which auto-resolves item
conflicts mid-rebase and continues.

**`manual`** turns all of that off. Conflicting item files get ordinary git
conflict markers and are left for you. `kira sync` will not try to push through
them; it aborts the rebase and says so:

```
kira: rebase halted with conflicts (merge.policy: manual)
```

Choose `manual` if you would rather see every concurrent edit yourself than
have a policy decide. Choose `auto` — the default — if you would rather the
mechanical 90% of item conflicts never reach you.

---

## `sync`

```yaml
sync:
  push: false      # true | false
```

| Key | Type | Default |
|---|---|---|
| `push` | bool | `false` |

**`push`** — when `true`, a bare `kira sync` publishes after a clean pull.
Default is pull-only, so `sync` never puts anything on a remote unless you
asked.

A dirty `.kira/` makes `kira sync` refuse; `kira sync --commit` commits it
first as a `sync checkpoint`. See
[Git integration](04-cli-reference/git-integration.md) for the full sync
pipeline.

---

## `workon`

```yaml
workon:
  branch_pattern: "{key}/{number}-{slug}"
  worktree: false
  worktree_dir: "../{repo}-{branch}"
```

| Key | Type | Default |
|---|---|---|
| `branch_pattern` | template string | `{key}/{number}-{slug}` |
| `worktree` | bool | `false` |
| `worktree_dir` | template path | `../{repo}-{branch}` |

**`branch_pattern`** is the branch name `kira workon` creates. Tokens: `{key}`
(board key), `{number}` (display number), `{slug}` (the title, slugified). The
default turns `KIRA-46: Report an unresolvable ref once` into
`KIRA/46-report-an-unresolvable-ref-once`.

It **must contain `{number}`** — without it, distinct tickets would collide on
one branch name, so kira rejects the pattern at load:

```
kira: config: workon.branch_pattern: must contain {number}, got "{key}/{slug}"
```

**`worktree`** makes `kira workon` create a git worktree by default instead of
switching the current checkout — the same as passing `--worktree` every time.

**`worktree_dir`** is where those worktrees go. Tokens: `{repo}`, `{branch}`,
`{key}`, `{number}`. Relative paths resolve against the repo; absolute paths
are used as-is. It may not start with `~`, because kira does not expand it and
you would get a literal `~` directory:

```
kira: config: workon.worktree_dir: "~/wt/{branch}" begins with ~, which is not expanded
  hint: use an absolute or repo-relative path
```

All four keys are also honoured in the user config, which is usually the right
home for `worktree` and `worktree_dir` — where you like your worktrees is a
property of your machine, not of the project.

---

## `ui`

Presentation. Every key here is personal, and belongs in
`~/.config/kira/config.yaml` rather than the repo file unless the whole team
genuinely needs it pinned.

```yaml
ui:
  icons: auto              # auto | nerd | emoji | text
  background: dark         # dark | light
  color: auto              # auto | always | never
  editor: ""
  auto_tui: true
  list:
    columns: [number, state, type, priority, title]
  tui:
    split: 0.5
    refresh: "0"
  theme:
    accent: "#7dcfff"
    dim: "#6c7086"
```

This repo's own config sets only one of them:

```yaml
ui:
  icons: auto     # if glyphs render as boxes, set emoji or text (nerd font not installed)
```

| Key | Type | Default |
|---|---|---|
| `icons` | `auto` \| `nerd` \| `emoji` \| `text` | `auto` |
| `background` | `dark` \| `light` | `dark` |
| `color` | `auto` \| `always` \| `never` | `auto` |
| `editor` | string | empty (falls back to `$EDITOR`, then `$VISUAL`) |
| `auto_tui` | bool | `true` |
| `list.columns` | list of column names | `[number, state, type, priority, title]` |
| `tui.split` | float in `(0,1)` | `0.5` |
| `tui.refresh` | duration string | `"0"` (disabled) |
| `theme` | map slot → hex colour | unset |

**`icons`** picks the glyph set. `nerd` uses Nerd Font glyphs, `emoji` uses
emoji, `text` uses plain ASCII, and `auto` probes your terminal. If kira's
output shows boxes or question marks, you do not have a Nerd Font installed —
set `emoji` or `text`.
The `KIRA_ICONS` environment variable overrides this per invocation.

**`background`** tells the TUI and board which palette to assume, for terminals
whose background kira cannot detect. Set it if your colours look washed out or
unreadable.

**`color`** is standard ANSI colour control. `auto` colours when stdout is a
terminal. The `--no-color` global flag and the usual environment conventions
also apply.

**`editor`** is the command used for drafting item bodies and comments,
overriding `$EDITOR` and `$VISUAL`. It runs through `sh -c`, so shell quoting
works and you can pass flags (`code --wait`, `nvim -c 'set ft=markdown'`).
This key is *only* read from the user config — a repo config that sets it is
ignored with a warning, since forcing an editor on every clone of a repo is
never right.

**`auto_tui`** decides what a bare `kira` does. When `true` (the default) and
you are on a terminal, it launches the TUI. It is suppressed automatically
under `--json` or `--no-input`. Set it to `false` if you want bare `kira` to
print help instead.

**`list.columns`** customizes which columns `kira list` renders, in the order
given. This repo pins `[number, state, type,
subtype, priority, title]`; the shipped default is `[number, state, type,
priority, title]`. The full set of selectable columns is:

```
number  title  type  subtype  state  category  priority  owner
labels  epic   epic_number  resolution  due  id
```

Any other name is a warning, not an error: the column is dropped and the rest
of the list still renders:

```
kira: warning: ui.list.columns: unknown column "assignee"; ignored
```

**`tui.split`** is the fraction of width given to the tree pane versus the
detail pane. Must be strictly between 0 and 1; out-of-range values warn and
fall back.

**`tui.refresh`** is how often the TUI re-reads the store, as a Go duration
string (`5s`, `1m`). `"0"` disables auto-refresh, which is the default.
Anything below the one-second minimum is rejected:

```
kira: config: ui.tui.refresh: 500ms is below the 1s minimum
  hint: use 0 to disable
```

**`theme`** overrides individual palette slots with hex colours. There are
eight slots:

| Slot | Used for |
|---|---|
| `accent` | selection, focus, highlights |
| `dim` | secondary and de-emphasized text |
| `todo` | `todo`-category states |
| `doing` | `doing`-category states |
| `done` | `done`-category states |
| `warm` | approaching a threshold (near a WIP cap, due soon) |
| `hot` | over a threshold (past a WIP cap, overdue) |
| `border` | pane borders and rules |

Values must be `#RGB` or `#RRGGBB`. Set only the slots you want to change; the
rest keep their defaults. Unknown slot names and malformed colours warn and are
dropped:

```
~/.config/kira/config.yaml: ui.theme: unknown slot "highlight"; ignored
~/.config/kira/config.yaml: ui.theme.accent: invalid color "blue"; ignored
```

---

## `git`

```yaml
git:
  landed_ref: origin/main
```

| Key | Type | Default |
|---|---|---|
| `landed_ref` | git ref | resolved `origin/HEAD`, falling back to `main` |

**`landed_ref`** is the ref whose history means "this work has reached
mainline". It is what drives automatic closing: when kira scans commits and
finds a `Kira-Closes:` trailer in this ref's history, the named item moves to
its type's first `done`-category state. It is also the boundary `kira
index --closes` and the git hooks work against.

By default kira resolves `origin/HEAD` and falls back to `main` if that fails.
Set it explicitly if your mainline is `origin/master`, `origin/develop`, or a
release branch. This is one of the keys `kira config set git.landed_ref …`
handles.

kira remembers a watermark per landed ref, so repeated scans only walk commits
it has not already seen.

---

## `estimate`

```yaml
estimate:
  unit: points     # points | hours
```

| Type | Default |
|---|---|
| `points` \| `hours` | `points` |

The unit for an item's `estimate` field, used when rendering estimates and
totals. It is a label and a formatting choice — kira does not convert between
the two or attach any other meaning.

---

## A complete example

This repo's own `.kira/config.yaml`, in full — a working config for a
single-maintainer project with several boards:

```yaml
version: 2

project:
  key: KIRA
  name: kira

filters:
  backlog: "state = TODO ORDER BY priority"
  core: "label = core AND state != DONE"

labels:
  known: [captured]
  strict: false

people:
  known: []
  strict: false

commit:
  mode: auto

merge:
  policy: auto

sync:
  push: false

ui:
  icons: auto
  list:
    columns: [number, state, type, subtype, priority, title]

estimate:
  unit: points

sprints:
  - { key: 2026-S14, name: Parity + restructure, start: "2026-07-13", end: "2026-07-20" }

boards:
  - { key: KIRA, name: kira, default: true }
  - { key: CORE, name: Core store and domain }
  - { key: TUI, name: Terminal UI }
  - { key: CLI, name: Command surface }
  - { key: DATA, name: Index query and schema }
  - { key: INFRA, name: CI CD and build }
  - { key: DX, name: Tooling and hygiene }
```

Everything not listed here takes its default. Run `kira config explain` in your
own repo to see the whole effective picture, defaults included.
