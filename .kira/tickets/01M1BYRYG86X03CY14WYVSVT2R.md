---
id: 01M1BYRYG86X03CY14WYVSVT2R
number: KIRA-111
aliases: []
type: ticket
title: "Three-way merge reverts a remote board move: Number is ours-wins but renumberable"
state: DONE
priority: P1
labels: []
epic: null
blocked_by: []
created: 2026-08-31T15:05:10+02:00
updated: 2026-08-31T15:53:57+02:00
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01M1BYRYR5FX5207QZ3PAFNTY3 author=Shivam-Shivanshu ts=2026-08-31T15:05:10+02:00 -->
From the DX-1 data contextual review.

internal/merge/merge.go:23 sets Number: ours.Number unconditionally, a rule enshrined by merge/test/merge_test.go:228-236 (TestImmutableFieldsFromOurs). But Number is NOT immutable - boardmove.go and reconcile.go renumber it by design.

Scenario: remote runs kira board move DX-7 CORE (item becomes CORE-3, alias DX-7); local concurrently edits the same item's title. Merge driver fires: merged item keeps Number DX-7 (ours, unchanged from base) while aliasUnion imports theirs' DX-7 alias, so the item ends up with live number equal to its own alias and CORE-3 is silently lost - already allocated and referenced by the remote's commit trailers, now dangling. Mirror-image loss in the other direction.

Fix: merge Number with threeWayScalar(base, ours, theirs, winner) - one-sided change wins, both-changed falls to the Updated winner like every other scalar - and drop the merged number from the alias union.
<!-- /kira:comment -->
