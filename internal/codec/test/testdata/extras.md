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
parent: 01J8X7B1Q2W3E4R5T6Y7U8I9O0
blocked_by: [01J8X9F2M3W7VJQK8N5R6T1B0C]
estimate: 3
created: 2026-07-10T09:14:00+05:30
updated: 2026-07-12T11:02:00+05:30
future_bool: true
future_date: "2026-07-20"
future_int: 7
future_list: [alpha, beta]
future_null: null
future_quoted_null: "null"
future_scalar: whatever
---

## Description

The snapshot merge path drops updates when two feed threads race on the
same price level. Repro: `bench/burst_test --dup-updates=high`.
