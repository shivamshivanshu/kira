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

Personal automation hooks are a separate file — see
[`automation`](#automation).

### Working with the two layers

```sh
kira config init      # scaffold ~/.config/kira/{config.yaml,hooks.yaml}, fully commented out
kira config explain   # print the effective rules, tagged (default) / (repo) / (user)
kira config filters   # list the named saved queries
kira config set KEY VALUE
```

`kira config init` writes both user files with every built-in default present
but commented out, so you uncomment the one line you want to change. It never
overwrites: if either file already exists it reports what it found and writes
nothing.

`kira config explain` is the fastest way to answer "why is kira doing this?".
Each section carries a provenance tag saying where its values came from:

```
ui (repo)
  icons: auto (override with the KIRA_ICONS env var: nerd|emoji|text|always|never)
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
commit.close_trailer   commit.mode        commit.trailer     estimate.unit
git.landed_ref         merge.policy       project.name       sync.push
ui.background          ui.icons           workon.branch_pattern
workon.casing
```

Everything else is edited by hand. For the full flags and output of these
commands, see
[Setup and schema commands](04-cli-reference/setup-and-schema-commands.md).

### Validation: errors vs warnings

Most mistakes are **hard errors** that fail the command outright:

```
kira: config: ui.icons: invalid value "bogus", want one of [auto nerd emoji text always never]
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
binary, and not anything about your tickets. Current kira understands `1` and
`2`; anything else is rejected:

```
kira: config: version: unsupported version 3 (this kira understands 1..2)
```

`kira init` scaffolds `version: 1`. Bump it to `2` when you want a
[`boards`](#boards) list, which is the only feature gated on it:

```
kira: config: boards require version 2, found 1
  hint: bump version to 2 or remove the boards list
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

If you have no [`boards`](#boards) list, kira synthesizes a single default
board from these two values, so `project.key` doubles as your only board key.
Once you declare `boards:`, that list takes over completely and `project.key`
is no longer used for board resolution.

`project.name` is settable via `kira config set project.name "..."`.
`project.key` is not — it is baked into every existing display number.

---

## `id`

```yaml
id:
  style: sequential     # sequential | hash
```

| Key | Type | Default |
|---|---|---|
| `style` | `sequential` \| `hash` | `sequential` |

This picks how kira mints the *display number* for a new item. Every item also
carries an immutable ULID that never changes and is what links actually point
at; the display number is the short, typeable handle layered on top.

**`sequential`** gives you `KIRA-1`, `KIRA-2`, `KIRA-3`. The next number is the
highest integer suffix already in use for that key, plus one. Small, tidy,
sortable, easy to say out loud.

The cost is that two people working on separate branches both mint `KIRA-42`,
and the collision only surfaces at merge time. kira handles this — `kira
doctor` detects colliding numbers and produces a deterministic renumber plan,
and the item that gets renumbered keeps its old number as an alias so old
references still resolve — but it *is* a step that has to happen, and it
rewrites numbers you may already have typed into a commit message or a chat
thread.

**`hash`** derives the number from the item's ULID instead: kira takes the last
six characters of the ULID and lengthens the suffix until it is both unused and
not all-digits, giving numbers like `KIRA-N3X2T4`. Because the ULID is unique
by construction, two branches can never mint the same number, and no
reconciliation is ever needed. The cost is that the numbers are longer, ugly,
and unordered — you lose "KIRA-3 came before KIRA-40" at a glance.

Pick `sequential` for a small team on a shared trunk where merges are frequent
and renumbering is cheap. Pick `hash` for a large or heavily-branched repo, or
anywhere a stable number matters more than a pretty one.

**On changing this later:** kira does not block the change, and it is safer than
it looks. Hash-style suffixes are guaranteed never to be all digits, and
sequential suffixes are always all digits, so the two forms occupy disjoint
namespaces and can coexist without colliding. Existing items keep the numbers
they already have. But nothing migrates them either — there is no renumber-all
command — so flipping the style on a populated repo leaves you with a
permanently mixed set of numbers. Treat it as a decision to make once, at
`kira init` time.

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
`CORE` board is `CORE-46`, not `KIRA-46`. Requires [`version: 2`](#version).

Rules enforced at load: keys are 2–10 characters, uppercase alphanumeric,
starting with a letter; keys are unique case-insensitively; `name` is
non-empty; and at most one board may be `default`.

`archived: true` keeps a board's items readable and its numbers resolvable
while removing it from pickers and from the default board list. An archived
board cannot serve as the default.

Omitting `boards:` entirely is fine — kira synthesizes a single default board
from `project.key` and `project.name`. Once you declare the list, it replaces
that synthesized board completely, so remember to include an entry for your
original project key (as this repo does with `KIRA`).

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
  mode: auto                  # auto | manual | prompt
  subject_prefix: ""
  trailer: Kira-Ticket
```

| Key | Type | Default |
|---|---|---|
| `mode` | `auto` \| `manual` \| `prompt` | `auto` |
| `trailer` | string | `Kira-Ticket` |
| `close_trailer` | string | `Kira-Closes` |
| `subject_prefix` | string, single line | `"kira: "` |
| `link_markers` | list | `[trailer, subject, leading_number]` |
| `reference_markers` | list | `[bare]` |

**`mode`** decides what happens to `.kira/` after a mutation.

- `auto` — kira stages and commits the change immediately. Most commands
  (`create`, `edit`, `move`, `label`, `link`, …) each produce their own commit.
  This is the default and what this repo uses.
- `manual` — changes are written and staged but not committed; you run `kira
  commit` when you want to bundle them.
- `prompt` — kira asks. This needs a terminal; without one, changes stage but
  are not committed.

**`trailer`** is the git trailer key that links a commit to an item, e.g.
`Kira-Ticket: KIRA-46`. It may not be empty — an empty trailer key would
disable commit linking entirely, so kira rejects it:

```
kira: config: commit.trailer: required; an empty trailer key disables all commit linking
```

A single commit's trailer may name up to 20 numbers.

**`close_trailer`** is the trailer that *closes* an item when its commit lands
on [`git.landed_ref`](#git), moving it to its type's first
`done`-category state. Writing `Kira-Closes: KIRA-46` in a commit
means the ticket transitions by itself once that commit reaches mainline.

**`subject_prefix`** is prepended to every commit subject kira generates —
`kira: KIRA-46 state TODO -> DONE` with the default `"kira: "`. It must be a
single line. This repo sets it to `""`, which is why its history reads
`CORE-46 state TODO -> DONE` with no prefix. Setting it lets you filter kira's
own commits out of `git log` with a single grep, or fold them into an existing
convention.

**`link_markers` and `reference_markers`** control how kira *recognizes* a
ticket in a commit that was written by hand — by you, or by a tool that knows
nothing about kira. This matters because `kira show` lists the commits that
touched an item, and that list is built by scanning history.

The distinction is strength of association. A **link** is a claim that this
commit is work on that ticket. A **reference** is a passing mention. They are
stored separately and displayed differently.

| Marker | Kind | Matches |
|---|---|---|
| `trailer` | link | a `Kira-Ticket:` git trailer naming the ticket |
| `subject` | link | a `[KEY-n]` marker in the commit subject |
| `leading_number` | link | a bare `KEY-n` token leading the subject |
| `bare` | reference | a `KEY-n` token anywhere in the commit body |

All three link markers and the one reference marker are on by default. Drop
`bare` from `reference_markers` (set it to `[]`) if your commit bodies mention
ticket numbers conversationally and the noise is not worth it. Drop
`leading_number` if your subjects start with something else that looks like a
ticket number.

Changing any of these invalidates the cached commit scan, so the next `kira
index` re-scans history under the new rules.

### `commit.subject` — the user-file key

The one `commit` sub-key honoured in the *user* config is `subject`, which
replaces the auto-generated subject for multi-item commits:

```yaml
# ~/.config/kira/config.yaml
commit:
  subject: "chore(tickets): batch update"
```

By default that subject is `kira: update {count} items` (with up to five
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

and on a conflicted stash pop:

```
kira: stash pop conflicted (merge.policy: manual); resolve, then `git stash drop`
```

Choose `manual` if you would rather see every concurrent edit yourself than
have a policy decide. Choose `auto` — the default — if you would rather the
mechanical 90% of item conflicts never reach you.

---

## `sync`

```yaml
sync:
  push: false      # true | false
  dirty: auto      # auto | fail | commit | stash
```

| Key | Type | Default |
|---|---|---|
| `push` | bool | `false` |
| `dirty` | `auto` \| `fail` \| `commit` \| `stash` | `auto` |

**`push`** — when `true`, a bare `kira sync` publishes after a clean pull.
Default is pull-only, so `sync` never puts anything on a remote unless you
asked.

**`dirty`** decides what `kira sync` does when `.kira/` has uncommitted changes
before it pulls, since a rebase over a dirty tree will not start.

| Value | Behaviour |
|---|---|
| `fail` | refuse to sync: *"uncommitted kira changes; re-run with `--commit` or `--stash`"* |
| `commit` | commit them first as a `sync checkpoint` commit, then pull |
| `stash` | stash them, pull, and pop afterwards |
| `auto` | `commit` when `commit.mode: auto`, otherwise `fail` |

The `auto` default is the right instinct in both directions: under
`commit.mode: auto` an uncommitted `.kira/` change is an accident of timing and
committing it is harmless, whereas under `manual` you are deliberately batching
changes and kira should not commit them behind your back.

The `--commit` and `--stash` flags on `kira sync` override this per run. See
[Git integration](04-cli-reference/git-integration.md) for the full sync
pipeline.

---

## `workon`

```yaml
workon:
  branch_pattern: "{key}/{number}-{slug}"
  casing: kebab                          # kebab | snake
  worktree: false
  worktree_dir: "../{repo}-{branch}"
```

| Key | Type | Default |
|---|---|---|
| `branch_pattern` | template string | `{key}/{number}-{slug}` |
| `casing` | `kebab` \| `snake` | `kebab` |
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

**`casing`** is the word separator used when building `{slug}`: `kebab` gives
`report-an-unresolvable-ref`, `snake` gives `report_an_unresolvable_ref`.

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
  icons: auto              # auto | nerd | emoji | text | always | never
  background: auto         # auto | dark | light
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
| `icons` | `auto` \| `nerd` \| `emoji` \| `text` \| `always` \| `never` | `auto` |
| `background` | `auto` \| `dark` \| `light` | `auto` |
| `color` | `auto` \| `always` \| `never` | `auto` |
| `editor` | string | empty (falls back to `$EDITOR`, then `$VISUAL`) |
| `auto_tui` | bool | `true` |
| `list.columns` | list of column names | `[number, state, type, priority, title]` |
| `tui.split` | float in `(0,1)` | `0.5` |
| `tui.refresh` | duration string | `"0"` (disabled) |
| `theme` | map slot → hex colour | unset |

**`icons`** picks the glyph set. `nerd` uses Nerd Font glyphs, `emoji` uses
emoji, `text` uses plain ASCII, and `auto` probes your terminal. `always` and
`never` force glyphs on or off wholesale. If kira's output shows boxes or
question marks, you do not have a Nerd Font installed — set `emoji` or `text`.
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

## `automation`

Shell commands that fire on kira events. Repo-wide automation lives in
`.kira/config.yaml` under `automation:`; your personal automation lives in a
separate file, `~/.config/kira/hooks.yaml`.

This repo defines no automation hooks. A representative one:

```yaml
automation:
  - name: notify-reviewers
    on: item.state_changed
    run: 'curl -sX POST "$WEBHOOK" -d "$KIRA_NUMBER moved to $KIRA_TO"'
    enabled: true
    timeout: 10s
    match:
      type: ticket
      to: REVIEW
```

| Key | Type | Required | Default |
|---|---|---|---|
| `name` | string | no | falls back to the event name |
| `on` | event name | yes | — |
| `run` | shell command | yes | — |
| `enabled` | bool | no | `true` |
| `timeout` | duration string | no | `30s` |
| `match` | mapping | no | fire on every occurrence |

### Events

| Event | Fires when |
|---|---|
| `item.created` | a new item is created |
| `item.state_changed` | an item moves between states |
| `sync.completed` | a `kira sync` run finishes |

Any other value is rejected at config load.

### `match`

Narrows a hook to specific occurrences.

| Key | Matches against |
|---|---|
| `type` | the item's type (`ticket`, `epic`, …) |
| `to` | the destination state — **or its category** |
| `from` | the origin state — **or its category** |

`to` and `from` accept either level, which is often what you want: `to: done`
fires on every closing transition regardless of which done state it lands in,
while `to: DONE` fires only for that one state.

All specified conditions must hold. Fields that cannot apply to the event are
rejected rather than silently ignored — `to`/`from` on `item.created`, and
`type`/`to`/`from` on `sync.completed`:

```
kira: config: automation[0].match: to/from are never set on "item.created" events
```

### How a hook runs

The command runs through `sh -c` with the repo root as its working directory.
Its stdout and stderr are captured and echoed line by line, prefixed with the
hook's name:

```
[automation:notify-reviewers] posted to #eng-reviews
```

On timeout the whole process group is killed and kira reports it:

```
[automation:notify-reviewers] timed out after 10s
```

`timeout` must be a positive duration; the default is 30 seconds. A hook that
fails does not fail the kira command that triggered it — the error is reported
and the command still succeeds.

Hooks are protected against recursion: `KIRA_AUTOMATION=1` is set in the
child's environment, and kira fires no hooks when it sees that variable. A hook
may safely call `kira` itself without triggering an infinite loop — but note
that the nested call fires no hooks of its own.

### Environment variables

Every hook receives these thirteen variables, in addition to the ambient
environment. All are always set; ones that do not apply to the event are empty
strings.

| Variable | Value |
|---|---|
| `KIRA_AUTOMATION` | always `1` — the recursion guard |
| `KIRA_EVENT` | the event name, e.g. `item.state_changed` |
| `KIRA_ITEM` | the item's ULID |
| `KIRA_NUMBER` | the item's display number, e.g. `KIRA-46` |
| `KIRA_TYPE` | the item's type, e.g. `ticket` |
| `KIRA_TITLE` | the item's title |
| `KIRA_FROM` | origin state (state-change events only) |
| `KIRA_TO` | destination state (state-change events only) |
| `KIRA_FROM_CATEGORY` | origin state's category |
| `KIRA_TO_CATEGORY` | destination state's category |
| `KIRA_SOURCE` | what caused the change: `cli`, `trailer`, or `sync` |
| `KIRA_ROOT` | absolute path to the repo root |
| `KIRA_COMMIT` | SHA of the commit kira made for this change |

A real capture from an `item.state_changed` hook:

```
KIRA_AUTOMATION=1
KIRA_COMMIT=3a3d641c55c39d9f506f2189ee4d91c7e261093e
KIRA_EVENT=item.state_changed
KIRA_FROM=TODO
KIRA_FROM_CATEGORY=todo
KIRA_ITEM=01KYHXN3X2T4W078B1V7B2NE27
KIRA_NUMBER=DEMO-1
KIRA_ROOT=/home/you/src/demo
KIRA_SOURCE=cli
KIRA_TITLE=test item
KIRA_TO=IN_PROGRESS
KIRA_TO_CATEGORY=doing
KIRA_TYPE=ticket
```

### The stdin payload

For anything richer than the environment variables, read stdin: kira writes a
single JSON object there.

```json
{
  "payload_version": 1,
  "event": "item.state_changed",
  "source": "cli",
  "ts": "2026-07-27T19:24:14+05:30",
  "repo": "/home/you/src/demo",
  "actor": { "name": "Ada Lovelace", "email": "ada@example.com" },
  "changes": { "state": { "old": "TODO", "new": "IN_PROGRESS" } },
  "from": "TODO",
  "to": "IN_PROGRESS",
  "to_category": "doing",
  "from_category": "todo",
  "commit": "3a3d641c55c39d9f506f2189ee4d91c7e261093e",
  "item": { "number": "DEMO-1", "title": "test item", "...": "…" }
}
```

`item` is the full item in the same shape `kira show --json` emits — number,
title, state, labels, links, body, comments and the rest. `changes` maps each
changed field to its `old` and `new` values. `sync` carries the sync report on
`sync.completed` events. `payload_version` is `1`; check it if you want your
script to be robust across kira upgrades.

Fields that do not apply to an event are omitted from the JSON entirely (unlike
the environment variables, which are present but empty).

### Trust: the security boundary

Repo automation and user automation are trusted differently, and the direction
may not be the one you expect.

**Repo hooks — from `.kira/config.yaml` — must be explicitly trusted before
they run.** kira takes a hash of the repo's `automation:` list and stores it
under `.kira/.cache/`. Until the hash on disk matches the config, no repo hook
fires, and kira tells you:

```
kira: 1 automation hook defined but not trusted — run `kira automation trust`
```

`kira automation trust` grants it. Any subsequent edit to the `automation:`
list changes the hash and revokes trust, so you are asked again.

**User hooks — from `~/.config/kira/hooks.yaml` — run without any trust step.**

The reason is who wrote the file. `.kira/config.yaml` arrives from other people:
a `git pull` can bring in a new `automation:` entry, and without a gate that
would be silent arbitrary code execution on your machine the next time you ran
any kira command. Cloning a hostile repo would be enough. The trust hash makes
that impossible — a pulled change to the automation list cannot run until you
have looked at it and said yes. Your own `hooks.yaml`, by contrast, is a file
only you can write, so gating it would protect you from nobody.

Inspect what is defined and whether it is trusted:

```sh
kira automation list
```

```
hook dump-env on item.state_changed: env | grep ^KIRA_ | sort > /tmp/env.txt
hook user-hook on item.state_changed: notify-send kira "$KIRA_NUMBER" (user)
trust: not trusted (run `kira automation trust`)
```

The `(user)` suffix marks hooks from your personal file. The `trust:` line
refers only to the repo hooks; user hooks fire regardless of what it says.

> **Note:** the commented `hooks.yaml` scaffolded by `kira config init` says to
> run `kira automation trust` after editing it. That appears to be inaccurate —
> the trust hash covers only the repo's `automation:` list, and user hooks fire
> without it, which is what actually happens when you test it. Running the
> command after editing `hooks.yaml` is harmless but unnecessary.

Two further guardrails worth knowing: writing `automation:` into your *user*
`config.yaml` does not work — personal hooks belong in `hooks.yaml`, and kira
says so:

```
~/.config/kira/config.yaml: automation: personal hooks belong in hooks.yaml; ignored
```

And a malformed `hooks.yaml` — bad YAML, an unknown event, a missing `run` — is
reported and skipped in full rather than partially applied.

---

## A complete example

This repo's own `.kira/config.yaml`, in full — a working config for a
single-maintainer project with several boards:

```yaml
version: 2

project:
  key: KIRA
  name: kira

id:
  style: sequential

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
  subject_prefix: ""
  trailer: Kira-Ticket

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
