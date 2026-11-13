# Overview

kira is a ticket tracker inside your git repository: no server, no login, no
database. A ticket is a Markdown file with YAML frontmatter under
`.kira/tickets/`, committed with the code it describes.

```
---
id: 01M7AN57WYK87HJKQZYZQSQT5D
number: DEMO-3
aliases: []
type: ticket
subtype: feature
title: "Add search to the docs site"
state: TODO
priority: P1
labels: []
parent: null
blocked_by: []
due: 2026-12-01
created: 2026-11-13T20:02:04+05:30
updated: 2026-11-13T20:02:04+05:30
---

## Description

## Acceptance criteria
```

## What git provides

| Need | Mechanism |
| --- | --- |
| Tracker on a new machine | `git clone` |
| Who changed what, when | `git log`, `kira log`, `kira blame` |
| Offline use | the local clone |
| Ticket changes reviewed with the code | same branch, same pull request |
| Backup | every clone |

With the default `commit.mode: auto` every kira mutation is its own commit;
the commit graph is the audit log.

## The model in one paragraph

Two item types, `ticket` and `epic`; an epic groups items through their
`parent` field. Each item belongs to one **board**, which gives it a display
number (`DEMO-3`); the immutable ULID `id` is the filename and what references
store. States are fixed per type (`TODO → IN_PROGRESS → REVIEW → DONE`, plus
`WONT_DO`, for tickets) and any state may move to any other; closing is refused
while blockers are open. Full detail: [Concepts](03-concepts.md).

## Front ends

- **CLI** — every command is scriptable; most accept `--json`.
- **TUI** — `kira` or `kira tui`: tree, board and stats screens
  ([TUI Guide](07-tui-guide.md)).
- **Web dashboard** — `kira server`, loopback only.

All three go through the same core operations.

## Concurrent edits

Plain `git pull`/`merge`/`push` move tickets between clones. kira handles the
results:

- `kira hooks install` registers a **merge driver**: ticket files merge per
  field, so edits to different fields never conflict.
- `kira sync` pulls with rebase, renumbers display-number collisions and
  refreshes the local cache.
- `kira resolve` field-merges ticket files a merge left conflicted.

See [Git integration](04-cli-reference/git-integration.md).

Next: [Getting Started](02-getting-started.md).
