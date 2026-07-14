---
id: 01KXH343735RY6JYMNK8EWFBAZ
number: DATA-5
aliases: []
type: ticket
subtype: bug
title: "ORDER BY board silently no-ops — missing board accessor"
state: TODO
priority: P1
labels: []
epic: null
blocked_by: []
created: 2026-07-15T01:24:35+05:30
updated: 2026-07-15T01:24:35+05:30
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01KXH343CJYZK7GMYRB0H9F6QZ author=Shivam-Shivanshu ts=2026-07-15T01:24:35+05:30 -->
parser.go:194-216 accepts any non-list, non-bool member of fields; Order.Keyer's default branch calls scalarGet("board") but accessors (internal/query/eval.go:265-281) has no fieldBoard entry, so scalarGet returns the constant-"" getter; Less(null,null) is false both ways and core's sortMatched falls to the precedence tiebreak — identical to no ORDER BY. Equality filtering only works because compilePred special-cases fieldBoard (eval.go:144-147, comparing strings.ToUpper(id.KeyOf(it.Number))) and bypasses the accessors map.

Fix: add fieldBoard to accessors returning strings.ToUpper(id.KeyOf(it.Number)) for consistency with the filter path (note: compilePred will NOT automatically reuse it — either leave its dedicated case or refactor it onto the accessor); compileIsEmpty stays safe via isAlwaysPresent. Alternative: reject board in parseOrder.

Verify: TestEval rows for owner=@me (set and unset -> error, eval.go:129-131 is asserted nowhere), board=KIRA, and an ORDER BY board row in TestOrderBy (test/order_test.go currently omits board).

Files: internal/query/eval.go, internal/query/parser.go, internal/query/test/order_test.go
<!-- /kira:comment -->
