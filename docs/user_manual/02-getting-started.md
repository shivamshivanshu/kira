# Getting Started

A walkthrough from an empty repository to a ticket marked done. Every
transcript below is real output; if yours differs, trust yours and read
`--help`.

## Install

Prebuilt archives for linux and macOS (amd64 and arm64) are published on the
GitHub Releases page. Download the archive for your platform, extract it, and
put the `kira` binary somewhere on your `PATH`.

If you have a Go toolchain:

```console
$ go install github.com/shivamshivanshu/kira/cmd/kira@latest
```

Check it works:

```console
$ kira version
dev
```

## 1. Initialize a repository

`kira init` needs to run inside a git repository — it writes into `.kira/` and
commits the result.

```console
$ kira init --help
Initialize a .kira/ store in the current git repository

Usage:
  kira init [flags]

Flags:
      --force        reinitialize over an existing .kira/
  -h, --help         help for init
      --key string   project key for display IDs (default: derived from directory name)
```

The `--key` is the prefix your tickets will be numbered with. Without it, kira
derives one from the directory name — a directory called `kira-docs-demo`
becomes the key `KIRADOCSDEMO`, which is rarely what you want, so pass `--key`
explicitly.

```console
$ kira init --key DEMO
Initialized kira in .kira (project key DEMO)
```

That creates:

| Path | Contents |
| --- | --- |
| `.kira/config.yaml` | Project config: vocabularies, commit mode. Heavily commented; read it. |
| `.kira/tickets/` | One Markdown file per item. Empty to begin with. |
| `.kira/templates/` | `ticket.md`, `epic.md` — what `$EDITOR` opens when you create an item interactively. |
| `.kira/.gitignore` | Ignores `.cache/`, the disposable local index. |

`init` commits its own work:

```console
$ git log --oneline
f209bfd kira: init
1625437 initial commit
```

## 2. Install the git hooks

`init` does **not** install git hooks — that is a separate, explicit step, and
it is per clone rather than committed. Check first:

```console
$ kira hooks status
hook post-merge: missing
hook prepare-commit-msg: missing
hook pre-commit: missing
merge driver: not registered (run `kira hooks install`)
```

Then install:

```console
$ kira hooks install
hook post-merge: installed
hook prepare-commit-msg: installed
merge driver: registered
```

The merge driver is the important part: it makes ticket files merge field by
field instead of producing conflict markers. Every collaborator should run
`kira hooks install` once in their own clone. `kira hooks uninstall` reverses
it, restoring any hooks kira chained onto.

## 3. Create your first ticket

`--no-edit` creates the item from flags alone:

```console
$ kira create ticket "Write the getting-started guide" --no-edit
Created DEMO-1 Write the getting-started guide
```

Look at what landed on disk:

```console
$ cat .kira/tickets/01KYHXCBZ585RAAWEN27N0EDAN.md
---
id: 01KYHXCBZ585RAAWEN27N0EDAN
number: DEMO-1
aliases: []
type: ticket
title: "Write the getting-started guide"
state: TODO
labels: []
epic: null
blocked_by: []
created: 2026-07-27T19:19:11+05:30
updated: 2026-07-27T19:19:11+05:30
---

## Description

## Acceptance criteria
```

The filename is the immutable `id`; `DEMO-1` is only the human-facing display
number. See [Concepts](03-concepts.md) for why there are two.

The create was committed automatically:

```console
$ git log --oneline -1
08f8842 kira: DEMO-1 create "Write the getting-started guide"
```

### The interactive path

Omit `--no-edit` (and `--from-file`) and kira opens `$EDITOR` on a template
instead. You fill in the frontmatter and the body, save, and quit; kira parses
the buffer into an item. The template is `.kira/templates/ticket.md`, and you
can preview the resolved version without creating anything:

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

You can also pipe a complete item in: `kira create ticket --from-file -`.

### A couple more, with fields

```console
$ kira create ticket "Fix broken anchor links" --no-edit --priority P0 --subtype bug
Created DEMO-2 Fix broken anchor links

$ kira create ticket "Add search to the docs site" --no-edit --priority P2 --subtype feature
Created DEMO-3 Add search to the docs site
```

