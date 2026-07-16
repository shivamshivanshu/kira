---
id: 01KXH342G01TW2T7TKBVEJC09X
number: CORE-11
aliases: []
type: ticket
subtype: bug
title: "--at: one malformed historical ticket permanently breaks every spanning query"
state: IN_PROGRESS
priority: P1
labels: []
epic: null
blocked_by: []
created: 2026-07-15T01:24:34+05:30
updated: 2026-07-16T12:32:05+05:30
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01KXH342NQZ1JFMW9N5ZMHF1F2 author=Shivam-Shivanshu ts=2026-07-15T01:24:34+05:30 -->
internal/treeish/loader.go:64-70 returns an error for the whole load on codec.Parse failure while live LoadAll (storage/read.go:58-63) skips with SkipNote — history is immutable, so a hand-committed malformed ticket breaks every --at/diff/changes query forever; all three callers wrap as fatal errx.User.

Fix: skip unparseable/missing blobs with collected warnings — add Warnings to Loaded; list/show/board already plumb loaded.notes -> StderrNotes, but DiffResult/ChangesResult (datamodel/results.go:162,198) have no warnings field and need new plumbing. Reword the missing-config-blob message (loader.go:53-55): ls-tree already proved the path exists, so it's a corrupt tree, not 'before kira init'.

Fold in (same file): return errx with hints from treeish and delete the five '%v' re-wrappers; resolve the treeish to a SHA once at entry; delete write-only Loaded.Treeish/Snapshot; hoist the shared items->snapshot/resolver tail into storage; drop the core import from tests. Latent (note, don't fix behavior): loader.go:56 uses config.Parse (defaults+repo only) and core/load.go:57 returns tl.Config wholesale, dropping the user tier — currently unobservable (--at paths read cfg only for workflows/categories/priorities; editor/icons/theme come from the live cfg at the CLI/TUI layer). Optionally overlay user-tier fields via config.OverlayUser to close the latent hazard.

Verify: add TestLoadBeforeInit and TestLoadCorruptItem (currently zero error-path tests).

Files: internal/treeish/loader.go, internal/core/load.go, internal/datamodel/results.go, internal/storage/read.go
<!-- /kira:comment -->
