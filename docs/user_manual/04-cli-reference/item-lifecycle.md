# Item lifecycle commands

The commands on this page create and mutate items: `create`, `edit`, `move`,
`link`, `label`, `assign`, and `comment`. Everything else — listing, showing,
searching, syncing — is covered elsewhere in the CLI reference.

Every command here accepts the global flags `-C/--C <path>`, `--json`, and
`--no-color`; see the CLI reference index for what they do.

All examples below were run against a demo project whose key is `KIRA`, with a
label vocabulary of `perf, backend, flaky, docs`, a people roster of
`alice, bob`, and one sprint `2026-S30`.

## Conventions shared by these commands

**Auto-commit.** When the project is configured with `commit.mode: auto` (the
default), each mutation writes the item file and commits it immediately. The
commit subject describes the change:

```console
$ git log --oneline
6614371 kira: KIRA-10 link duplicate-of KIRA-5
409fef1 kira: KIRA-8 link relates KIRA-9
bc06f15 kira: KIRA-9 label add tooling
a4a2e0f kira: label create tooling,release
```

**Bulk forms.** `edit`, `move`, `assign`, `label add`, and `label rm` accept
several ids at once. Each id is applied independently and reported on its own
line. `create`, `link`, and `comment` are single-item commands.

**What `--force` means.** Every command on this page has a `--force` flag, and
it always means *accept a value the configuration would otherwise reject* —
a label, owner, priority, subtype, or resolution outside the configured
vocabulary. On `move` (and on `edit` when it changes state) it additionally
bypasses the workflow transition graph and `require:` guards. It never lets you
invent a new *field name*.

**Vocabulary: warning or error.** With `strict: false` (the default for both
labels and people) an out-of-vocabulary value is accepted with a warning on
stderr. With `strict: true` it is a hard error, and `--force` downgrades that
error back to a warning:

```console
$ kira create ticket "Tidy the makefile" --no-edit --label tooling      # strict: false
kira: warning: field "labels": value "tooling" is not in the "label" vocabulary
Created KIRA-10 Tidy the makefile

$ kira label add KIRA-8 tooling                                          # strict: true
kira: invalid item: field "labels": value "tooling" is not in the "label" vocabulary

$ kira label add KIRA-8 tooling --force
kira: warning: field "labels": value "tooling" is not in the "label" vocabulary
Added tooling to KIRA-8
```

---

## `create`

Creates a new item of a configured type (`ticket`, `epic`, or whatever types
your workflow declares) and assigns it the next display number. There are three
ways to supply the content, and choosing the right one is most of what you need
to know about this command:

1. **Interactive.** `kira create ticket` with no content flags opens `$EDITOR`
   on a template for that type. You fill in the frontmatter and body, save, and
   the item is created from what you wrote.
2. **Flags only.** `--no-edit` skips the editor entirely and builds the item
   from the flags you passed. This is the form to use in scripts and aliases.
3. **From a file.** `--from-file <path>` (or `-` for stdin) reads a complete
   item — frontmatter and body — from outside. Use this when another tool is
   generating tickets.

```
kira create <type> [<title>] [flags]
```

The title may be given positionally or with `--title`, but not both.

### Flags

| Flag | Type | Description |
| --- | --- | --- |
| `--board` | string | Board key to create the item under. |
| `--blocking` | bool | With `--here`, mark the new item as blocking the active ticket. Requires `--here`. |
| `--due` | string | Target completion date (RFC3339 date, e.g. `2026-08-05`). |
| `--estimate` | float | Estimate, in the unit configured under `estimate.unit`. |
| `--field` | string array | `key=value` for an item field, repeatable. See the note below on which keys are accepted. |
| `--force` | bool | Accept field values outside the configured vocabulary. |
| `--from-file` | string | Read a complete item from a file, or `-` for stdin. |
| `--here` | bool | Capture under the active ticket's epic and sprint. Requires `--no-edit` and `--title`. |
| `--label` | string array | Label, repeatable. |
| `--no-edit` | bool | Create from flags only, without opening `$EDITOR`. |
| `--owner` | string | Owner. |
| `--parent` | string | Epic to set as this item's parent. |
| `--print-template` | bool | Print the resolved template for the type and exit without creating anything. |
| `--priority` | string | Priority, validated against the configured priorities. |
| `--rank` | string | Lexicographic grooming key. |
| `--reporter` | string | Reporter. |
| `--sprint` | string | Sprint key, from the configured sprints. |
| `--subtype` | string | Item subtype (e.g. `bug`, `story`), validated against the configured subtypes. |

