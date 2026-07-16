---
id: 01KXH34BMPKR22DR3WMCYV2B6G
number: CORE-24
aliases: []
type: ticket
subtype: task
title: "gitx API cleanup: shared command() helper, single-spawn RebaseInProgress, naming/theming"
state: WONT_DO
resolution: dropped
priority: P3
labels: []
epic: null
blocked_by: []
created: 2026-07-15T01:24:43+05:30
updated: 2026-07-16T18:56:00+05:30
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01KXH34BTFE22A804MZR9TB9SN author=Shivam-Shivanshu ts=2026-07-15T01:24:43+05:30 -->
Command setup duplicated at gitx.go:28, :260-268, index.go:33, show.go:6, treeish.go:55. RebaseInProgress (:217-227) does one GitPath rev-parse per candidate dir (up to two spawns per poll; short-circuits after one when rebase-merge exists), called per iteration at core/sync.go:155/179/184 — rev-parse accepts repeated --git-path in one spawn. gitx.go:64 repo.Commit(subject, trailerKey, trailerVal) collides with type Commit; seed passes ('','') sentinels. sync.go:4 Pull is silently pull --rebase. index.go holds ToplevelHead/IsAncestor/StatusPorcelain/DiffNameStatus, none touching the index (staged.go has the real index ops). store.go:108 gitConfig duplicates ConfigValue (third copy automation.go:107); raw rev-parse HEAD at git.go:53/kiracommit.go:56; index/trailers.go:86 near-copies ResolveTreeish (differs only by ^{commit} suffix and tolerated error — keep the plain-ref form in the new helper; LandedRef may rely on peeling behavior). RemoveLineIfPresent scans twice; FileCommitMeta hardcodes %x00 where nulFmt exists.

Fix: private (r Repo) command(args ...) *exec.Cmd plus env-aware runner (GIT_EDITOR ticket wants it); rebuild the five sites on it; single-spawn RebaseInProgress via repeated --git-path; delete the Commit method (callers use CommitParts); rename Pull -> PullRebase; rename index.go to status.go and move ToplevelHead/IsAncestor by theme; add gitx.Head()/ResolveRef() and switch core/index raw calls to ConfigValue and the new helpers; single-pass RemoveLineIfPresent; nulFmt in FileCommitMeta.

Verify: existing gitx + core suites green; behavior audit that the refactor is intent-preserving.

Files: internal/gitx/gitx.go, internal/gitx/index.go, internal/gitx/sync.go, internal/gitx/show.go, internal/gitx/treeish.go, internal/core/store.go, internal/core/git.go, internal/core/kiracommit.go, internal/core/automation.go, internal/index/trailers.go
Depends on: Force GIT_EDITOR=true via env in rebase — editor failures masked or sync hangs; StatusPorcelain misses untracked ticket files — index never sees new tickets in manual-commit mode; gitx parsing robustness: NumstatNoIndex errors, CatFileBatch bounds, trailer sentinel, -z parsing
<!-- /kira:comment -->
