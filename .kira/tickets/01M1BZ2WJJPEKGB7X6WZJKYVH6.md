---
id: 01M1BZ2WJJPEKGB7X6WZJKYVH6
number: KIRA-141
aliases: []
type: ticket
title: "DECISION: per-type identity is declared in schemas but inert"
state: DONE
resolution: done
priority: P3
labels: []
epic: null
blocked_by: []
created: 2026-08-31T15:10:36+02:00
updated: 2026-09-01T10:26:00+02:00
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01M1BZ2WVJW6PC15Q8G6PA52D1 author=Shivam-Shivanshu ts=2026-08-31T15:10:36+02:00 -->
entityschema.Identity (schema.go:42) carries style and prefix, and builtin ticket.json/epic.json both declare identity {style: sequential, prefix: KIRA} - but nothing generates ids from it. Every type shares the single project sequence, so a config-defined custom type gets KIRA-1 (see tests/e2e/testdata/script/custom-type.txtar). schema.go's own comment says Identity is descriptive only.
Decide: implement per-type sequences and prefixes, or delete the fields so the schema stops advertising a capability that does not exist. The status quo is the worst of the two - a user reading the shipped schema reasonably expects a prefix that never takes effect.
<!-- /kira:comment -->

<!-- kira:comment id=01M1EACEA45H3DZE0YBQP58XQS author=Shivam-Shivanshu ts=2026-09-01T13:06:32+02:00 -->
Reversed by the origin/main reconciliation merge (cfd7e19). This ticket removed the per-type `identity` declaration on the grounds that it was inert. It is not: origin added a live `Identity` type read by `Schema.ConfigBacked()`, which gates whether a schema is config-backed or a real item type.

Removing it from board.json made ConfigBacked() false for boards, so the board schema was treated as an item type and its fields (key, name, default, archived, description) leaked into the item query grammar. `kira list --query 'description = x'` then silently matched nothing instead of reporting that a body-placed field cannot be filtered — and which 'description' won depended on map iteration order across schemas.

Caught by origin's own e2e (schema-custom-fields.txtar:58). All three builtin schemas are now byte-identical to origin's, identity restored.
<!-- /kira:comment -->
