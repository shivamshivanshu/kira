---
id: 01KXH348NX897HJBJHZ0SPCD3V
number: DX-5
aliases: []
type: ticket
subtype: task
title: "Merge matrix: add delete/modify conflict scenarios"
state: WONT_DO
resolution: dropped
priority: P2
labels: []
epic: null
blocked_by: []
created: 2026-07-15T01:24:40+05:30
updated: 2026-07-16T18:55:59+05:30
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01KXH348VKJZA7D7SAR525DV8G author=Shivam-Shivanshu ts=2026-07-15T01:24:40+05:30 -->
matrixScenarios() (tests/integration/matrix_test.go:108-270) covers field conflicts, comments, body prose, ties, number collisions — every scenario only w.write()s; none deletes a ticket on one side, though diff_test.go:39 shows deletion is a supported user action. git's merge machinery skips merge.kira.driver for delete/modify, leaving an unmerged index entry with a missing stage — a different code path from everything exercised; Resolve/Reconcile behavior there is unverified.

Expected outcome differs from existing scenarios: delete/modify is NOT auto-resolvable by design — Resolve skips paths with a missing ours/theirs stage (resolve.go:50-53), so under MergeAuto the sync path must surface a conflict and abort the rebase (sync.go:174-178), and the driver path leaves an unmerged index entry. New scenarios cannot reuse the conflicts:true + auto => clean-merge assertion flow (assertNoUnmerged/assertIdempotent are unconditional); they need their own expected-outcome shape asserting surfaced-conflict behavior on both policies, plus a manual-resolution completion path for the driver case.

Fix: extend the scenario struct so ours/theirs can delete (w.delete(ulid) via git rm), add delete-vs-modify and modify-vs-delete scenarios, assert the expected policy outcome plus assertNoUnmerged/assertIdempotent on both paths and both policies.

Files: tests/integration/matrix_test.go
<!-- /kira:comment -->
