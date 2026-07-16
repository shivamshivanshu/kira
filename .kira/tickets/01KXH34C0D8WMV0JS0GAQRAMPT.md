---
id: 01KXH34C0D8WMV0JS0GAQRAMPT
number: CORE-25
aliases: []
type: ticket
subtype: bug
title: "core correctness batch: SubjectPrefix %-escapes, locked-config guards, --since parsing, Overdue"
state: DONE
priority: P2
labels: []
epic: null
blocked_by: []
created: 2026-07-15T01:24:44+05:30
updated: 2026-07-16T18:45:00+05:30
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01KXH34C6F86BWWES0DWG78B7Q author=Shivam-Shivanshu ts=2026-07-15T01:24:44+05:30 -->
Five small correctness defects:
- move.go:77, reconcile.go:42, link.go:45-47, sprint.go:131 do fmt.Sprintf(cfg.Commit.SubjectPrefix+"...") — a '100% kira: ' prefix yields '%!k(MISSING)' subjects (validate only rejects newlines). Fix: Sprintf("%s%s...", prefix, ...).
- board.go:187-190 checks cfg.ActiveBoards() before the lock while mutateConfig exists precisely to validate against the locked parse; configset.go:23/sprint.go:54 use stale cfg.Commit. Fix: move the guard inside the closure on locked; switch configset/sprint to locked.Commit.
- create.go:54-70 would call s.Link with ref "" (only cli/create.go:92 guards). Fix: validate Blocking-requires-Here in core.Create.
- stats.go:87 'it.Created < opts.Since' is wrong across timezone offsets. Fix: validate/parse --since with time.DateOnly anchored to a chosen timezone and compare against the parsed Created timestamp (timex.CompareRFC3339 parses full RFC3339, so date-only Since can't be passed to it verbatim).
- timex.go:22 'due < now.Format(DateOnly)' trusts hand-edited due values; the done bool param forces both callers to reimplement category==Done + nil guard. Fix: parse due in Overdue (false on failure), narrow it to (due, now), hoist datamodel.IsOverdue(due *string, category, now) shared by core/now and tui (delete the tui wrapper).

Verify: Overdue/HumanSince table tests; a subject test with a %-containing prefix.

Files: internal/core/move.go, internal/core/reconcile.go, internal/core/link.go, internal/core/sprint.go, internal/core/board.go, internal/core/configset.go, internal/core/create.go, internal/core/stats.go, internal/timex/timex.go, internal/datamodel
<!-- /kira:comment -->
