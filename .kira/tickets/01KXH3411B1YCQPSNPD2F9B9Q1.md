---
id: 01KXH3411B1YCQPSNPD2F9B9Q1
number: DATA-4
aliases: []
type: ticket
subtype: bug
title: "StatusPorcelain misses untracked ticket files — index never sees new tickets in manual-commit mode"
state: IN_PROGRESS
priority: P1
labels: []
epic: null
blocked_by: []
created: 2026-07-15T01:24:32+05:30
updated: 2026-07-15T01:26:33+05:30
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01KXH3417119QBCZTNQB1ZE6W3 author=Shivam-Shivanshu ts=2026-07-15T01:24:33+05:30 -->
internal/gitx/index.go:47 runs status --porcelain with default -unormal while staged.go:12 DirtyPaths passes --untracked-files=all into the same parser. Caller index/staleness.go:106 uses StatusPorcelain; ticketAbsPaths (:175-184) drops the collapsed directory entry (ULIDFromPath('') fails). Under manual commit mode the first tickets leave .kira/tickets/ wholly untracked, so dirtyHash stays constant and decide() returns actionFresh — new tickets (and edits to existing untracked ticket files, which never get content-hashed by dirtyState) never enter the index until a commit. Masked on machines with status.showUntrackedFiles=all in global gitconfig (this dev box sets it), explaining inconsistent local repro — must pass -uall explicitly.

Fix: delete StatusPorcelain and have staleness.go call root.DirtyPaths(pathspec) (already variadic, already -uall, same parser); or minimally add --untracked-files=all.

Verify: test with hermetic git config (no showUntrackedFiles): create ticket in manual mode without committing, assert it appears in kira list via the index (not FS fallback).

Files: internal/gitx/index.go, internal/index/staleness.go, internal/gitx/staged.go
<!-- /kira:comment -->
