---
id: 01M1GHBTQY8WNMJMK1W1WK4P8Z
number: KIRA-163
aliases: []
type: ticket
title: "An unparseable ticket is invisible to the number allocator, so create issues a duplicate"
state: DONE
resolution: done
priority: P0
labels: []
epic: null
blocked_by: []
created: 2026-09-02T09:47:01+02:00
updated: 2026-09-02T11:09:24+02:00
---

## Description

## Acceptance criteria

<!-- kira:comment id=01M1GHCQFPAKDJ52WP0JF3FJFC author=Shivam-Shivanshu ts=2026-09-02T09:47:30+02:00 -->
Reproduced by the durability review agent:

  $ ls .kira/tickets    # XX-1, XX-2 where XX-2 has a nested-map frontmatter field
  $ kira create ticket three --no-edit
  Created XX-2 three                 # exit 0, no warning
  $ grep -h '^number:' .kira/tickets/*.md
  number: XX-1
  number: XX-2
  number: XX-2

kira list then prints two rows both labelled XX-2. Only kira doctor notices (error id-collision).

Mechanism, traced: storage.LoadAll (read.go:85-101) drops a file that fails to parse and records a warning. id.HighestN (number.go:46-61) takes max(N)+1 over surviving items only. core/create.go destructures ld.items/ld.snap/ld.resolver and never reads ld.warnings, so create is the one command that does not even mention the skipped file — core/load.go:85 is where list surfaces them. Same blind spot for id.style: hash via takenNumbers (number.go:68-77).

P0 because 'unparseable' is not an exotic state: it is what a git conflict marker produces, and what a ticket written by a newer kira produces (see KIRA-164). Duplicate numbers are the one thing kira's two-tier identity exists to prevent.

Fix direction: have LoadAll grep ^number: out of files it skips and hand the allocator a reserved-number set, and surface ld.warnings on the create and mutate paths.
<!-- /kira:comment -->
