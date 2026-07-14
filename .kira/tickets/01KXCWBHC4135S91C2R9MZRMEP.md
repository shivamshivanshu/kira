---
id: 01KXCWBHC4135S91C2R9MZRMEP
number: KIRA-39
aliases: []
type: ticket
title: "Consolidate item ref-match predicate into one datamodel helper (4 copies)"
state: DONE
resolution: done
priority: P3
labels: []
epic: 01KXCWAN7PK346KK28DTB1BCRQ
blocked_by: []
created: 2026-07-13T10:09:21+05:30
updated: 2026-07-13T22:15:17+05:30
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01KXE5T2KNHRYWTTTMMDAZBJNA author=Shivam-Shivanshu ts=2026-07-13T22:13:49+05:30 -->
Verified in professionalize worktree: ref matching is consolidated into id.Resolver.Resolve (internal/id/resolve.go) with a single core caller Store.resolveRef (internal/core/store.go:58). The 4-copy premise no longer holds after the W1 dedup wave.
<!-- /kira:comment -->
