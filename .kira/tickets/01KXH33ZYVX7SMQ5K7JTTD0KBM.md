---
id: 01KXH33ZYVX7SMQ5K7JTTD0KBM
number: CORE-6
aliases: []
type: ticket
subtype: bug
title: "Nested store root breaks git pathspec contract in sync/resolve/commit/hooks"
state: IN_PROGRESS
priority: P1
labels: []
epic: null
blocked_by: []
created: 2026-07-15T01:24:31+05:30
updated: 2026-07-15T02:13:52+05:30
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01KXH3404GZHVXJGTZTJABF107 author=Shivam-Shivanshu ts=2026-07-15T01:24:31+05:30 -->
When the kira store root is a subdirectory of the git toplevel, gitx porcelain output (toplevel-relative) is fed to git commands run from the subdir. Verified experimentally: git status --porcelain from a subdir prints 'sub/.kira/tickets/x.md' and 'git add -- sub/.kira/...' from that subdir fails 'pathspec did not match'.

Broken consumers: core/sync.go — DirtyPaths detects dirt fine, but finalize->Stage fails at sync.go:141 under DirtyCommit policy (under DirtyStash, prepareTree succeeds and the sync breaks later only on rebase conflict, where autoResolve resolves nothing and aborts); resolve.go:21-28/:81 (IsItemPath never matches — direct 'kira resolve' returns an empty ResolveResult with no error; in sync the rebase aborts loudly with 'could not auto-resolve'); kiracommit.go summarizeDirty; hookrun.go:83 ValidateStaged. index/staleness.go:97-110 does it right (ToplevelHead + filepath.Rel), proving nested layout is intended to work. No test anywhere sets up toplevel != store root.

Fix — pick one contract at the boundary: (a) rebase gitx porcelain/diff/staged output onto r.Dir via one ToplevelHead-based helper in StatusPorcelain/DirtyPaths/StagedPaths/UnmergedPaths/DiffNameStatus, or (b) hard-require store root == git toplevel in Discover and delete index's toplevel plumbing.

Verify: add the nested-root integration test that currently fails — init in subdir, mutate -> index -> kira commit -> dirty sync -> conflict + resolve.

Files: internal/core/sync.go, internal/core/resolve.go, internal/core/kiracommit.go, internal/core/hookrun.go, internal/gitx/staged.go, internal/index/staleness.go, tests/integration
<!-- /kira:comment -->