### Seeing the template before you write

`--print-template` prints exactly what interactive mode would put in front of
you. It is the quickest way to learn which fields and body sections a type
expects:

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
epic: 
sprint: 
due: 
estimate: 
---

## Description

## Acceptance criteria
```

Templates are per-type. `kira create epic --print-template` prints the epic
template, with `type: epic`.

### Creating from flags

```console
$ kira create ticket "Cache warmup is slow" --no-edit \
    --subtype bug --priority P1 --owner alice --label perf --label backend
Created KIRA-1 Cache warmup is slow

$ kira show KIRA-1
KIRA-1  Cache warmup is slow  [TODO]
id:        01KYHXAXTJWTKP7EAWX2E289J1
type:      ticket
subtype:   bug
category:  todo
priority:  P1
owner:     alice
labels:    perf, backend
created:   2026-07-27T19:18:23+05:30
updated:   2026-07-27T19:18:23+05:30


## Description

## Acceptance criteria
```

`--label` is repeatable and additive within a single `create` — the item above
ends up with both `perf` and `backend`.

> **Common mistake.** Forgetting `--no-edit` in a script. Without it, `create`
> tries to open `$EDITOR` and the script hangs waiting for an editor that never
> appears. Any non-interactive use of `create` wants either `--no-edit` or
> `--from-file`.

`--owner` and `--reporter` accept `@me`, which resolves to your git identity
the same way it does on `assign`, `edit`, and `list --owner`:

```console
$ kira create ticket "Retry budget is unbounded" --no-edit --owner @me
Created KIRA-12 Retry budget is unbounded

$ kira list --owner @me
NUMBER   STATE  TYPE    PRIORITY  TITLE
KIRA-12  TODO   ticket  -         Retry budget is unbounded
```

### Creating from a file or stdin

The file must be a complete item: YAML frontmatter followed by the body.

```console
$ cat new-ticket.md
---
title: "Retry budget is unbounded"
type: ticket
subtype: bug
priority: P0
owner: bob
labels: [backend]
---

## Description

Clients retry forever when the upstream returns 503.

## Acceptance criteria

- Retries capped at 3 per request

$ kira create ticket --from-file new-ticket.md
Created KIRA-8 Retry budget is unbounded
```

The same content can be piped in:

```console
$ generate-ticket.sh | kira create ticket --from-file -
Created KIRA-9 Timeout is too aggressive
```

You do not supply an `id`, `number`, or `state` — those are assigned on
creation.

Flags still apply alongside `--from-file`. The file plays the same role as the
template it replaces: it supplies the values, and an explicit flag overrides
them. This lets one stored draft be filed several ways:

```console
$ kira create ticket --from-file draft.md --parent KIRA-2 --priority P1
Created KIRA-10 Retry budget is unbounded
```

Here `--parent` and `--priority` are applied on top of the file, and would win
over a `priority:` the file set itself.

### Quick capture with `--here`

`--here` is for the interruption you want to write down without losing your
place. It copies the epic *and* the sprint from your active ticket (the one
`kira now` reports) onto the new item, and tags it with the built-in `captured`
label so quick captures are easy to find and groom later. The `captured` label
is always accepted regardless of vocabulary strictness.

```console
$ kira now
KIRA-3  Add p99 dashboard  [IN_PROGRESS]
in state:    just now
due:         2026-08-01
branch:      main
commits:     0 since last state change

$ kira create ticket --no-edit --title "Pick a histogram library" --here
Created KIRA-6 Pick a histogram library

