---
id: 01J8X8Q7RZTN5Y3VXW2A9K4E7F
number: KIRA-142
aliases: []
type: ticket
title: "Fix race in order-book snapshot merge"
state: IN_PROGRESS
priority: P1
owner: shivam
reporter: shivam
labels: [bug, orderbook]
epic: 01J8X7B1Q2W3E4R5T6Y7U8I9O0
blocked_by: [01J8X9F2M3W7VJQK8N5R6T1B0C]
estimate: 3
created: 2026-07-10T09:14:00+05:30
updated: 2026-07-12T11:02:00+05:30
---

## Description

The snapshot merge path drops updates when two feed threads race on the
same price level. Repro: `bench/burst_test --dup-updates=high`.

## Acceptance criteria
- [ ] TSan clean on order_book_test
- [ ] No p99 regression on hot path

## Comments

<!-- kira:comment id=01J8XA1F6Q2N9K3M7V0R5T8B4C author=shivam ts=2026-07-11T18:30:00+05:30 -->
Confirmed repro with TSan; missing acquire fence on the consumer side.
<!-- /kira:comment -->
