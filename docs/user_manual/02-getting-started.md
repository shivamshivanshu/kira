# Getting Started

From an empty git repository to a closed ticket. Each step links to the
reference that owns its details.

## Install

Release archives (linux/darwin × amd64/arm64) are on the GitHub Releases page;
put `kira` on your `PATH`. With Go:

```sh
go install github.com/shivamshivanshu/kira/cmd/kira@latest
```

```console
$ kira version
dev
```

## 1. Initialize

Run inside an existing git repository. Pass `--key` (the display-number prefix)
rather than relying on the one derived from the directory name.

```console
$ kira init --key DEMO
Initialized kira in .kira (project key DEMO)
$ git ls-files
.gitattributes
.kira/.gitignore
.kira/config.yaml
```

`init` commits these files. See
[`kira init`](04-cli-reference/setup-and-schema-commands.md#kira-init).

## 2. Install the git hooks

Per clone, not committed. The merge driver makes ticket files merge field by
field.

```console
$ kira hooks install
hook post-merge: installed
hook prepare-commit-msg: installed
merge driver: registered
```

See [`kira hooks`](04-cli-reference/git-integration.md#kira-hooks).

## 3. Create

`--no-edit` builds the item from flags; without it kira opens `$EDITOR` on a
template.

```console
$ kira create ticket "Write the getting-started guide" --no-edit
Created DEMO-1 Write the getting-started guide
$ cat .kira/tickets/*.md
---
id: 01M7AN57TZNZBYSHA2AZ1EGEGR
number: DEMO-1
aliases: []
type: ticket
title: "Write the getting-started guide"
state: TODO
labels: []
parent: null
blocked_by: []
created: 2026-11-13T20:02:04+05:30
updated: 2026-11-13T20:02:04+05:30
---

## Description

## Acceptance criteria
$ kira create ticket "Fix broken anchor links" --no-edit --priority P0 --subtype bug
Created DEMO-2 Fix broken anchor links
$ kira create ticket "Add search to the docs site" --no-edit --priority P2 --subtype story
kira: warning: field "subtype": value "story" is not in the "subtype" vocabulary
Created DEMO-3 Add search to the docs site
```

The warning is the default advisory vocabulary; see
[Concepts](03-concepts.md#vocabularies).

## 4. List, edit, show

```console
$ kira list
NUMBER  STATE  TYPE    PRIORITY  TITLE
DEMO-2  TODO   ticket  P0        Fix broken anchor links
DEMO-3  TODO   ticket  P2        Add search to the docs site
DEMO-1  TODO   ticket  -         Write the getting-started guide
$ kira edit DEMO-3 --priority P1 --subtype feature
Edited DEMO-3: subtype, priority
$ kira edit DEMO-3 --field due=2026-12-01
Edited DEMO-3: due
$ kira show DEMO-3
DEMO-3  Add search to the docs site  [TODO]
id:       01M7AN57WYK87HJKQZYZQSQT5D
type:     ticket
subtype:  feature
category: todo
priority: P1
created:  2026-11-13T20:02:04+05:30
updated:  2026-11-13T20:02:04+05:30


## Description

## Acceptance criteria
```

## 5. Move

```console
$ kira move DEMO-2 IN_PROGRESS
Moved DEMO-2: TODO -> IN_PROGRESS
$ kira move DEMO-2 DONE
Moved DEMO-2: IN_PROGRESS -> DONE
$ kira link DEMO-3 --blocked-by DEMO-1
Linked DEMO-3
$ kira move DEMO-3 DONE
kira: invalid item: TODO -> DONE is blocked by open items: DEMO-1
  hint: close the blockers first, or use `--force` to override
```

Any state may move to any other; the rules are in
[Concepts](03-concepts.md#workflow-states-and-categories).

## 6. History

Every mutation above was a commit (`commit.mode: auto`):

```console
$ git log --oneline
130c6fe DEMO-3 link blocked-by DEMO-1
d47cbe5 DEMO-2 state IN_PROGRESS -> DONE
70200a0 DEMO-2 state TODO -> IN_PROGRESS
a374260 DEMO-3 edit due
7b553fa DEMO-3 edit subtype,priority
200538e DEMO-3 create "Add search to the docs site"
8705dcc DEMO-2 create "Fix broken anchor links"
9fa4e18 DEMO-1 create "Write the getting-started guide"
6455128 kira: init
fbfbef2 initial commit
$ kira log DEMO-3
2026-11-13T09:16:00+05:30  commit 130c6fe DEMO-3 link blocked-by DEMO-1 (Demo User)
2026-11-13T09:12:00+05:30  due:  -> 2026-12-01
2026-11-13T09:12:00+05:30  commit a374260 DEMO-3 edit due (Demo User)
2026-11-13T09:11:00+05:30  subtype: story -> feature
2026-11-13T09:11:00+05:30  priority: P2 -> P1
2026-11-13T09:11:00+05:30  commit 7b553fa DEMO-3 edit subtype,priority (Demo User)
2026-11-13T09:09:00+05:30  commit 200538e DEMO-3 create "Add search to the docs site" (Demo User)
```

## 7. TUI and web dashboard

`kira` (or `kira tui`) opens the terminal UI — [TUI Guide](07-tui-guide.md).
`kira server` serves a local web dashboard at `http://127.0.0.1:7432`.

Next: [Concepts](03-concepts.md), then the
[CLI Reference](04-cli-reference/README.md).
