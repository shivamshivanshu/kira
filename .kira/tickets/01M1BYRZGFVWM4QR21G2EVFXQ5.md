---
id: 01M1BYRZGFVWM4QR21G2EVFXQ5
number: KIRA-113
aliases: []
type: ticket
title: "Sync rebase loop cannot skip an empty commit and aborts with 'did not converge'"
state: DONE
resolution: done
priority: P1
labels: []
epic: null
blocked_by: []
created: 2026-08-31T15:05:11+02:00
updated: 2026-09-01T10:25:57+02:00
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01M1BYRZRQ1GEYD845CZYBQT7W author=Shivam-Shivanshu ts=2026-08-31T15:05:11+02:00 -->
Found independently by BOTH the core blind and platform contextual DX-1 reviewers - convergent finding.

internal/core/sync.go:167-194 (pullRebase) loops autoResolve then RebaseContinue; internal/gitx/sync.go:21-24 is a bare git rebase --continue with no --skip path. merge.Merge picks the later-updated side per field (merge.go:19,43).

Scenario: local and remote both edit the same field and the remote edit is newer. During kira sync's rebase the driver arbitrates every conflicting field to the remote side, so the merged file is byte-identical to the rebase parent. The staged result is a no-op; git rebase --continue fails with 'No changes ... use git rebase --skip'; the rebase is still in progress and nothing is unmerged, so the loop repeats identically 100 times and aborts the whole sync.

Fix: when RebaseContinue fails while a rebase is still in progress and there are no unmerged paths, run git rebase --skip instead of looping.
<!-- /kira:comment -->
