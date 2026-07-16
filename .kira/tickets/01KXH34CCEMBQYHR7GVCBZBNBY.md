---
id: 01KXH34CCEMBQYHR7GVCBZBNBY
number: CORE-26
aliases: []
type: ticket
subtype: task
title: "Return mutation warnings in result types instead of printing from core"
state: REVIEW
priority: P2
labels: []
epic: null
blocked_by: []
created: 2026-07-15T01:24:44+05:30
updated: 2026-07-16T18:29:55+05:30
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01KXH34CJ82T7TQ00HWTJHC1HV author=Shivam-Shivanshu ts=2026-07-15T01:24:44+05:30 -->
internal/core/mutate.go:92 commit() calls emitWarnings (warnings.go:10, fmt.Fprintln(os.Stderr)) though doc.go declares core the service layer; reads return datamodel.Warning in Result.StderrNotes (all tagged json:"-" — results.go:70,91,147,393 — the CLI prints them to stderr, respecting cmd.ErrOrStderr()). The layering defect is the transport: reads pass warnings as structured data across the core->CLI boundary letting the CLI choose the sink; mutations print to os.Stderr as a side effect inside core, so mutation warnings (except Move's hand-copied wipGuard subset — move.go:24,71,99) are unavailable to any programmatic consumer and bypass cobra's stderr writer. workon.go:64 also prints directly.

Fix: return warns from commit/commitMutation in the result types (Warnings on MutationResult/CreateResult like StderrNotes on reads); the CLI layer prints; delete emitWarnings from core; fix workon.go:64 the same way.

Verify: existing warning-emitting scenarios (forced transition, require guard, blockers) still print to stderr through the CLI; a redirected commandRunner buffer test showing warnings land in cmd.ErrOrStderr().

Files: internal/core/mutate.go, internal/core/warnings.go, internal/core/move.go, internal/core/workon.go, internal/datamodel/results.go, internal/cli
<!-- /kira:comment -->
