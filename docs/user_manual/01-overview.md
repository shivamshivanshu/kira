# Overview

kira is a ticket tracker that lives inside your git repository. There is no
server, no login, and no database — a ticket is a Markdown file with YAML
frontmatter under `.kira/`, committed alongside the code it describes.

```
---
id: 01KYHXCJK0A3GJ2QWE1VRXV8F2
number: DEMO-3
type: ticket
title: "Add search to the docs site"
state: TODO
priority: P1
---

## Description

## Acceptance criteria
```

## Why files in git

Because tickets are ordinary tracked files, git already provides most of what a
tracker normally has to reinvent:

| You want | git gives you |
| --- | --- |
| The whole tracker on a new machine | `git clone` |
| History of who changed what, and when | `git log`, `git blame` |
| Offline access | the local clone |
| Ticket changes reviewed with the code | the same branch and pull request |
| Backups and replication | every clone is one |

Tickets travel on branches. A feature branch can carry both the code and the
state change that says the work is in review, and merging the branch merges
both.

## What you track

Two item types ship out of the box:

| Type | Purpose |
| --- | --- |
| `ticket` | A unit of work. |
| `epic` | A grouping of tickets. A ticket may optionally name one epic as its parent. |

Every item belongs to exactly one **board**, and boards give items their
display number. A repository can define several boards to separate concerns —
this repository uses `KIRA`, `CORE`, `TUI`, `CLI`, `DATA`, `INFRA`, and `DX`, so
its items are numbered `CORE-43`, `TUI-12`, and so on.

Item state moves through a fixed **workflow** built into kira. Tickets go:

```
TODO -> IN_PROGRESS -> REVIEW -> DONE
   \________ WONT_DO _________/
```

and epics go `PLANNED -> ACTIVE -> DONE`. Any state may move to any other; the
only transition guard is that entering a `done` state needs the item's blockers
closed. See
[Concepts](03-concepts.md#workflow-states-and-categories).

## Ticket history is git history

By default (`commit.mode: auto`) every kira mutation is its own git commit,
made immediately:

```console
$ git log --oneline
0628f5c DEMO-2 state REVIEW -> DONE
badf3af DEMO-2 state IN_PROGRESS -> REVIEW
e76a029 DEMO-2 state TODO -> IN_PROGRESS
5c692a6 DEMO-3 edit due
f105046 DEMO-3 edit priority
6f16580 DEMO-3 create "Add search to the docs site"
9a26938 DEMO-2 create "Fix broken anchor links"
08f8842 DEMO-1 create "Write the getting-started guide"
```

This is deliberate. There is no separate audit log to consult, because the
audit log is the commit graph. `kira log <id>` reads a ticket's field history
straight out of it.

## Two front ends

- **The CLI.** Every command is scriptable, and every command accepts `--json`
  for machine-readable output.
- **The TUI.** `kira tui` — or just `kira` with no arguments — opens an
  interactive terminal UI with tree, board, and stats screens.

Both drive exactly the same operations; neither can do something the other
cannot.

## Being git-native cuts both ways

Ordinary `git pull`, `git merge`, and `git push` move tickets between clones,
which means two people can edit the same ticket on different branches. kira
handles this rather than leaving you with conflict markers in a YAML file:

- `kira hooks install` registers a **merge driver** so ticket files merge
  field by field — two people changing different fields of the same ticket is
  not a conflict.
- `kira sync` pulls, repairs any display-number collisions the merge created,
  and refreshes the local cache.
- `kira resolve` auto-resolves conflicted ticket files left by a merge that ran
  without the driver.

## Next

- [Getting Started](02-getting-started.md) — install, initialize a repository,
  and drive a ticket from creation to done.
- [Concepts](03-concepts.md) — identity, states, vocabularies, sprints,
  relationships, and configuration layering in depth.
