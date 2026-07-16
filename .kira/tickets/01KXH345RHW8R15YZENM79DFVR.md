---
id: 01KXH345RHW8R15YZENM79DFVR
number: CORE-18
aliases: []
type: ticket
subtype: bug
title: "Resolve writes tickets unlocked and non-atomically"
state: REVIEW
priority: P1
labels: []
epic: null
blocked_by: []
created: 2026-07-15T01:24:37+05:30
updated: 2026-07-16T13:04:24+05:30
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01KXH345Y9VKSMBCB72W95TN64 author=Shivam-Shivanshu ts=2026-07-15T01:24:37+05:30 -->
writeResolvedFile (internal/core/resolve.go:88-93) does os.WriteFile(join(s.root, path), codec.Serialize(it), itemFileMode) and Resolve never calls s.fs().Lock(). Contrast mutate.go lockAndResolve+WriteItem, comment.go, reconcile.go, kiracommit.go — all locked and atomic. A concurrent 'kira edit' during 'kira resolve' can interleave; a crash mid-WriteFile leaves a truncated ticket that LoadAll then skips.

Fix: take s.fs().Lock() in Resolve (not held elsewhere on this path: Sync locks only inside Reconcile, after autoResolve) and route the write through s.fs().WriteItemRaw(it.ID, codec.Serialize(it)). Caveat: WriteItemRaw writes to s.ItemPath(it.ID), which equals the git-reported conflicted path only under the filename==ID invariant — stage the rel path returned by WriteItemRaw (or assert it equals the unmerged path) instead of the original path, so a filename/ID mismatch can't stage a file that was never rewritten. MergeFile (merge-driver temp %A) stays exempt.

Verify: test that Resolve under an already-held lock returns errx.Conflict, and the resolved write is atomic (tmp+rename path exercised).

Files: internal/core/resolve.go, internal/storage/write.go
<!-- /kira:comment -->
