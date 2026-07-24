---
id: 01KY9S1P4DJT649ZF3X10DQZEV
number: CORE-43
aliases: []
type: ticket
title: "Enforce blocked_by target-type on write (align with doctor)"
state: DONE
labels: []
epic: null
blocked_by: []
created: 2026-07-24T15:29:31+05:30
updated: 2026-07-24T19:09:32+05:30
---

## Description

The write path validates `blocked_by` ref existence (`normalizeAndCheckRefs`)
but not its target type: a ticket can be `blocked_by` an epic and the write
succeeds. `kira doctor` (`RefTypeMismatches`, phase 2) already flags this;
`entityschema.Validate` now computes it too but `core.validateItem` suppresses
the `CodeRef` for `blocked_by` to preserve current behavior.

Decide the intended policy and enforce it on write (stop suppressing, or add a
`blocked_by` type check to `validateGraph`) — a deliberate tightening, so it
needs its own review + test/golden updates.

Surfaced during CORE-3 phase 3b-i review.

## Acceptance criteria

## Comments
