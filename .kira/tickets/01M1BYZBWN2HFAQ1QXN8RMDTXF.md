---
id: 01M1BYZBWN2HFAQ1QXN8RMDTXF
number: KIRA-117
aliases: []
type: ticket
title: "Collision detection is case-sensitive while resolution is case-insensitive, leaving unrepairable pairs"
state: REVIEW
priority: P2
labels: []
epic: null
blocked_by: []
created: 2026-08-31T15:08:40+02:00
updated: 2026-08-31T15:48:08+02:00
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01M1BYZC503B6NE842N9D11X93 author=Shivam-Shivanshu ts=2026-08-31T15:08:41+02:00 -->
Convergent: found by both data reviewers.
reconcile/reconcile.go:19-21 keys liveHolders by the raw it.Number, and doctor/collision.go:14-25 groups by exact string. Everything else compares case-insensitively: id/resolve.go upper-cases numbers/aliases/tokens, id/number.go HighestN uses EqualFold and takenNumbers uses ToUpper.
Scenario: two items numbered KIRA-5 and kira-5 (hand edit, or a differently-cased board key). Every resolve of 'kira-5' returns AmbiguousError, but Plan sees two distinct map keys, reports no collision, and doctor cannot repair it - a permanently unresolvable pair.
Fix: key liveHolders/live/alias maps by strings.ToUpper(number), keeping a representative original for the report.
<!-- /kira:comment -->
