---
id: 01M1BYZGZ62E78W1XNYNWT6S0X
number: KIRA-127
aliases: []
type: ticket
title: "kira sync stashes the whole tree though its dirty check is .kira-scoped"
state: DONE
resolution: done
priority: P2
labels: []
epic: null
blocked_by: []
created: 2026-08-31T15:08:46+02:00
updated: 2026-09-01T10:25:58+02:00
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01M1BYZH79T2DCSSNBT4ECV9PV author=Shivam-Shivanshu ts=2026-08-31T15:08:46+02:00 -->
The dirty gate is repo.DirtyPaths('.kira') (core/sync.go:122) but Stash() runs an unscoped 'git stash push --include-untracked' (gitx/sync.go:31-34).
Scenario: dirty .kira plus unrelated modified or untracked source files -> kira sync --stash sweeps the user's non-kira WIP into the stash. If the pop then conflicts on a non-kira path (popStash cannot auto-resolve those), sync errors out with the user's unrelated work stranded in a stash entry. A ticket tool must not touch files it was never pointed at.
Fix: scope the stash to the same pathspec as the dirty check - git stash push --include-untracked -- .kira.
<!-- /kira:comment -->
