---
id: 01KY766NCPKD4ZRWYQY0415K1C
number: CORE-39
aliases: []
type: ticket
title: "Migrate config-backed entities (board, sprint) to file-backed entity storage"
state: TODO
priority: P3
labels: []
epic: null
blocked_by: []
created: 2026-07-23T15:21:42+05:30
updated: 2026-07-23T15:21:42+05:30
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01KY766NK7H097NT4RR5SMS1Q5 author=Shivam-Shivanshu ts=2026-07-23T15:21:42+05:30 -->
Follow-up to CORE-3 (storage model B). CORE-3 phase 1 keeps storage model A: schema describes shape, but board/sprint instances remain config.yaml list entries while ticket/epic are file-backed. This ticket tracks the end state where config-backed entities also become file-backed under the entity-schema system, so all entities share one storage backend. See docs/design/CORE-3-extensible-schema.md (Phasing, Phase 6).
<!-- /kira:comment -->
