---
id: 01KXH343JHTSQ2H5GS1NVAC9QB
number: DATA-6
aliases: []
type: ticket
subtype: task
title: "Deduplicate blocker-open walk: shared CategoryOf + OpenBlockers helper"
state: IN_PROGRESS
priority: P2
labels: []
epic: null
blocked_by: []
created: 2026-07-15T01:24:35+05:30
updated: 2026-07-16T16:02:57+05:30
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01KXH343QW8EKN37VXWFHPG7DN author=Shivam-Shivanshu ts=2026-07-15T01:24:35+05:30 -->
query/eval.go:378 walks cfg.Workflows[typ].States exactly like core/workflow.go:64; blockedPred (eval.go:249-262) and blockersClosedGuard (core/move.go:157-176) both re-derive 'blocker is open' (byID map, skip unresolvable, skip unknown category, compare CategoryDone). The two copies agree today (unknown/unresolvable treated as satisfied in both) — the defect is drift risk: a semantic tweak (e.g. dropped-resolution blockers, cf. core/progress.go:45) must land in two packages. Nothing asserts the shared truth table; core has NO tests touching blockersClosedGuard at all (grep 'blocker' in internal/core/*_test.go is empty), while query_test.go:259-264 pins blocked membership.

Fix: add func (c *Config) CategoryOf(typ, state string) (Category, bool) to internal/datamodel next to Workflow; delete both package-local copies; add a shared OpenBlockers helper consumed by blockedPred and blockersClosedGuard — it must return skipped blockers (unresolvable/unknown-category) alongside open ones, since the move guard warns on those while the query predicate is silent.

Verify: one table test asserting query 'blocked' membership matches whether the move guard reports open blockers per case (missing blocker, orphan-type, doing, done).

Files: internal/datamodel/config.go, internal/query/eval.go, internal/core/move.go, internal/core/workflow.go
<!-- /kira:comment -->