$ kira show KIRA-6
KIRA-6  Pick a histogram library  [TODO]
id:        01KYHXDH38VVXE0X1PNEYPWJQP
type:      ticket
category:  todo
labels:    captured
epic:      01KYHXAXZCVGMH7TBAMAM5P63S
created:   2026-07-27T19:19:49+05:30
updated:   2026-07-27T19:19:49+05:30
```

Adding `--blocking` also records the new item as a blocker of the active
ticket, which is what you want when the interruption is something that has to
be resolved before the current work can finish:

```console
$ kira create ticket --no-edit --title "Fix flaky metrics test" --here --blocking
Created KIRA-7 Fix flaky metrics test

$ kira show KIRA-3
KIRA-3  Add p99 dashboard  [IN_PROGRESS]
...
blocked_by: 01KYHXEAMT0JW34BN5A9NCH6NZ
```

Both guards are enforced:

```console
$ kira create ticket --no-edit --title "Y" --blocking
kira: --blocking requires --here

$ kira create ticket --no-edit --here
kira: --here requires --no-edit and --title
```

### `--field` and which keys it accepts

`--field key=value` sets a field by name, and is repeatable. It accepts only
the item's editable fields — it is not an escape hatch for arbitrary
frontmatter, and `--force` does not change that. Pass an unrecognised key and
the error lists the full accepted set:

```console
$ kira create ticket "X" --no-edit --field severity=high
kira: invalid item: --field: unknown or immutable field "severity"
  hint: editable fields: acceptance, description, due, epic, estimate, labels,
        owner, priority, rank, reporter, resolution, sprint, state, subtype, title
```

For the fields that have a dedicated flag (`--priority`, `--owner`, …), prefer
the dedicated flag; `--field` is most useful for the ones that do not, such as
`description` and `acceptance`.

---

## `edit`

Changes fields on one or more existing items. Like `create`, it has an
interactive mode and a flag-driven mode, and the mode is chosen by whether you
pass any field flags.

- **No field flags, one id** — opens `$EDITOR` on the whole item, frontmatter
  and body together. Edit anything, save, and the item is rewritten. This is
  how you change the description or acceptance criteria in practice.
- **Any field flag** — skips the editor and applies exactly the changes named
  on the command line. This is the scriptable mode, and the only mode that
  works across multiple ids.

```
kira edit <id>... [flags]
```

### Flags

| Flag | Type | Description |
| --- | --- | --- |
| `--due` | string | Set due date (RFC3339 date; empty clears). |
| `--field` | string array | `key=value` edit, repeatable; skips `$EDITOR`. Same accepted key set as `create --field`. |
| `--force` | bool | Accept values outside the configured vocabulary, and bypass workflow guards on state changes. |
| `--from-file` | string | Round-trip an edited item file, or `-` for stdin. Single id only. |
| `--label` | string array | **Set** labels, repeatable — replaces the entire label set. |
| `--owner` | string | Set owner (empty clears). |
| `--priority` | string | Set priority (empty clears). |
| `--rank` | string | Set rank (empty clears). |
| `--resolution` | string | Set resolution directly. An out-of-band correction; normally `move` sets this. |
| `--sprint` | string | Set sprint key (empty clears). |
| `--subtype` | string | Set subtype (empty clears). |
| `--title` | string | Set title. |

Passing an empty string clears the field:

```console
$ kira edit KIRA-11 --owner ""
Edited KIRA-11: owner
```

### Examples

Setting several fields at once; the output names exactly which fields changed:

```console
$ kira edit KIRA-3 --sprint 2026-S30 --due 2026-08-01 --rank m
Edited KIRA-3: rank, sprint, due
```

Bulk edit across ids:

```console
$ kira edit KIRA-8 KIRA-9 --priority P2
Edited KIRA-8: priority
Edited KIRA-9: priority
```

Editor mode is single-id only, so a bulk invocation with no field flags is
rejected rather than silently opening an editor per item:

```console
$ kira edit KIRA-8 KIRA-9
kira: editing multiple ids requires field flags; editor mode is single-id only
```

### `--field` versus editor mode

`--field x=y` and editor mode reach the same fields by different routes.
`--field` is surgical and non-interactive: it changes exactly the keys you name
and leaves everything else, including the body, byte-identical. Editor mode
gives you the whole file and accepts whatever you save — convenient for prose,
risky for scripts. Use `--field` (or the dedicated flags) in anything
automated.

### `--from-file`

Round-trips a complete item file back into the store. The usual workflow is to
export or copy the item, transform it with a tool, and feed it back:

```console
$ kira edit KIRA-8 --from-file edited.md
Edited KIRA-8: priority
```

`--from-file` cannot be combined with multiple ids.

### `--resolution` is a correction, not the normal path

Resolution is normally recorded as part of a `move` into a done-category state,
where the workflow can validate and require it. `edit --resolution` writes the
field directly, bypassing that. Reach for it when a resolution was recorded
wrongly or when an item arrived in a done state through an import, not as your
routine way to close work.

---

## `move`

Transitions one or more items to a new state. The final positional argument is
the target state; everything before it is an id.

```
kira move <id>... <state> [flags]
```

Moves are validated against the workflow declared in the project configuration:
the transition must be adjacent in the graph, and any `require:` guard on that
edge must be satisfied.

### Flags

| Flag | Type | Description |
| --- | --- | --- |
| `--activate` | bool | Set this item as the active ticket — what `kira now` reports and what `create --here` inherits from. |
| `--force` | bool | Bypass the transition adjacency check and `require:` guards. |
| `--resolution` | string | Resolution to record, validated against the configured resolutions. |

### A full transition sequence

```console
$ kira move KIRA-1 IN_PROGRESS --activate
Moved KIRA-1: TODO -> IN_PROGRESS
Activated KIRA-1
```

An off-graph jump is refused, and the error tells you what *is* reachable:

```console
$ kira move KIRA-1 DONE
kira: invalid item: IN_PROGRESS -> DONE is not an allowed transition
  hint: allowed from IN_PROGRESS: REVIEW, TODO, WONT_DO (or `--force` to override)
