---
id: 01KXH340NTD9Y0DRWWKSV5B4YW
number: DATA-3
aliases: []
type: ticket
subtype: bug
title: "Index self-healing: dead-SHA recovery, deleted-DB detection, dup-ID skip, WAL-safe reads"
state: TODO
priority: P1
labels: []
epic: null
blocked_by: []
created: 2026-07-15T01:24:32+05:30
updated: 2026-07-15T01:24:32+05:30
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01KXH340VHGM35K9TN93XKB4YN author=Shivam-Shivanshu ts=2026-07-15T01:24:32+05:30 -->
Five storage-spine defects, one healing mechanism:
1) decide() (internal/index/staleness.go:135) calls IsAncestor; gitx maps exit 128 ('not a valid object', e.g. after history rewrite) to *CmdError and load.go:23-25 returns early WITHOUT discard(), so meta keeps the dead sha and every command falls back with WarnIndexFallback forever. Same hole in trailerRange (trailers.go:162). Fix: map only unknown-object IsAncestor failures to rewrite=true (full rebuild); other git CmdErrors still propagate.
2) open() (index.go:35-47) recreates an empty index.db while loadMetaAt reads the sidecar independently — 'rm index.db' yields actionFresh over zero items. Fix: open() notes DB-file creation and forces full when meta claimed fresh.
3) read.go:15 opens 'mode=ro&immutable=1' against a WAL DB opened rw at index.go:36: -wal content invisible, no read locks. Impact limited (sole caller is shell completion via core/index.go:29 -> cli/complete.go:137, errors swallowed; single-process Close() checkpoints), but concurrent writers / crash-leftover -wal give stale or failed completions. Fix: drop immutable=1 (mode=ro + busy_timeout).
4) full() (rebuild.go:33-37) hits UNIQUE-constraint on duplicate it.ID; loadRetry discards and retries full, failing identically every run — index never builds, every command pays discard + full rescan (trailer watermarks wiped each attempt) then FS fallback. Fix: skip duplicate ids in full()/refresh() with a SkipNote naming both files.
5) Warnings only surface from dispatch(); meta has no skipped field, so a malformed ticket is silently absent from listings on every fresh run. Fix: persist skip-warnings in meta, replay until the file reindexes cleanly.

Verify: tests for each — history rewrite triggers rebuild not permanent fallback; rm index.db rebuilds; two files with one ULID indexes n-1 items with a warning; warning replay across runs.

Files: internal/index/staleness.go, internal/index/load.go, internal/index/index.go, internal/index/read.go, internal/index/rebuild.go, internal/index/trailers.go, internal/index/meta.go
<!-- /kira:comment -->
