---
id: 01KXH34JAZBEAV5N0FC09AXXGS
number: DATA-9
aliases: []
type: ticket
subtype: task
title: "Index perf and cleanup: targeted fillActivity, prepared statements, delete EnsureFresh"
state: REVIEW
priority: P2
labels: []
epic: null
blocked_by: []
created: 2026-07-15T01:24:50+05:30
updated: 2026-07-16T19:43:12+05:30
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01KXH34JGQ4DPR9GBH7JGFJGQV author=Shivam-Shivanshu ts=2026-07-15T01:24:50+05:30 -->
- staleness.go:64-68 + rebuild.go:159/:197/:188-192: one dirty edit at 1k items costs two full scans + 1000 UPDATEs. Fix: UPDATE only rows where a commit ts beats updated (optionally unioned with refreshed ULIDs — do NOT use 'restrict to refreshed ULIDs' alone: the trailer scan can add commit_links for items outside the refresh set whose activity also needs boosting).
- rebuild.go:74-109 tx.Exec in loops while upsertCommitLinks/fillActivity already tx.Prepare (modernc re-prepares per Exec) — prepare the four statements per transaction.
- EnsureFresh has zero production callers and lacks the discard-retry net — delete it; retarget the ~30 test call sites at Refresh (tests hold an open *Index; Refresh opens/closes its own — hence size M).
- decide() (staleness.go:124) takes nine positionals its caller restates as planResult — make it a planResult method; plain switch d.name in dispatch; hoist the duplicated reindex condition (:64/:74).
- events.go:46-50 discards the Scan error (only unchecked error in the package); :26-35 re-queries what it just wrote — return (string, error) from eventHead treating ErrNoRows as empty; use the derived slice on cache miss.
- trailers.go:113-137 latestCloses discards time.Parse errors (unparseable ts silently loses the race) — standardize on timex.CompareRFC3339 there and in core reopenedSince; add an in-package trailers unit test (latestCloses/bodyOutsideTrailers/lenientPattern/trailerRange are unreachable from the external test package).
- core/load.go readRaw drops the index.Load error text behind generic WarnIndexFallback while silently changing Activity semantics — include the swallowed error in the args and pin the fallback Activity semantics; drop redundant test env plumbing testutil already sets.

Verify: benchmark before/after on a 1k-item fixture with one dirty edit; suite green after EnsureFresh removal.

Files: internal/index/staleness.go, internal/index/rebuild.go, internal/index/events.go, internal/index/trailers.go, internal/core/load.go
Depends on: Index self-healing: dead-SHA recovery, deleted-DB detection, dup-ID skip, WAL-safe reads
<!-- /kira:comment -->