```

### Hitting a `require:` guard

The demo project's workflow declares `REVIEW -> DONE` as
`{ to: DONE, require: [resolution] }`. Taking that edge without a resolution
fails:

```console
$ kira move KIRA-1 REVIEW
Moved KIRA-1: IN_PROGRESS -> REVIEW

$ kira move KIRA-1 DONE
kira: invalid item: REVIEW -> DONE requires resolution to be set
  hint: set resolution first, or use `--force` to override

$ kira move KIRA-1 DONE --resolution done
Moved KIRA-1: REVIEW -> DONE
```

The resolution lands in the item's frontmatter:

```yaml
state: DONE
resolution: done
```

### Bulk moves and `--force`

```console
$ kira move KIRA-8 KIRA-9 IN_PROGRESS
Moved KIRA-8: TODO -> IN_PROGRESS
Moved KIRA-9: TODO -> IN_PROGRESS
```

`--force` takes an edge the graph does not contain. It succeeds, but warns, so
the override is visible in the terminal and in review:

```console
$ kira move KIRA-11 DONE --force
kira: warning: forced off-graph transition TODO -> DONE
Moved KIRA-11: TODO -> DONE
```

---

## `link`

Creates or removes a relationship between two items: an epic parent, a
blocked-by dependency, or a typed link (`relates`, `duplicate_of`). Exactly one
edge flag must be given per invocation — the command names the edge, the
`--remove` flag decides the direction of the operation.

```
kira link <id> [flags]
```

### Flags

| Flag | Type | Description |
| --- | --- | --- |
| `--blocked-by` | string | Blocking item to add, or remove with `--remove`. |
| `--duplicate-of` | string | Item to add to `links.duplicate_of`, or remove with `--remove`. |
| `--epic` | string | Epic parent to set, or clear with `--remove`. |
| `--relates` | string | Item to add to `links.relates`, or remove with `--remove`. |
| `--force` | bool | Accept field values outside the configured vocabulary. |
| `--remove` | bool | Remove the given edge instead of adding it. |

### Adding and removing a dependency

```console
$ kira link KIRA-3 --epic KIRA-2
Linked KIRA-3

$ kira link KIRA-3 --blocked-by KIRA-4
Linked KIRA-3

