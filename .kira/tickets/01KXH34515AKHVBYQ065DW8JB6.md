---
id: 01KXH34515AKHVBYQ065DW8JB6
number: CORE-16
aliases: []
type: ticket
subtype: bug
title: "Force GIT_EDITOR=true via env in rebase — editor failures masked or sync hangs"
state: DONE
resolution: done
priority: P1
labels: []
epic: null
blocked_by: []
created: 2026-07-15T01:24:36+05:30
updated: 2026-07-15T02:47:11+05:30
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01KXH3456YCHP9CHFWX25NJT56 author=Shivam-Shivanshu ts=2026-07-15T01:24:37+05:30 -->
internal/gitx/sync.go:21-24 uses '-c core.editor=true' but git precedence is GIT_EDITOR env > core.editor config > VISUAL/EDITOR — a user GIT_EDITOR wins. Failure is partially masked at core/sync.go:179: 'if err := repo.RebaseContinue(); err != nil && !repo.RebaseInProgress()' — a failed editor leaves the rebase in progress so the error is swallowed; autoResolve finds nothing unmerged, RebaseContinue fails identically each pass, and after maxRebaseIterations the loop aborts with the misleading 'rebase did not converge' (core/sync.go:189-191). For editors that block without a tty (GUI --wait editors), sync hangs indefinitely (OutputRaw has no timeout, stdin /dev/null).

Fix: run with cmd.Env = append(os.Environ(), "GIT_EDITOR=true") — extend gitx with an env-aware command variant (the gitx command() helper cleanup also wants this; see 'gitx API cleanup' ticket).

Verify: test that a rebase-continue with GIT_EDITOR set in the parent env still uses true (env override wins over user env).

Files: internal/gitx/sync.go, internal/gitx/gitx.go
<!-- /kira:comment -->
