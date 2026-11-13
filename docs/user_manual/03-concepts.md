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
  board: DEMO (kira-docs-demo) [default]

vocab (default)
  system label: captured (auto-added by `kira create --here`; excluded from vocabulary checks)
  labels: none (advisory)
  people: none (advisory)
  priorities: P0, P1, P2, P3 (advisory)
  subtypes: bug, task, feature, support (advisory)
  resolutions: done, dropped, duplicate, cannot-reproduce (advisory)
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
| `key` | The number prefix. Fixed once the board exists. |
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

Display numbers are sequential: `DEMO-1`, `DEMO-2`, `DEMO-3`, counted from the
existing items on the board. Two people creating tickets on separate branches
will both mint the same number; after the merge, kira detects the collision and
renumbers one of them, and the renumbered item keeps its old number as an
alias. A branch that referred to `DEMO-7` in a commit message may therefore find
that `DEMO-7` is now a different ticket.

Reconciliation is deterministic and automatic; `kira sync` runs it after
pulling, and `kira doctor` reports collisions it has not repaired yet.

## Workflow, states, and categories

Each item type has a fixed workflow built into kira — a set of states and the
one an item starts in:

| Type | States (category) | Initial |
| --- | --- | --- |
| `ticket` | `TODO` (todo), `IN_PROGRESS` (doing, WIP 3), `REVIEW` (doing, WIP 2), `DONE` (done), `WONT_DO` (done) | `TODO` |
| `epic` | `PLANNED` (todo), `ACTIVE` (doing), `DONE` (done) | `PLANNED` |

`state` is a plain enum — any state may move to any other state of the item's
type, in either direction. There is no transition graph to declare or
maintain. Changing a type's states is a change to kira itself, not to config.

### Categories

Every state has a `category`, one of `todo`, `doing`, or `done`. Categories
are what tooling reasons about:

| Category | Used for |
| --- | --- |
| `todo` | Backlog. Not started. |
| `doing` | In flight. Feeds cycle-time metrics. |
| `done` | Closed, for any reason — including `WONT_DO`. |

`kira list --category doing` works regardless of what the states are called.
Note that `done` means "no longer open", not "successfully completed" — which
is what the `dropped` resolution exists to disambiguate.

Two `done` states carry a default resolution, recorded when an item lands
there: `DONE` stamps `done`, `WONT_DO` stamps `dropped`.

### Moving between states

Any state may move to any other state of the item's type, in either
direction. Reopening a closed ticket, jumping `TODO` straight to `DONE`, or
skipping `REVIEW` are all ordinary moves, not overrides. The only membership
rule is that the target has to be one of the type's states:

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
  state's default resolution on the item if it has none; leaving the `done`
  category for a `todo` or `doing` state clears `resolution` again. Pass
  `--resolution` to record something other than the state's default.

### WIP limits

`IN_PROGRESS` is capped at 3 tickets and `REVIEW` at 2. The caps are advisory:
a move that exceeds one warns on stderr and still happens.

```console
$ kira move DEMO-3 IN_PROGRESS
kira: warning: IN_PROGRESS is over its WIP limit (4 > 3)
Moved DEMO-3: TODO -> IN_PROGRESS
```

WIP limits are a process signal, not a correctness constraint; `kira doctor`
reports a state that is over its cap.

### `--force`

`--force` on `move` bypasses the blockers-closed check. On `edit` and `create`
it additionally accepts out-of-vocabulary field values. It is loud about it:

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
else, or importing history from another tracker. It is not needed to move
backwards or to skip a state — those are ordinary moves.

## Labels, priorities, subtypes, and resolutions

These four are **vocabulary fields**: each has a set of accepted values, and
kira validates against them.

`labels` and `people` are declared per repository in config, as `known` plus
`strict`:

```yaml
labels:
  known: [captured]
  strict: false

people:
  known: []
  strict: false
```

`priorities`, `subtypes`, and `resolutions` are built into kira:

| Field | Values |
| --- | --- |
| `priority` | `P0`, `P1`, `P2`, `P3` (highest first) |
| `subtype` | `bug`, `task`, `feature`, `support` |
| `resolution` | `done`, `dropped`, `duplicate`, `cannot-reproduce` |

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

An empty `labels.known` or `people.known` means nothing is known yet, so with
`strict: false` every value only warns.

`priority`, `subtype`, and `resolution` have no `strict` of their own: they
follow `labels.strict`. Setting `labels.strict: true` therefore tightens all
four at once.

### Ordering

Priorities are ordered, highest first. That order is not just cosmetic — it
defines what `ORDER BY priority` sorts by and what `>` / `<` comparisons in a
query expression mean.

### The `dropped` resolution

An item resolved `dropped` is closed — it is out of the backlog and off the
board — but it does not count as delivered work: it is excluded from
throughput and cycle time and from epic progress. Without this distinction,
cancelling enough tickets would make an epic look finished. `WONT_DO` stamps
`dropped` by default.

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
$ kira link DEMO-1 --parent DEMO-4
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

This is what makes renumbering safe. A merge can change `DEMO-3` into `DEMO-9`; because no other file ever wrote down the string
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
| `workon` | Branch naming pattern, `worktree` and `worktree_dir`. |
| `commit` | `subject` only — a personal commit-subject override. |

Anything else in the user file is ignored with a warning naming the offending
key as repo-authoritative.

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

## Built-in item shape

Item types, their states, and the `priority`/`subtype`/`resolution`
vocabularies are part of kira, not of a repository's config. An item file may
carry only the fields kira knows: an unknown frontmatter key, an unknown link
type under `links:`, or an unknown `type` is a parse error. Commands skip such
a file with a note, and `kira doctor` reports it as a `schema` error.

Besides the frontmatter fields, two Markdown body sections are addressable as
fields: `description` (`## Description`) and `acceptance`
(`## Acceptance criteria`). `--field key=value` on `create` and `edit` sets
any of them, and names what it accepts when you misspell one:

```console
$ kira edit DEMO-6 --field severity=high
kira: invalid item: --field: unknown or immutable field "severity"
  hint: editable fields: acceptance, description, due, estimate, labels, owner, parent, priority, rank, reporter, resolution, sprint, state, subtype, title
```
