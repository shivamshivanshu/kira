---
id: 01M1BZ2Y8FYS2YSJVHG241FQXM
number: KIRA-144
aliases: []
type: ticket
title: "DECISION: cross-schema field-name collisions resolve first-wins by alphabetical schema order"
state: IN_PROGRESS
priority: P3
labels: []
epic: null
blocked_by: []
created: 2026-08-31T15:10:38+02:00
updated: 2026-08-31T23:51:35+02:00
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01M1BZ2YGZEYM4J6FR9WTQAK50 author=Shivam-Shivanshu ts=2026-08-31T15:10:38+02:00 -->
From the phase 3c review. core.queryExtras dedupes the extras spec across all loaded schemas first-wins, iterating schemas in sorted order. If ticket.json declares 'severity' as an enum scalar and epic.json declares it as a list, one schema's traits silently win for both types - deterministic, but arbitrary and unexplainable to a user.
Options: (a) namespace the query field by type; (b) reject conflicting declarations at schema load with a clear error; (c) keep first-wins and document it. The reviewer flagged it as needing a design decision rather than a quiet patch, and I agree.
<!-- /kira:comment -->
