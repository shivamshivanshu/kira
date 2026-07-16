---
id: 01KXH34E7N9S8YNP51AJA72VDR
number: DATA-8
aliases: []
type: ticket
subtype: task
title: "query: field-spec table, Match typechecking and notes, datetime day comparison, priority capture"
state: IN_PROGRESS
priority: P2
labels: []
epic: null
blocked_by: []
created: 2026-07-15T01:24:46+05:30
updated: 2026-07-16T18:58:36+05:30
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01KXH34EDFECB6FQKEFXQCVF29 author=Shivam-Shivanshu ts=2026-07-15T01:24:46+05:30 -->
Batch of query-package defects and structure:
- Match (internal/query/eval.go:57-66) builds predExpr with num zero so estimate Matches compare against 0 and blocked errors self-contradictorily (latent — sole caller core/list.go:140 passes only scalar fields); compilePred's sprint WarnNoActiveSprint lands in a discarded compiler and core/list.go:146-148 duplicates the rule (live). Fix: Match runs typeCheckValue, rejects bool fields, returns notes; delete core's hand-rolled check.
- datePred formats the literal's own-zone day vs localDay's t.Local() — same instant fails equality for IST users (created/updated/activity only; due is DateOnly raw-string compare). Canonicalize/reject time-of-day literals.
- scalarPred lets empty owner satisfy owner!=x while datePred/estimatePred never match absent fields — pick one != semantic (SQL-style absent-never-matches) or document.
- isAlwaysPresent omits fieldActivity (backfilled on every load, NOT NULL column). parsePrimary falls to termExpr for unknown words before comparison ops, making unknownFieldErr unreachable from Parse — return it when an unknown word precedes a comparison op.
- Order.Keyer rebuilds the priority index from cfg.Priorities.Values at sort time while predicates rank from opts.Priorities — two different sources; capture the priority index in Order at Compile time.
- Structure: metadata split over fields/accessors/isDateField/isBoolField/isListField/allowsOrderedCmp/isAlwaysPresent plus a hardcoded field list in an error string, with parser.go:92-114 restating datamodel Key* constants and core/list.go:131-135 a third copy — one field-spec table {kind, accessor, alwaysPresent} deriving all of it, names aliased to datamodel.Key*, query.Fields() export for cli wiring. Cleanups: parseIn stores predExpr children, collapse compileBool (dead fallback), native &&/|| closures, fold test/ into the package behind one fixture().

Verify: table tests per semantic decision; existing query/order suites green.

Files: internal/query/eval.go, internal/query/parser.go, internal/query/test, internal/core/list.go
Depends on: ORDER BY board silently no-ops — missing board accessor; Deduplicate blocker-open walk: shared CategoryOf + OpenBlockers helper
<!-- /kira:comment -->
