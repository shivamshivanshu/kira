# Concepts

The model behind every command. `kira config explain` prints the effective
rules for a repository; its output is described in the
[CLI reference](04-cli-reference/setup-and-schema-commands.md#kira-config-explain).

## Items and types

An **item** is one Markdown file under `.kira/tickets/`, named `<id>.md`:
YAML frontmatter, then a body. Types are built into kira:

| Type | Role |
| --- | --- |
| `ticket` | A unit of work. |
| `epic` | A container. Items point at it through their `parent` field; epics may nest. |

Any item may have one `parent`, which must be an epic. Epic progress (done
leaves / all leaves, counted through nested epics) is computed from the
children, not stored. `kira tree` renders the hierarchy.

Frontmatter may carry only fields kira knows. An unknown key, an unknown link
type under `links:`, or an unknown `type` is a parse error: commands skip the
file with a note and `kira doctor` reports it as a `schema` error. The legacy
key `epic:` is read as `parent:`. Two body sections are addressable as fields:
`description` (`## Description`) and `acceptance` (`## Acceptance criteria`).

## Identity: `id` and `number`

| Field | Example | Properties |
| --- | --- | --- |
| `id` | `01M7AN57WYK87HJKQZYZQSQT5D` | ULID. Immutable, unique, the filename. Stored in every cross-reference. |
| `number` | `DEMO-3` | Display number `<board key>-<n>`. Sequential per board, **renumberable**. |

Commands accept a number, an alias, or an id (prefix). References are written
as ids, so renumbering never breaks a link.

Numbers are allocated locally, so two branches can mint the same one. After the
merge, reconciliation (run by `kira sync` and the `post-merge` hook)
deterministically renumbers one of them; `kira doctor` reports collisions not
yet repaired. A renumbered item keeps its old number in `aliases`, which still
resolves — but a commit message that said `DEMO-7` may now name a different
item.

## Boards

A board is a numbering namespace declared in `.kira/config.yaml`
([keys](05-configuration-reference.md#boards)). Every item belongs to exactly
one board; `kira create` uses the `default: true` board unless given
`--board`. `kira board move` renumbers an item onto another board and records
the old number as an alias. Archiving a board hides it from pickers and
`--board`, but its items still list and resolve. Boards are config entries,
not items — see [Limitations](09-limitations-and-roadmap.md).

## Workflow, states and categories

Each type has a fixed state set:

| Type | States (category) | Initial |
| --- | --- | --- |
| `ticket` | `TODO` (todo), `IN_PROGRESS` (doing, WIP 3), `REVIEW` (doing, WIP 2), `DONE` (done), `WONT_DO` (done) | `TODO` |
| `epic` | `PLANNED` (todo), `ACTIVE` (doing), `DONE` (done) | `PLANNED` |

`state` is a plain enum: any state may move to any other state of the type,
in either direction. A state outside the set is refused:

```console
$ kira move DEMO-5 SHIPPED
kira: invalid item: "SHIPPED" is not a state in the ticket workflow
  hint: valid states: TODO, IN_PROGRESS, REVIEW, DONE, WONT_DO
```

Categories (`todo`, `doing`, `done`) are what filters and metrics use.
`done` means "closed for any reason", `WONT_DO` included.

Rules around the `done` category:

- **Blocker guard.** Entering a `done` state is refused while any
  `blocked_by` target is not in a `done` state (a dangling blocker only warns).
  `--force` overrides and warns:

  ```console
  $ kira move DEMO-5 DONE
  kira: invalid item: TODO -> DONE is blocked by open items: DEMO-2
    hint: close the blockers first, or use `--force` to override
  $ kira move DEMO-5 DONE --force
  kira: warning: forced past blocker guard: DEMO-2 still open
  Moved DEMO-5: TODO -> DONE
  ```

- **Resolution.** Entering `DONE` stamps `resolution: done`, `WONT_DO` stamps
  `dropped`, unless the item already has one or `--resolution` gives another.
  Leaving the `done` category clears it.
- **WIP limits** are advisory: exceeding one warns and proceeds, and
  `kira doctor` reports the over-limit state.

  ```console
  $ kira move DEMO-3 DEMO-4 DEMO-6 DEMO-7 IN_PROGRESS
  Moved DEMO-3: TODO -> IN_PROGRESS
  Moved DEMO-4: TODO -> IN_PROGRESS
  Moved DEMO-6: TODO -> IN_PROGRESS
  kira: warning: IN_PROGRESS is over its WIP limit (4 > 3)
  Moved DEMO-7: TODO -> IN_PROGRESS
  ```

### `dropped`

A `dropped` resolution marks closed-but-not-delivered work. `kira stats`
counts it separately from completion and excludes it from cycle time, lead
time and throughput. Epic progress does not distinguish it: every `done`-state
child counts as done.

## Vocabularies

| Field | Values | Declared in |
| --- | --- | --- |
| `labels` | `labels.known` | config |
| `owner`, `reporter` | `people.known` | config |
| `priority` | `P0`, `P1`, `P2`, `P3` (most urgent first) | kira |
| `subtype` | `bug`, `task`, `feature`, `support` | kira |
| `resolution` | `done`, `dropped`, `duplicate`, `cannot-reproduce` | kira |

`people.strict` governs owner/reporter; `labels.strict` governs labels **and**
the three built-in vocabularies. Out-of-vocabulary value:

| `strict` | Effect |
| --- | --- |
| `false` (default) | Warning on stderr; value written. |
| `true` | Error. `--force` downgrades it to a warning. |

```console
$ kira edit DEMO-6 --label nope
kira: invalid item: field "labels": value "nope" is not in the "label" vocabulary
$ kira edit DEMO-6 --label nope --force
kira: warning: field "labels": value "nope" is not in the "label" vocabulary
Edited DEMO-6: labels
$ kira edit DEMO-7 --priority P9
kira: invalid item: field "priority": value "P9" is not in the "priority" vocabulary
```

The system label `captured` (added by `create --here`) is always accepted.
Priority order defines `ORDER BY priority` and `<`/`>` in queries.

## Sprints

A sprint is a keyed date range in config, appended by `kira sprint create`
and committed like any config change. An item joins one through its `sprint`
field, validated against the declared keys.

The **active sprint** is a per-clone pointer in `.kira/.cache/active-sprint`
(git-ignored), set by `kira sprint activate`. The sprint list is shared; which
one you are looking at is not, so activating changes nothing in `git status`.
`active` is accepted wherever a sprint key is (`list --sprint active`,
`stats --sprint active`, `sprint = active` in queries). The active-ticket
pointer behind `kira now` / `kira workon` is local in the same way.

## Relationships

| Relationship | Field | Cardinality | Constraint |
| --- | --- | --- | --- |
| Parent | `parent` | one | Target must be an epic; no cycles. |
| Dependency | `blocked_by` | list | Any item; no cycles. Drives the blocker guard and the `blocked` query term. |
| Typed link | `links.relates`, `links.duplicate_of` | list each | None. |

All are set and removed with [`kira link`](04-cli-reference/item-lifecycle.md#link).
A dependency is stored only on the dependent item; `blocks` in
`kira show --json` is derived. On disk every reference is an id:

```yaml
parent: 01M7AN8600NWVDEGJ04K861BT9
blocked_by: [01M7AN860Q0P9Y5WRWKHJ8RSZS]
```

## Commits

Under the default `commit.mode: auto`, each mutation is its own commit, so an
item's history is `git log` (`kira log <id>` reads it back per field). Code
commits link to items through a `Kira-Ticket:` trailer, a `[[KEY-n]]` marker or
a leading `KEY-n` in the subject; `Kira-Closes:` closes the item when the
commit lands. See [Git integration](04-cli-reference/git-integration.md).

## Configuration tiers

Built-in defaults, then the committed repo file `.kira/config.yaml`, then a
personal file `~/.config/kira/config.yaml` limited to `ui`, `workon` and
`commit.subject`. On a collision the repo wins, except `ui.editor`, which is
read only from the personal file. Details:
[Configuration reference](05-configuration-reference.md#two-config-layers).
