# Concepts

The mental model behind kira. Examples throughout are taken from this
repository's own `.kira/config.yaml`, which is a real, working configuration
with multiple boards, sprints, and saved filters.

A useful companion to this page is `kira config explain`, which prints your
repository's effective rules with provenance:

```console
$ kira config explain
identity (repo)
  project: DEMO (kira-docs-demo)
  id style: sequential
  board: DEMO (kira-docs-demo) [default]

workflow: ticket (default)
  initial: TODO
  wip_policy: warn
  TODO (todo)
  IN_PROGRESS (doing), wip 3/warn
  REVIEW (doing), wip 2/warn
  DONE (done), resolution=done; entering it requires blockers closed (a dangling or unknown blocker counts as satisfied and only warns)
  WONT_DO (done), resolution=dropped; entering it requires blockers closed (a dangling or unknown blocker counts as satisfied and only warns)
...
```

The `(repo)` / `(default)` tag on each section tells you whether that rule came
from your `.kira/config.yaml` or from kira's built-in defaults.

## Items, types, and boards

An **item** is one Markdown file under `.kira/tickets/`. Its `type` field says
what kind of item it is. Two types ship built in:

| Type | Default states | Role |
| --- | --- | --- |
| `ticket` | `TODO`, `IN_PROGRESS`, `REVIEW`, `DONE`, `WONT_DO` | A unit of work. |
| `epic` | `PLANNED`, `ACTIVE`, `DONE` | A container. Tickets point at it via their `epic` field. |

An epic groups tickets; it does not own them in any storage sense. Both live
in the same directory, and both are listed by `kira list` unless you filter
with `--type`. `kira tree` renders the resulting hierarchy, and epic progress
(how many child tickets are in a `done`-category state) is computed from the
children rather than stored on the epic.

A **board** is a namespace with its own numbering. Boards are declared in
config:

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

Every item belongs to exactly one board and is numbered `<KEY>-N` within it —
`CORE-43` and `TUI-43` are different items and neither numbering affects the
other. Boards have four properties:

| Key | Meaning |
| --- | --- |
| `key` | The number prefix. Declared immutable by the board schema. |
| `name` | Display name. Changeable with `kira board rename`. |
| `default` | Exactly one board should carry this. `kira create` puts items here when `--board` is not given. |
| `archived` | Hides the board from pickers; its tickets still resolve and list. |

`kira board move` moves an item to another board. That renumbers it, and the
old number is retained as an **alias** — the `aliases` field you see in every
item file — so anything that referred to the old number still resolves.

If a repository declares no `boards:` block at all, kira synthesizes a single
default board from `project.key` — which is what a fresh `kira init` produces.

## Identity: `id` versus `number`

Every item carries two identifiers, and the distinction matters.

| Field | Example | Properties |
| --- | --- | --- |
| `id` | `01KYHXCJK0A3GJ2QWE1VRXV8F2` | A ULID. Immutable, globally unique, never reused. It is also the filename. |
| `number` | `DEMO-3` | The human-facing display number. Board-scoped, short, and **renumberable**. |

