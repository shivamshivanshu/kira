---
id: 01M1BDHMDQWVWZ31XP0E5TGQH0
number: KIRA-106
aliases: []
type: ticket
title: "TestSeedProducesItems flakes on TempDir cleanup: .git directory not empty"
state: TODO
priority: P3
labels: []
epic: null
blocked_by: []
created: 2026-08-31T10:04:05+02:00
updated: 2026-08-31T10:04:05+02:00
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01M1BDHYVZ0XFQAHR56RGJZR2G author=Shivam-Shivanshu ts=2026-08-31T10:04:15+02:00 -->
Pre-existing flake, not introduced by CORE-3 phase 3a (measured 2026-08-31). internal/seed/test TestSeedProducesItems intermittently fails in t.TempDir cleanup: 'RemoveAll cleanup: unlinkat <tmp>/001/.git: directory not empty'. Rate ~2/10 with 'go test -race -count=10 ./internal/seed/test/', and identical on main and on the phase-3a branch, so it is not caused by the codec/extras work. Not caused by core.fsmonitor either: reproduces with GIT_CONFIG_GLOBAL=/dev/null. The test body passes; only cleanup fails, which suggests something still writing into .git when the temp dir is removed (a git subprocess outliving the call, or a lingering lock/index file). Fix likely belongs in the seed test harness or internal/testutil - close the repo / wait for git subprocesses before the TempDir cleanup hook runs.
<!-- /kira:comment -->
