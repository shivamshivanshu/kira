---
id: 01KY9ND7GJKVY1BGEX56M6EEGE
number: CORE-41
aliases: []
type: ticket
title: "Round-trip unknown link types losslessly"
state: DONE
resolution: done
labels: []
epic: null
blocked_by: []
created: 2026-07-24T14:25:55+05:30
updated: 2026-07-28T14:28:03+05:30
---

## Description

`codec.optLinks` still drops target IDs for unrecognized link types, so
`guardWritable` still hard-blocks writes on items carrying them. Extend phase-3a's
lossless round-trip (`ExtraScalars`/`ExtraLists`) to unknown `links:` types so
they survive edits too.

Follow-up from CORE-3 phase 3a.

## Acceptance criteria

## Comments
