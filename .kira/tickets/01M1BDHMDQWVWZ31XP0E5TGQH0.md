---
id: 01M1BDHMDQWVWZ31XP0E5TGQH0
number: KIRA-106
aliases: []
type: ticket
title: "TestSeedProducesItems flakes on TempDir cleanup: .git directory not empty"
state: DONE
priority: P3
labels: []
epic: null
blocked_by: []
created: 2026-08-31T10:04:05+02:00
updated: 2026-08-31T12:30:05+02:00
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01M1BDHYVZ0XFQAHR56RGJZR2G author=Shivam-Shivanshu ts=2026-08-31T10:04:15+02:00 -->
Pre-existing flake, not introduced by CORE-3 phase 3a (measured 2026-08-31). internal/seed/test TestSeedProducesItems intermittently fails in t.TempDir cleanup: 'RemoveAll cleanup: unlinkat <tmp>/001/.git: directory not empty'. Rate ~2/10 with 'go test -race -count=10 ./internal/seed/test/', and identical on main and on the phase-3a branch, so it is not caused by the codec/extras work. Not caused by core.fsmonitor either: reproduces with GIT_CONFIG_GLOBAL=/dev/null. The test body passes; only cleanup fails, which suggests something still writing into .git when the temp dir is removed (a git subprocess outliving the call, or a lingering lock/index file). Fix likely belongs in the seed test harness or internal/testutil - close the repo / wait for git subprocesses before the TempDir cleanup hook runs.
<!-- /kira:comment -->

<!-- kira:comment id=01M1BGS8A44Y839C33GWCDG4Y0 author=Shivam-Shivanshu ts=2026-08-31T11:00:40+02:00 -->
Second instance of the same family observed 2026-08-31: TestPopStashManualConflictSurfaces (internal/core) failed with 'TempDir RemoveAll cleanup: unlinkat <tmp>/001/.git/objects: directory not empty' during a full-suite -race run. Both it and TestSeedProducesItems pass 6/6 when run in isolation with -count=6, so the trigger is full-suite parallel load, not the individual test. Not toolchain-related: the local toolchain is go1.26.5 for every run, and the failure reproduces regardless of whether go.mod declares go 1.25.0 or go 1.26.0. Common shape: a hermetic git repo under t.TempDir where a git subprocess (or its lock/objects writes) is still live when the cleanup hook runs. Fix likely belongs in internal/testutil's repo helper - ensure git subprocesses are waited on and any long-lived handle is closed before the TempDir cleanup registered by the helper.
<!-- /kira:comment -->
