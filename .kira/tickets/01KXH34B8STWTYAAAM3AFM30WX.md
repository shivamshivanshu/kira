---
id: 01KXH34B8STWTYAAAM3AFM30WX
number: CORE-23
aliases: []
type: ticket
subtype: bug
title: "gitx parsing robustness: NumstatNoIndex errors, CatFileBatch bounds, trailer sentinel, -z parsing"
state: REVIEW
priority: P2
labels: []
epic: null
blocked_by: []
created: 2026-07-15T01:24:43+05:30
updated: 2026-07-16T19:43:12+05:30
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01KXH34BEPJ8MGTZP6PHXH7EAJ author=Shivam-Shivanshu ts=2026-07-15T01:24:43+05:30 -->
Batch of gitx parsing/error defects:
- treeish.go:59 '_ = cmd.Run()' swallows exit 128/missing git in NumstatNoIndex and Atoi ignores '-' binary markers. Partly deliberate (diff --no-index exits 1 when files differ) — mirror MergeText's exit-code discrimination (1 = differ, >=128/exec = error). Both call sites (core/changes.go:67, core/diff.go:103) currently discard the returned error — update them too or the fix has no effect.
- index.go:23 ToplevelHead returns plain fmt.Errorf -> load.go:23-27 discards the SQLite cache then fails identically; gitx.go:267 vs OutputRaw's empty-stderr fallback — return &CmdError from ToplevelHead and CatFileBatch (with err.Error() fallback).
- gitx.go:298 parseCatFileBatch: pos += size+1 can pass len(buf) and the NEXT iteration's IndexByte slice panics (only when the malformed record isn't the last; well-formed git output never triggers) — bounds-check pos after content.
- trailers.go:25-37: no trailing sentinel so f[7] = body+'\n' for all but the last record — terminate the trailer format with a trailing nulFmt.
- index.go:61 ' -> ' substring truncates unquoted paths containing it; LastCommits/LsTreeNames (gitx.go:107-124, treeish.go:38) never unquote C-quoted paths (near-theoretical for ASCII ULID filenames; robustness) — switch status/diff to -z parsing (removes the arrow heuristic and unquotePath) and apply -z/quotepath=off to LastCommits/LsTreeNames.

Verify: table tests for parseCatFileBatch (found/missing/truncated), LastCommits/RevListSince fixtures, splitValues/unquotePath — all currently untested (only trailers_test/porcelain_test exist).

Files: internal/gitx/treeish.go, internal/gitx/gitx.go, internal/gitx/index.go, internal/gitx/trailers.go, internal/core/changes.go, internal/core/diff.go, internal/index/load.go
<!-- /kira:comment -->
