---
id: 01KXH34CR69Y0CDQ9N1MFCF9NA
number: CORE-27
aliases: []
type: ticket
subtype: task
title: "Batch mutate entry point: one lock/load for closes, sprint --move-to, and bulk"
state: TODO
priority: P2
labels: []
epic: null
blocked_by: []
created: 2026-07-15T01:24:44+05:30
updated: 2026-07-15T01:24:44+05:30
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01KXH34CY2JJ6ZG1G738ME8KC3 author=Shivam-Shivanshu ts=2026-07-15T01:24:45+05:30 -->
internal/core/closes.go:51 loops s.Move per candidate; each Move -> mutate -> lockAndResolve -> LoadAll re-reads every file, right after applyCloses already holds a full ld (whose initial load is unlocked — only each Move locks; batching strengthens atomicity as a side effect). sprint.go:133-136 identical for --move-to. cli/bulk.go:11-38 invokes apply per id, so 'kira move A B C DONE' does 3 full reads + 3 commits. findByULID is a linear scan inside these loops; RefExists (core/store.go:116) loads everything per-arg from cli/assign.go:36 and cli/label.go:107.

Fix: add a batch-aware mutate entry point in core — one lock, one load, apply a list of (ref, applyFn) against the shared snapshot, committing per item (keeping per-item subjects). Critical: fold each committed mutation back into the shared items snapshot before applying the next item — validateMutation reads the shared slice (e.g. WIP-limit counting), so a stale snapshot would validate later items against pre-batch state. Use from applyCloses, SprintClose, and runBulk (CLI outcome shape unchanged). Have RefExists reuse an already-loaded resolver where the CLI holds one.

Verify: existing closes/sprint/bulk tests green; a WIP-limit batch test where item 2's validity depends on item 1's move.

Files: internal/core/closes.go, internal/core/sprint.go, internal/core/mutate.go, internal/core/store.go, internal/cli/bulk.go, internal/cli/assign.go, internal/cli/label.go
<!-- /kira:comment -->
