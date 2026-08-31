---
id: 01M1BZ2YSDBE14SH7419STRR3S
number: KIRA-145
aliases: []
type: ticket
title: "DECISION: typed extras serialize as strings in --json"
state: DONE
priority: P3
labels: []
epic: null
blocked_by: []
created: 2026-08-31T15:10:38+02:00
updated: 2026-08-31T23:51:36+02:00
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01M1BZ2Z2RX5H3AKMS1ZZEA7YJ author=Shivam-Shivanshu ts=2026-08-31T15:10:38+02:00 -->
An extra declared 'type: int' with value 8 appears in list --json and show --json as "8", not 8. datamodel.ExtraJSON renders as anyOf[string, string[]] because extras store rendered canonical YAML text (the encoding that makes 'points: 42' survive a rewrite as an int rather than being requoted).
Consequence: a --json consumer must read the schema to know how to interpret an extra. Defensible - typing is by schema, not by JSON shape, matching how kira treats due and priority - but worth a deliberate call rather than an accident of the storage format.
Decide: keep strings and document, or type the JSON per FieldDef (which makes the generated output schema per-field rather than static).
<!-- /kira:comment -->
