# Item lifecycle commands

`create`, `edit`, `move`, `link`, `label`, `assign`, `comment` — the commands
that write items. Global flags: [Setup and schema
commands](setup-and-schema-commands.md#global-flags).

Samples: project `DEMO`, labels `perf, backend, docs`, people `alice, bob,
demo` (`demo` mapped to the git user), sprint `S1`.

## Conventions

- **Commits.** Under `commit.mode: auto` each mutation is committed at once,
  subject `<NUMBER> <verb> …` (`DEMO-3 edit priority`); under `manual` it is
  only staged. See [`kira commit`](git-integration.md#kira-commit).
- **Bulk.** `edit`, `move`, `assign`, `label add`, `label rm` take several ids;
  each is applied and reported separately. `create`, `link`, `comment` take one.
- **No-op.** A mutation that changes nothing prints `<NUMBER>: no changes`.
- **`--force`** accepts an out-of-vocabulary value under `strict: true`, and on
  `move` (or `edit` changing state) passes the blocker guard. It never admits an
  unknown field. Rules: [Concepts](../03-concepts.md#vocabularies).
- **`@me`** in `--owner`/`--reporter`/`assign` resolves the git identity
  through `people` (`git:` aliases), else the git user name with spaces
  replaced by `-`.

## `create`

```
kira create <type> [<title>] [flags]
```

`<type>` is `ticket` or `epic`. The title is positional or `--title`, not both.
Content comes from one of:

| Mode | Trigger |
| --- | --- |
| Editor | default — opens `$EDITOR` on the template |
| Flags only | `--no-edit` (required in scripts; otherwise `create` waits on an editor) |
| File | `--from-file <path>`, `-` for stdin |

| Flag | Description |
| --- | --- |
| `--title` | Title. |
| `--board` | Board key; default is the `default: true` board. |
| `--parent` | Epic to set as parent. |
| `--subtype`, `--priority` | Built-in vocabularies. |
| `--owner`, `--reporter` | Person; `@me` allowed. |
| `--label` | Label, repeatable, additive. |
| `--sprint` | Sprint key. |
| `--due` | Date, `YYYY-MM-DD`. |
| `--estimate` | Number, in `estimate.unit`. |
| `--rank` | Lexicographic grooming key. |
| `--field key=value` | Any editable field, repeatable (see below). |
| `--from-file` | Read a complete item; flags override the file's values. |
| `--here` | Inherit the active ticket's parent and sprint, add label `captured`. Requires `--no-edit` and a title. |
| `--blocking` | With `--here`: add the new item to the active ticket's `blocked_by`. |
| `--force` | Accept out-of-vocabulary values. |
| `--print-template` | Print the resolved template (with flags applied) and exit. |
| `--no-edit` | Build from flags only. |

### Templates

The editor buffer comes from `.kira/templates/<type>.<subtype>.md`, else
`.kira/templates/<type>.md`, else the built-in default. `kira init` writes no
templates.

```console
$ kira create ticket --print-template
---
title: 
type: ticket
subtype: 
priority: 
rank: 
owner: 
reporter: 
labels: []
parent: 
sprint: 
due: 
estimate: 
---

## Description

## Acceptance criteria
$ kira create ticket --subtype bug --print-template
---
title: 
type: ticket
subtype: bug
priority: P1
rank: 
owner: 
reporter: 
labels: []
parent: 
sprint: 
due: 
estimate: 
---

## Steps to reproduce

## Expected
```

The second run used a `.kira/templates/ticket.bug.md` with its own sections.

### Examples

```console
$ kira create ticket "Cache warmup is slow" --no-edit --subtype bug --priority P1 --owner alice --label perf --label backend --parent DEMO-1
Created DEMO-2 Cache warmup is slow
$ kira create ticket "Tidy the makefile" --no-edit --label tooling
kira: warning: field "labels": value "tooling" is not in the "label" vocabulary
Created DEMO-4 Tidy the makefile
$ cat draft.md
---
title: "Timeout is too aggressive"
type: ticket
subtype: bug
priority: P0
owner: bob
labels: [backend]
---

## Description

Clients give up after 2s.
$ kira create ticket --from-file draft.md --priority P2
Created DEMO-5 Timeout is too aggressive
$ cat draft.md | kira create ticket --from-file -
Created DEMO-6 Timeout is too aggressive
```

`id`, `number` and `state` in the file are ignored — the item gets a fresh id,
the next number and the type's initial state. An unknown key is rejected:

```console
$ kira create ticket --from-file d3.md
kira: --from-file: invalid item: field "severity": unknown frontmatter key
```

### `--here` / `--blocking`

```console
$ kira move DEMO-2 IN_PROGRESS --activate
Moved DEMO-2: TODO -> IN_PROGRESS
Activated DEMO-2
$ kira create ticket --no-edit --title "Pick a histogram library" --here
Created DEMO-7 Pick a histogram library
$ kira create ticket --no-edit --title "Fix flaky metrics test" --here --blocking
Created DEMO-8 Fix flaky metrics test
$ kira show DEMO-2
DEMO-2  Cache warmup is slow  [IN_PROGRESS]
id:         01M7ANMJ6710KXGV8WQGECTMBP
type:       ticket
subtype:    bug
category:   doing
priority:   P1
owner:      alice
labels:     perf, backend
parent:     DEMO-1
blocked_by: DEMO-8
created:    2026-11-13T20:10:26+05:30
updated:    2026-11-13T20:10:26+05:30


## Description

## Acceptance criteria
$ kira create ticket --no-edit --title Y --blocking
kira: --blocking requires --here
$ kira create ticket --no-edit --here
kira: --here requires --no-edit and --title
```

The active ticket is the one `kira now` reports.

### `--field`

Accepts exactly the editable fields; `description` and `acceptance` address
the body sections.

```console
$ kira create ticket --title "X" --field severity=high --no-edit
kira: invalid item: --field: unknown or immutable field "severity"
  hint: editable fields: acceptance, description, due, estimate, labels, owner, parent, priority, rank, reporter, resolution, sprint, state, subtype, title
```

## `edit`

```
kira edit <id>... [flags]
```

No field flags and one id: opens the whole file in `$EDITOR`. Any field flag:
applies exactly those changes, leaves the rest (body included) byte-identical,
and works across ids.

| Flag | Description |
| --- | --- |
| `--title`, `--owner`, `--priority`, `--subtype`, `--sprint`, `--rank`, `--due` | Set the field; `""` clears it (not `--title`). |
| `--label` | **Replace** the label set, repeatable. Use `label add`/`rm` to change one. |
| `--resolution` | Set resolution directly — a correction; `move` normally sets it. |
| `--field key=value` | Any editable field, repeatable; same set as `create`. |
| `--from-file` | Replace the item with a full file, or `-` for stdin. One id. |
| `--force` | Accept out-of-vocabulary values; pass the blocker guard when `state` changes. |

```console
$ kira edit DEMO-3 --sprint S1 --due 2026-12-01 --rank m
Edited DEMO-3: rank, sprint, due
$ kira edit DEMO-3 DEMO-5 --priority P3
Edited DEMO-3: priority
Edited DEMO-5: priority
$ kira edit DEMO-3 --owner ""
Edited DEMO-3: owner
$ kira edit DEMO-3 --field description="Cap retries at 3 per request."
Edited DEMO-3: body
$ kira edit DEMO-3 DEMO-5
kira: editing multiple ids requires field flags; editor mode is single-id only
```

## `move`

```
kira move <id>... <state> [flags]
```

The last argument is the target state; it must be one of the type's states.
State rules (blocker guard, resolution stamping, WIP warnings):
[Concepts](../03-concepts.md#workflow-states-and-categories).

| Flag | Description |
| --- | --- |
| `--resolution` | Resolution to record instead of the state's default. |
| `--activate` | Also make the item the active ticket (`kira now`, `create --here`). |
| `--force` | Pass the blocker guard (warns). |

```console
$ kira move DEMO-3 DEMO-5 IN_PROGRESS
Moved DEMO-3: TODO -> IN_PROGRESS
Moved DEMO-5: TODO -> IN_PROGRESS
$ kira move DEMO-5 WONT_DO --resolution duplicate
Moved DEMO-5: IN_PROGRESS -> WONT_DO
$ kira move DEMO-2 DONE
kira: invalid item: IN_PROGRESS -> DONE is blocked by open items: DEMO-8
  hint: close the blockers first, or use `--force` to override
$ kira move DEMO-2 DONE --force
kira: warning: forced past blocker guard: DEMO-8 still open
Moved DEMO-2: IN_PROGRESS -> DONE
```

## `link`

```
kira link <id> (--parent|--blocked-by|--relates|--duplicate-of) <id> [--remove]
```

Exactly one edge flag per call; `--remove` deletes that edge (for `--parent`,
clears it). Constraints: [Concepts](../03-concepts.md#relationships).

```console
$ kira link DEMO-3 --blocked-by DEMO-4
Linked DEMO-3
$ kira link DEMO-3 --blocked-by DEMO-4
DEMO-3: no changes
$ kira link DEMO-3 --blocked-by DEMO-4 --remove
Linked DEMO-3
$ kira link DEMO-6 --relates DEMO-3
Linked DEMO-6
$ kira link DEMO-6 --relates DEMO-3 --blocked-by DEMO-4
kira: give exactly one of --parent, --blocked-by, --relates, --duplicate-of
```

Typed links are stored under `links:`:

```yaml
links:
  relates: [01M7ANMJ7J9TVWZXC19ADRYMHR]
```

## `label`

```
kira label add <id>... <label> [--force]
kira label rm  <id>... <label> [--force]
kira label create <name>...
kira label list
```

`add`/`rm` change one label and keep the rest; the label is the **last**
argument. `create` registers names in `labels.known` (a config commit) without
touching items. `list` counts items per label, including registered-but-unused
ones.

```console
$ kira label add DEMO-3 DEMO-6 docs
Added docs to DEMO-3
Added docs to DEMO-6
$ kira label add DEMO-3 backend
DEMO-3: already labelled backend
$ kira label rm DEMO-3 backend
Removed backend from DEMO-3
$ kira label rm DEMO-3 perf
DEMO-3: not labelled perf
$ kira label add DEMO-3 DEMO-6
kira: "DEMO-6" resolves to an existing item, not a label
  hint: the last argument must be the label; ids come first
$ kira label create release ops
Created label release
Created label ops
$ kira label create release
Label release already exists
$ kira label list
LABEL     ITEMS
backend   3
captured  2
docs      1
ops       0
perf      1
release   0
tooling   1
```

## `assign`

```
kira assign <id> <user>
kira assign <id>... --owner <user> [--reporter] [--force]
```

`--reporter` sets the reporter instead of the owner. Several ids require
`--owner`.

```console
$ kira assign DEMO-3 bob
Assigned DEMO-3: bob
$ kira assign DEMO-3 DEMO-6 --owner alice
Assigned DEMO-3: alice
Assigned DEMO-6: alice
$ kira assign DEMO-3 DEMO-6 alice
kira: provide an owner: `assign <id> <user>` or `assign <id>... --owner <user>`
$ kira assign DEMO-3 DEMO-6
kira: "DEMO-6" resolves to an existing item, not an owner
  hint: did you mean `--owner` to assign multiple ids?
$ kira assign DEMO-3 --owner bob --reporter
Assigned DEMO-3: bob
$ kira assign DEMO-3 carol
kira: warning: field "owner": value "carol" is not in the "person" vocabulary
Assigned DEMO-3: carol
```

## `comment`

```
kira comment <id> [-m <text>]
```

Appends a comment to the item body; without `-m` opens `$EDITOR`. Existing
comments are never edited. Stored as an HTML-comment-delimited block with id,
author (the `people` name for the git user) and timestamp:

```console
$ kira comment DEMO-3 -m "Blocked on the cache instrumentation landing first."
Commented on DEMO-3
$ sed -n "/## Acceptance/,\$p" $(grep -l "number: DEMO-3" .kira/tickets/*.md)
## Acceptance criteria

<!-- kira:comment id=01M7ANMK4PYJGQQPFXM2JF5SYY author=demo ts=2026-11-13T20:10:27+05:30 -->
Blocked on the cache instrumentation landing first.
<!-- /kira:comment -->
```
