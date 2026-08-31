---
id: 01M1BZ2WJJPEKGB7X6WZJKYVH6
number: KIRA-141
aliases: []
type: ticket
title: "DECISION: per-type identity is declared in schemas but inert"
state: IN_PROGRESS
priority: P3
labels: []
epic: null
blocked_by: []
created: 2026-08-31T15:10:36+02:00
updated: 2026-08-31T23:51:32+02:00
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01M1BZ2WVJW6PC15Q8G6PA52D1 author=Shivam-Shivanshu ts=2026-08-31T15:10:36+02:00 -->
entityschema.Identity (schema.go:42) carries style and prefix, and builtin ticket.json/epic.json both declare identity {style: sequential, prefix: KIRA} - but nothing generates ids from it. Every type shares the single project sequence, so a config-defined custom type gets KIRA-1 (see tests/e2e/testdata/script/custom-type.txtar). schema.go's own comment says Identity is descriptive only.
Decide: implement per-type sequences and prefixes, or delete the fields so the schema stops advertising a capability that does not exist. The status quo is the worst of the two - a user reading the shipped schema reasonably expects a prefix that never takes effect.
<!-- /kira:comment -->
