---
id: 01M1BYZAVSHV259E57AQAMVAHV
number: KIRA-115
aliases: []
type: ticket
title: "Three-way merge deletes an untouched empty-list extra"
state: DONE
priority: P2
labels: []
epic: null
blocked_by: []
created: 2026-08-31T15:08:39+02:00
updated: 2026-08-31T15:53:59+02:00
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01M1BYZB3SZVPY1TATNF5KV0E8 author=Shivam-Shivanshu ts=2026-08-31T15:08:40+02:00 -->
Convergent: found by both the data blind and data contextual DX-1 reviewers. Regression from CORE-3 phase 3a.
merge/extras.go:8-11,18-36 routes every extra list through setMerge; setMerge (merge/set.go:5) returns nil for three empty inputs, so ExtraValue{List: []string{}} collapses to the zero value and extrasMerge drops the key - even when NO side changed it. The rest of the system deliberately keeps empty-list distinct from absent: codec/serialize.go:79-81 writes 'key: []' and index/rebuild.go:284-297 stores a dedicated NULL row for exactly that reason. ChangedFields does not flag it either (slices.Equal(nil, []) is true), so the deletion is invisible to arbitration reporting.
Scenario: item carries 'tags: []' identically on all three sides; any merge-driver run on that file (conflict on an unrelated field) emits a file with no 'tags' line. Also '[a, a]' unchanged everywhere becomes '[a]', and a pure reorder reverts to base order.
Fix: when neither side changed the list relative to base, return it verbatim; only apply set-union on genuine divergence, and preserve a non-nil empty list.
<!-- /kira:comment -->