You type `number` at the command line. kira stores `id` on disk — including
inside every cross-reference, which is why relationships survive renumbering
(see [Relationships](#relationships) below).

The `id.style` key selects how `number` is derived:

```yaml
id:
  style: sequential          # sequential | hash
```

| Style | Number looks like | Behaviour |
| --- | --- | --- |
| `sequential` (default) | `DEMO-1`, `DEMO-2`, `DEMO-3` | Counted from the existing items on the board. Two people creating tickets on separate branches will both mint the same number; after the merge, kira detects the collision and renumbers one of them. |
| `hash` | `HSH-1F2QDM` | Derived from the ULID. Two clones can never collide, so nothing is ever renumbered. |

The tradeoff is legibility against stability. `sequential` gives you the tidy
small numbers people actually enjoy saying out loud, at the cost of an item's
number occasionally changing after a merge — a branch that referred to
`DEMO-7` in a commit message may find that `DEMO-7` is now a different ticket.
`hash` never renumbers, so a number written down anywhere stays correct
forever, but it is longer and harder to read.

Reconciliation under `sequential` is deterministic and automatic; `kira sync`
runs it after pulling, and `kira doctor` reports collisions it has not repaired
yet. If your team frequently develops long-lived parallel branches, `hash` is
worth the ugliness. Otherwise `sequential` is the right default.

## Workflow, states, and categories

Each item type has a workflow in `.kira/config.yaml` under `workflows:`. Here
is this repository's ticket workflow:

```yaml
workflows:
  ticket:
    states:
      - { key: TODO,        category: todo }
      - { key: IN_PROGRESS, category: doing, wip: 3 }
      - { key: REVIEW,      category: doing, wip: 2 }
      - { key: DONE,        category: done, resolution: done }
      - { key: WONT_DO,     category: done, resolution: dropped }
    initial: TODO
```

That is the whole workflow: a set of states and the one an item starts in.
`state` is a plain enum — any state may move to any other declared state, in
either direction. There is no transition graph to declare or maintain.

### Categories

Every state declares a `category`, one of `todo`, `doing`, or `done`. Categories
are what tooling reasons about, so that a repository can rename or add states
without breaking reports:

| Category | Used for |
| --- | --- |
| `todo` | Backlog. Not started. |
| `doing` | In flight. Feeds cycle-time metrics. |
| `done` | Closed, for any reason — including `WONT_DO`. |

`kira list --category doing` works regardless of what your states are called.
Note that `done` means "no longer open", not "successfully completed" — which
is what `resolutions_dropped` exists to disambiguate.

A state may also carry `resolution:`, which records a default resolution when
an item lands there — `resolution: done` on `DONE`, `resolution: dropped` on
`WONT_DO`.

### Moving between states

Any state may move to any other state declared in that workflow, in either
direction. Reopening a closed ticket, jumping `TODO` straight to `DONE`, or
skipping `REVIEW` are all ordinary moves, not overrides. The only membership
rule is that the target has to be a state the workflow declares:

```console
$ kira move DEMO-2 SHIPPED
kira: invalid item: "SHIPPED" is not a state in the ticket workflow
  hint: valid states: TODO, IN_PROGRESS, REVIEW, DONE, WONT_DO
```

Two things happen automatically around the `done` category, whatever the state
you came from:

- **Entering a `done` state refuses while the item has open blockers.** Every
  item in `blocked_by` must be in a `done`-category state (a dangling or
  unknown blocker counts as satisfied and only warns).

  ```console
  $ kira move DEMO-2 DONE
  kira: invalid item: REVIEW -> DONE is blocked by open items: DEMO-4
    hint: close the blockers first, or use `--force` to override
  ```

- **`resolution` follows the category.** Entering a `done` state stamps that
  state's declared `resolution:` on the item if it has none; leaving the `done`
  category for a `todo` or `doing` state clears `resolution` again. Pass
  `--resolution` to record something other than the state's default.

### WIP limits

A state can declare `wip: N`, a soft cap on how many items may sit in it. The
workflow-level `wip_policy` decides what happens when you exceed it:

| `wip_policy` | Behaviour over the limit |
| --- | --- |
| `warn` (default) | Warns on stderr, performs the move. |
| `block` | Refuses the move. |

```console
$ kira move DEMO-3 IN_PROGRESS          # wip_policy: warn
kira: warning: IN_PROGRESS is over its WIP limit (4 > 3)
Moved DEMO-3: TODO -> IN_PROGRESS

$ kira move DEMO-8 IN_PROGRESS          # wip_policy: block
kira: invalid item: IN_PROGRESS is over its WIP limit (5 > 3)
  hint: move an item out of IN_PROGRESS first, or use `--force` to override
```

WIP limits are a process signal, not a correctness constraint. `warn` is the
right default for most teams; `block` is for teams who have agreed the limit is
a real rule.

### `--force`

`--force` on `move` bypasses the blockers-closed check and a `block` WIP
policy. On `edit` and `create` it additionally accepts out-of-vocabulary field
values. It is loud about it:

```console
$ kira move DEMO-5 DONE --force
kira: warning: forced past blocker guard: DEMO-4 still open
Moved DEMO-5: IN_PROGRESS -> DONE
```

Note what that example did: it closed a ticket whose blocker is still open, so
`blocked` reports and any query that assumed a closed item had nothing left
under it are now wrong. That is the danger — `--force` does not skip a
formality, it lands the item in a state the guard existed to prevent.

Appropriate uses are narrow: closing work whose blocker is tracked somewhere
else, importing history from another tracker, or exceeding a WIP cap you have
agreed to exceed. It is not needed to move backwards or to skip a state —
those are ordinary moves.

## Labels, priorities, subtypes, and resolutions

These four are **vocabulary fields**: config declares the accepted values, and
kira validates against them. They come in two spellings.

`labels` and `people` use `known` plus `strict`:

```yaml
labels:
  known: [captured]
  strict: false

people:
  known: []
  strict: false
```

`priorities`, `subtypes`, and `resolutions` are either a bare list or a map of
`values` plus `strict`:

```yaml
priorities: [P0, P1, P2, P3]

# equivalently, with explicit strictness:
priorities:
  values: [P0, P1, P2, P3]
  strict: true
```

### Strictness

| `strict` | Out-of-vocabulary value |
| --- | --- |
| `false` (default) | Warning on stderr; the value is accepted and written. |
| `true` | Error; the command fails. `--force` downgrades it back to a warning. |

```console
$ kira edit DEMO-6 --label nope           # strict: true
kira: invalid item: field "labels": value "nope" is not in the "label" vocabulary

$ kira edit DEMO-6 --label nope --force
kira: warning: field "labels": value "nope" is not in the "label" vocabulary
Edited DEMO-6: labels
```

An empty vocabulary (`[]` or `known: []`) means free-form: nothing to validate
against, so nothing is rejected.

One inheritance rule is easy to miss: if `priorities`, `subtypes`, or
`resolutions` is written as a bare list, it has no `strict` of its own and
falls back to `labels.strict`. Setting `labels.strict: true` therefore tightens
all four at once unless you override each explicitly.

### Ordering

`priorities` is ordered, highest first. That order is not just cosmetic — it
defines what `ORDER BY priority` sorts by and what `>` / `<` comparisons in a
query expression mean. With `[]`, priority becomes an equality-only free-form
string.

### `resolutions_dropped`

```yaml
resolutions: [done, dropped, duplicate, cannot-reproduce]
resolutions_dropped: [dropped]
```

`resolutions_dropped` names the resolutions that put an item in a `done`
category without it really having been completed. An item resolved `dropped`
is closed — it is out of the backlog and off the board — but it should not
count as delivered work when epic progress is computed. Without this
distinction, cancelling enough tickets would make an epic look finished.

`wont_fix`, `duplicate`, and `cannot-reproduce` are the usual candidates
depending on how your team wants them counted.

## Sprints

A sprint is declared in config, with a key, a display name, and dates:

```yaml
sprints:
  - { key: 2026-S14, name: Parity + restructure, start: "2026-07-13", end: "2026-07-20" }
```

`kira sprint create` appends to that list and commits it, like any other config
mutation. Items key into a sprint through their `sprint` field, and the value
is validated against the declared keys.

The **active sprint is different in kind**. It is a local, git-ignored, per-clone
pointer, set with:

```console
$ kira sprint activate 2026-S14
Activated sprint 2026-S14
```

It is written under `.kira/.cache/`, which `.kira/.gitignore` excludes, so
activating a sprint produces no commit and no change to `git status`.

This is deliberate. The set of sprints — their names and their dates — is a
shared fact about the project, so it belongs in the committed config. Which
sprint *you* are currently looking at is a personal working-set choice, and it
would be wrong for your teammate's `kira sprint activate` to redirect your
`kira list --sprint active`. The same reasoning applies to the active-ticket
pointer behind `kira now` and `kira workon`.

The practical payoff is that `active` works as a sprint value anywhere a
sprint key does — `kira list --sprint active`, `kira stats --sprint active` —
without you having to type the key or worry about whose clone you are in.

## Relationships

Items refer to each other through three mechanisms.

| Relationship | Field | Cardinality | Constraint |
| --- | --- | --- | --- |
| Parent epic | `epic` | One | Target must be an `epic`. |
| Dependency | `blocked_by` | List | Targets must be the **same type** as the item. |
| Typed link | `links.relates`, `links.duplicate_of` | List each | Free-form association. |

All of them are set with `kira link`:

```console
$ kira link DEMO-1 --epic DEMO-4
Linked DEMO-1

$ kira link DEMO-1 --blocked-by DEMO-3
Linked DEMO-1
```

`--remove` reverses any of them.

The same-type rule on `blocked_by` is enforced, not advisory — a ticket cannot
be blocked by an epic:

```console
$ kira link DEMO-1 --blocked-by DEMO-4
kira: invalid item: field "blocked_by": DEMO-4 is type epic, expected ticket
```

The reason is that "blocked by" is a scheduling relationship between peers,
while an epic's relationship to its tickets is containment. Mixing them makes
the blockers check on a close incoherent: an epic is closed only when its
tickets are, so a ticket blocked by its own epic can never be unblocked.

### References are stored as ULIDs

You type display numbers, but kira writes the immutable `id`:

```yaml
epic: 01KYHXHB98BZFV9D45N07GVV79
blocked_by: [01KYHXCJK0A3GJ2QWE1VRXV8F2]
```

This is what makes renumbering safe. Under `id.style: sequential`, a merge can
change `DEMO-3` into `DEMO-9`; because no other file ever wrote down the string
`DEMO-3`, every link keeps pointing at the same item. Display numbers are
resolved on input and rendered on output, and never persisted inside a
reference.

## Configuration layers

There are two tiers, and the repository tier always wins.

| Tier | Path | Committed | Scope |
| --- | --- | --- | --- |
| Repository | `.kira/config.yaml` | Yes | Authoritative. Workflow, boards, vocabularies, sprints, commit mode, merge policy — everything that must be the same for everyone. |
| User | `~/.config/kira/config.yaml` | No | A personal overlay, limited to a narrow allowed subset. |

The user tier accepts only three top-level keys:

| Key | Covers |
| --- | --- |
| `ui` | Icons, colour, background, editor, list columns, TUI layout. |
| `workon` | Branch naming pattern, casing, `worktree` and `worktree_dir`. |
| `commit` | `subject` only — a personal commit-subject override. |

Anything else in the user file is ignored with a warning naming the offending
key as repo-authoritative. Personal automation hooks live in a separate file,
`~/.config/kira/hooks.yaml`, not in the user config.

`kira config init` scaffolds the user config directory with commented
defaults, and `kira config set` edits a scalar key in place while preserving
comments and formatting.

The one place the precedence inverts is `ui.editor`, which is personal by
definition. Setting it in the committed repo config is ignored, with a warning
pointing you at the user file — a repository has no business dictating which
editor its contributors use.

Precedence anywhere the two tiers collide on a key both may set: **repo wins**.
The rationale is the same one that puts tickets in git — shared facts are
committed, personal preferences are not, and a clone should behave the same for
everyone who checks it out.

For the full key-by-key breakdown of both files, see the
[Configuration Reference](05-configuration-reference.md).

## Extensibility

Item types are not hardcoded — each is defined by a JSON schema file under
`.kira/schema/`, which `kira init` seeds with `ticket.json`, `epic.json`, and
`board.json`. Those files declare each type's fields, their types, whether they
are required or immutable, which config vocabulary validates them, and which
columns `kira list` shows. Adding a field to tickets means adding an entry to
`ticket.json`; adding a whole new item type means adding a schema file and a
matching workflow.

Once a field is declared there, you set it with `--field key=value` on `create`
and `edit`. kira rejects fields the schema does not know about, and tells you
what it does know:

```console
$ kira edit DEMO-6 --field severity=high
kira: invalid item: --field: unknown or immutable field "severity"
  hint: editable fields: acceptance, description, due, epic, estimate, labels, owner, priority, rank, reporter, resolution, sprint, state, subtype, title
```

See [Custom Fields and Schema](06-custom-fields-and-schema.md) for the schema
format and a worked example of adding a field.