$ kira show KIRA-3
KIRA-3  Add p99 dashboard  [TODO]
id:        01KYHXAY46955FEJA2DBJMMWGG
type:      ticket
subtype:   task
category:  todo
priority:  P2
epic:      01KYHXAXZCVGMH7TBAMAM5P63S
blocked_by: 01KYHXBXHFA127VGH85GVV8ZXD
created:   2026-07-27T19:18:24+05:30
updated:   2026-07-27T19:18:56+05:30
```

Removing is the same command plus `--remove` — you still name which edge you
mean and which item is on the other end:

```console
$ kira link KIRA-3 --blocked-by KIRA-4 --remove
Linked KIRA-3

$ kira show KIRA-3
KIRA-3  Add p99 dashboard  [TODO]
id:        01KYHXAY46955FEJA2DBJMMWGG
type:      ticket
subtype:   task
category:  todo
priority:  P2
epic:      01KYHXAXZCVGMH7TBAMAM5P63S
created:   2026-07-27T19:18:24+05:30
updated:   2026-07-27T19:18:56+05:30
```

### Typed links

`--relates` and `--duplicate-of` write into a `links:` map in the frontmatter
rather than a top-level field:

```console
$ kira link KIRA-8 --relates KIRA-9
Linked KIRA-8

$ kira link KIRA-10 --duplicate-of KIRA-5
Linked KIRA-10
```

```yaml
links:
  relates: [01KYHXEMZJCTDJ3C3R620ZW305]
```

### One edge at a time

Passing zero or several edge flags is an error, in both cases with the same
message:

```console
$ kira link KIRA-8 --relates KIRA-9 --blocked-by KIRA-5
kira: give exactly one of --epic, --blocked-by, --relates, --duplicate-of
```

When a link makes no change — adding an edge that already exists — the command
reports `KIRA-n: no changes` instead of `Linked KIRA-n`.

---

## `label`

Manages both the project's label vocabulary and the labels on individual items.
It has four subcommands: `add` and `rm` change item labels, `create` registers
names in the vocabulary, and `list` reports usage.

> ### The one thing to get right: replace versus add
>
> There are two ways to change an item's labels, and they do different things.
>
> | Command | Effect on an item labelled `perf, backend` |
> | --- | --- |
> | `kira edit KIRA-1 --label docs` | Labels become `docs`. The set is **replaced**. |
> | `kira label add KIRA-1 docs` | Labels become `perf, backend, docs`. **Additive**. |
> | `kira label rm KIRA-1 perf` | Labels become `backend`. **Subtractive**. |
>
> Verified end to end:
>
> ```console
> $ kira show KIRA-1 | grep labels
> labels:    perf, backend
>
> $ kira edit KIRA-1 --label docs
> Edited KIRA-1: labels
>
> $ kira show KIRA-1 | grep labels
> labels:    docs
> ```
>
> The two original labels are gone. `edit --label` is a *set* operation: to keep
> the existing labels you must relist all of them. If you only mean "also tag
> this with `docs`", use `label add`.

### `label add` / `label rm`

```
kira label add <id>... <label> [--force]
kira label rm  <id>... <label> [--force]
```

The label is always the **last** argument; every argument before it is an id.

| Flag | Type | Description |
| --- | --- | --- |
| `--force` | bool | Accept a label outside the configured vocabulary. |

```console
$ kira label add KIRA-1 perf
Added perf to KIRA-1

$ kira label add KIRA-8 KIRA-9 KIRA-10 release
Added release to KIRA-8
Added release to KIRA-9
Added release to KIRA-10

$ kira label rm KIRA-1 docs
Removed docs from KIRA-1
```

Both are idempotent, and say so rather than failing:

```console
$ kira label add KIRA-8 release
KIRA-8: already labelled release

$ kira label rm KIRA-8 docs
KIRA-8: not labelled docs
```

If the last argument happens to resolve to an existing item, the command stops
rather than creating a label named after a ticket:

```console
$ kira label add KIRA-8 KIRA-9
kira: "KIRA-9" resolves to an existing item, not a label
  hint: the last argument must be the label; ids come first
