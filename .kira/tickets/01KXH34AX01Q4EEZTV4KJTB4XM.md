---
id: 01KXH34AX01Q4EEZTV4KJTB4XM
number: CORE-22
aliases: []
type: ticket
subtype: task
title: "storage: directory sync on rename, shared WriteFileAtomic, portable flock, lock tests"
state: REVIEW
priority: P2
labels: []
epic: null
blocked_by: []
created: 2026-07-15T01:24:42+05:30
updated: 2026-07-16T19:43:12+05:30
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01KXH34B2WTV63JG0K9XJ9ZAAB author=Shivam-Shivanshu ts=2026-07-15T01:24:43+05:30 -->
internal/storage/write.go:30 syncs the file but commit() (:73-79) renames with no directory sync — the one missing step of the standard crash-safe recipe. internal/index/meta.go:47-54 saveMetaAt re-implements tmp+rename, leaking a partial .tmp on WriteFile failure with no Sync. lock.go:6/:28 calls syscall.Flock with no build tag while golang.org/x/sys is already an (indirect, go.mod:46 — promotes to direct) dep and the rest of the stack is Windows-viable. lock.go:17-40 is the only untested file in the package; seven core mutation paths depend on it; flock contends per open-file-description so it is testable in-process, but the hardcoded 2s lockTimeout makes the timeout path expensive.

Fix: parent-dir Sync after rename (soft-fail on darwin EINVAL); collapse atomicFile into an exported storage.WriteFileAtomic and switch saveMetaAt to it; move flock behind x/sys/unix in lock_unix.go with //go:build unix (plus a Windows impl or explicit stub); make lockTimeout/poll injectable.

Verify: add lock_test.go — second Lock -> errx.Conflict; release -> acquire.

Files: internal/storage/write.go, internal/storage/lock.go, internal/index/meta.go, go.mod
<!-- /kira:comment -->
