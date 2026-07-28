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

<!-- kira:comment id=01KYKDXR64CPQ2PBNCQT0NVWQ0 author=Shivam-Shivanshu ts=2026-07-28T09:27:32+05:30 -->
Deferred 2026-07-28 after assessment, kept open at P3.

Scope is small in data terms (Board = 5 scalar fields, Sprint = 4) but the change is not: ~970 lines of board/sprint code across core/cli/config, a new storage path, a migration for every existing repo, merge-driver and doctor coverage, and rewired JSON output shapes.

Three reasons not to do it now:
1. This ticket cites CORE-3 Phasing Phase 6, which is explicitly marked 'maybe never -- evaluate whether the hybrid is good enough first'. The hybrid is working; nothing has surfaced as painful in practice.
2. A board key is not just data, it is a numbering namespace embedded in every ticket number (CORE-39). File identity would give boards a second identity alongside a key that is already stable, and forces a rename/renumber story that does not exist today.
3. The benefit is thin: per-entity git history (config.yaml already has history via git log -p), comments on a board/sprint, and kira show <board>.

The schema engine already models both backends cleanly via identity.style: config, so this is not blocking the CORE-3 follow-through work (CORE-50).

Revisit if the two-backend split starts costing something concrete -- e.g. if boards/sprints need comments or per-entity history for real, or if a third config-backed entity type appears.
<!-- /kira:comment -->