```

### `label create`

Registers one or more names in the configuration's label vocabulary. It touches
no items — it only makes the names known, so that a strict vocabulary will
accept them and so that `label list` shows them at zero usage.

```console
$ kira label create tooling release
Created label tooling
Created label release

$ kira label add KIRA-9 tooling
Added tooling to KIRA-9
```

Re-registering an existing label is harmless:

```console
$ kira label create tooling
Label tooling already exists
```

### `label list`

Lists every label with the number of items carrying it. Registered-but-unused
labels appear with a count of `0`.

```console
$ kira label list
LABEL     ITEMS
backend   3
captured  2
docs      0
flaky     1
perf      1
release   0
tooling   3
```

---

## `assign`

Sets the owner — or, with `--reporter`, the reporter — of one or more items.
It is a convenience wrapper over what `edit --owner` would do, with a shorter
argument form for the common single-item case.

```
kira assign <id>... [<user>] [flags]
```

### Flags

| Flag | Type | Description |
| --- | --- | --- |
| `--force` | bool | Accept a person outside the configured roster. |
| `--owner` | string | Owner to assign. Required when assigning more than one id. |
| `--reporter` | bool | Assign the reporter field instead of owner. |

### The two argument forms

The positional form takes exactly one id and one user:

```console
$ kira assign KIRA-3 bob
Assigned KIRA-3: bob
```

For several ids, the user must move to `--owner`, because a bare trailing
argument would be ambiguous with another id:

```console
$ kira assign KIRA-3 KIRA-4 --owner alice
Assigned KIRA-3: alice
Assigned KIRA-4: alice
```

Getting this wrong produces a specific error rather than a confusing
assignment. Three positional arguments are not accepted:

```console
$ kira assign KIRA-3 KIRA-4 alice
kira: provide an owner: `assign <id> <user>` or `assign <id>... --owner <user>`
```

…and two ids with no `--owner` are caught before anything is written, because
the second one resolves to a real item:

```console
$ kira assign KIRA-8 KIRA-9
kira: "KIRA-9" resolves to an existing item, not an owner
  hint: did you mean `--owner` to assign multiple ids?
```

### Assigning the reporter

`--reporter` redirects the whole command at the reporter field; the owner is
left alone.

```console
$ kira assign KIRA-3 --owner bob --reporter
Assigned KIRA-3: bob
```

```yaml
owner: alice
reporter: bob
```

---

## `comment`

Appends a timestamped, attributed comment to the end of an item's body. Works
on tickets and epics alike. Comments are appended, never replaced — the command
does not edit or delete existing ones.

```
kira comment <id> [flags]
```

### Flags

| Flag | Type | Description |
| --- | --- | --- |
| `-m`, `--message` | string | Comment text. Opens `$EDITOR` when omitted. |

### Example

```console
$ kira comment KIRA-3 -m "Blocked on the cache instrumentation landing first."
Commented on KIRA-3
```

The comment is stored in the item file below the body sections, wrapped in
marker comments that carry its own id, the author taken from your git identity,
and the timestamp:

```markdown
---
id: 01KYHXAY46955FEJA2DBJMMWGG
number: KIRA-3
aliases: []
type: ticket
subtype: task
title: "Add p99 dashboard"
state: TODO
priority: P2
owner: alice
reporter: bob
labels: []
epic: 01KYHXAXZCVGMH7TBAMAM5P63S
blocked_by: []
created: 2026-07-27T19:18:24+05:30
updated: 2026-07-27T19:19:06+05:30
---

## Description

## Acceptance criteria

<!-- kira:comment id=01KYHXC7JA0KMF1GQZXC09T9RC author=Demo-Dev ts=2026-07-27T19:19:06+05:30 -->
Blocked on the cache instrumentation landing first.
<!-- /kira:comment -->
```

Because the markers are HTML comments, the file still renders cleanly as
Markdown anywhere else — a code host's file preview, a static site generator —
while remaining unambiguously parseable back into structured comments.

Omitting `-m` opens `$EDITOR` for a longer comment. As with `create` and
`edit`, that form is unsuitable for scripts; always pass `-m` there.