If you pass a value outside the vocabulary — say `--subtype story`, when the
subtypes are `bug, task, feature, support` — the default non-strict config
warns and accepts it:

```console
$ kira create ticket "Add search to the docs site" --no-edit --priority P1 --subtype story
kira: warning: field "subtype": value "story" is not in the "subtype" vocabulary
Created DEMO-2 Add search to the docs site
```

Set `labels.strict: true` and the same command fails instead. See
[Concepts](03-concepts.md#labels-priorities-subtypes-and-resolutions).

## 4. List

```console
$ kira list
NUMBER  STATE  TYPE    PRIORITY  TITLE
DEMO-2  TODO   ticket  P0        Fix broken anchor links
DEMO-3  TODO   ticket  P2        Add search to the docs site
DEMO-1  TODO   ticket  -         Write the getting-started guide
```

`kira list` takes filters — `--state`, `--priority`, `--label`, `--owner`
(`@me` resolves to your git identity), `--sprint`, `--parent`, `--type`,
`--category`, a full `--query` expression, or a saved `--filter` from
`.kira/config.yaml`. `--tree` groups the results under their epics.

## 5. Show one item

```console
$ kira show DEMO-2
DEMO-2  Fix broken anchor links  [TODO]
id:        01KYHXCJJ4K0M9PZ56KCE5E98C
type:      ticket
subtype:   bug
category:  todo
priority:  P0
created:   2026-07-27T19:19:17+05:30
updated:   2026-07-27T19:19:17+05:30


## Description

## Acceptance criteria
```

`kira show` prints a curated subset of fields — it deliberately omits `due`,
`sprint`, `estimate`, and `resolution` from the human view. For the complete
record, including linked commits and recent field history, use `--json`:

```console
$ kira show DEMO-3 --json
{
  "id": "01KYHXCJK0A3GJ2QWE1VRXV8F2",
  "number": "DEMO-3",
  "board": "DEMO",
  "type": "ticket",
  "subtype": "feature",
  "title": "Add search to the docs site",
  "state": "TODO",
  "category": "todo",
  "resolution": null,
  "priority": "P1",
  "sprint": null,
  "due": "2026-08-15",
  "owner": null,
  "labels": [],
  "epic": null,
  "blocked_by": [],
  "links": { "duplicate_of": [], "relates": [] },
  "estimate": null,
  "created": "2026-07-27T19:19:17+05:30",
  "updated": "2026-07-27T19:19:24+05:30",
  "linked_commits": [
    { "sha": "5c692a656662b27b3700961df8131148dc3f5468", "subject": "kira: DEMO-3 edit due", "author": "Demo User", "ts": "2026-07-27T19:19:24+05:30" },
    { "sha": "f1050467b04f86488751d75bb46fd266f293f6c0", "subject": "kira: DEMO-3 edit priority", "author": "Demo User", "ts": "2026-07-27T19:19:24+05:30" }
  ],
  "history_tail": [
    { "ts": "2026-07-27T19:19:24+05:30", "field": "due", "from": null, "to": "2026-08-15" },
    { "ts": "2026-07-27T19:19:24+05:30", "field": "priority", "from": "P2", "to": "P1" }
  ]
}
```

(Trimmed here for length; the real output includes every field.)

## 6. Edit a field without an editor

Common fields have dedicated flags:

```console
$ kira edit DEMO-3 --priority P1
Edited DEMO-3: priority
```

Any other editable field goes through `--field key=value`, which is
repeatable:

```console
$ kira edit DEMO-3 --field due=2026-08-15
Edited DEMO-3: due
```

`--field` only accepts fields kira knows. Misspell one and kira tells you what
is editable:

```console
$ kira edit DEMO-6 --field severity=high
kira: invalid item: --field: unknown or immutable field "severity"
  hint: editable fields: acceptance, description, due, estimate, labels, owner, parent, priority, rank, reporter, resolution, sprint, state, subtype, title
```

With no flags at all, `kira edit <id>` opens the item in `$EDITOR`.

## 7. Move through the workflow

```console
$ kira move DEMO-2 IN_PROGRESS
Moved DEMO-2: TODO -> IN_PROGRESS
```

`state` is a plain enum: any state may move to any other state of the type,
forwards or backwards. Only a state outside the type's workflow is refused:

```console
$ kira move DEMO-2 SHIPPED
kira: invalid item: "SHIPPED" is not a state in the ticket workflow
  hint: valid states: TODO, IN_PROGRESS, REVIEW, DONE, WONT_DO
```

So take the next step, or skip straight to the end — both work:

```console
$ kira move DEMO-2 REVIEW
Moved DEMO-2: IN_PROGRESS -> REVIEW

$ kira move DEMO-2 DONE
Moved DEMO-2: REVIEW -> DONE
```

Landing in a `done` state stamps that state's default resolution — `done`
for `DONE`, `dropped` for `WONT_DO` — unless you pass `--resolution` to record
something else. Moving back out of `done` clears it again.

The one thing a close does check is dependencies: a ticket with an open
`blocked_by` cannot enter a `done` state.

```console
$ kira move DEMO-2 DONE
kira: invalid item: REVIEW -> DONE is blocked by open items: DEMO-1
  hint: close the blockers first, or use `--force` to override
```

States can also carry advisory WIP limits. `IN_PROGRESS` has a limit of 3, so
a fourth item warns but proceeds:

```console
$ kira move DEMO-3 IN_PROGRESS
kira: warning: IN_PROGRESS is over its WIP limit (4 > 3)
Moved DEMO-3: TODO -> IN_PROGRESS
```

Every one of those moves is a commit:

```console
$ git log --oneline
0628f5c kira: DEMO-2 state REVIEW -> DONE
badf3af kira: DEMO-2 state IN_PROGRESS -> REVIEW
e76a029 kira: DEMO-2 state TODO -> IN_PROGRESS
5c692a6 kira: DEMO-3 edit due
f105046 kira: DEMO-3 edit priority
6f16580 kira: DEMO-3 create "Add search to the docs site"
9a26938 kira: DEMO-2 create "Fix broken anchor links"
08f8842 kira: DEMO-1 create "Write the getting-started guide"
13772a5 kira: install hooks
f209bfd kira: init
1625437 initial commit
```

And `kira log` reads that history back per item, interleaving field changes
with the commits that caused them:

```console
$ kira log DEMO-3
2026-07-27T19:22:02+05:30  state: TODO -> IN_PROGRESS
2026-07-27T19:22:02+05:30  commit b27843c kira: DEMO-3 state TODO -> IN_PROGRESS (Demo User)
2026-07-27T19:21:54+05:30  commit 2752bc8 kira: DEMO-1 link blocked-by DEMO-3 (Demo User)
2026-07-27T19:19:24+05:30  due:  -> 2026-08-15
2026-07-27T19:19:24+05:30  priority: P2 -> P1
2026-07-27T19:19:24+05:30  commit 5c692a6 kira: DEMO-3 edit due (Demo User)
2026-07-27T19:19:24+05:30  commit f105046 kira: DEMO-3 edit priority (Demo User)
2026-07-27T19:19:17+05:30  commit 6f16580 kira: DEMO-3 create "Add search to the docs site" (Demo User)
```

## 8. The TUI

```console
$ kira tui
```

Running `kira` with no command does the same thing. The UI has three screens —
a **tree** of epics and their tickets, a **kanban board** grouped by workflow
state, and a **stats** screen with completion and cycle-time metrics. It is
fully interactive: you can create, edit, move, and link items without dropping
back to the shell. Press `q` to quit. The full keymap and per-screen behaviour
are in the [TUI Guide](07-tui-guide.md).

## What to read next

| Document | When |
| --- | --- |
| [Concepts](03-concepts.md) | Before configuring anything. Identity, workflows, boards, sprints, relationships. |
| [CLI Reference](04-cli-reference/README.md) | Command-by-command flags and output, grouped by area. |
| [Configuration Reference](05-configuration-reference.md) | Every key in `.kira/config.yaml`. |
| [TUI Guide](07-tui-guide.md) | Keybindings and screens. |
| [Recipes and Examples](08-recipes-and-examples.md) | Worked day-to-day workflows. |
| [Limitations and Roadmap](09-limitations-and-roadmap.md) | What kira does not do yet. |
