---
id: 01M1BYZDXFDBM03RVW7TXX6XSK
number: KIRA-121
aliases: []
type: ticket
title: "resolve --interactive silently no-ops on arbitrated extra fields"
state: DONE
priority: P2
labels: []
epic: null
blocked_by: []
created: 2026-08-31T15:08:43+02:00
updated: 2026-08-31T16:12:09+02:00
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01M1BYZE5K81ZV25YK27J0F0X3 author=Shivam-Shivanshu ts=2026-08-31T15:08:43+02:00 -->
merge.Merge arbitrates extras (Arbitrated comes from ChangedFields, which includes them - datamodel/fields.go:91-98), but pickFields/fieldString (core/resolve.go:123-144) consult only datamodel.Field, which misses extras.
Scenario: both branches change declared extra 'severity'; kira resolve --interactive prompts 'severity: [o]urs= [t]heirs=' with EMPTY values, and answering o or t changes nothing - the auto-merge winner is written regardless.
Fix: fall back to datamodel.ExtraField(f, list) for Get/Copy when datamodel.Field misses, taking list-ness from DeclaredExtra.
<!-- /kira:comment -->
