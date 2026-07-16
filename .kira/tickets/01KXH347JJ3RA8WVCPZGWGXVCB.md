---
id: 01KXH347JJ3RA8WVCPZGWGXVCB
number: CORE-20
aliases: []
type: ticket
subtype: task
title: "Automation payload: export and schema-register the hook-stdin contract; dedupe Event fields"
state: REVIEW
priority: P2
labels: []
epic: null
blocked_by: []
created: 2026-07-15T01:24:39+05:30
updated: 2026-07-16T19:43:12+05:30
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01KXH347RDN38PBC1MF32M19FK author=Shivam-Shivanshu ts=2026-07-15T01:24:39+05:30 -->
The hook-stdin payload struct (internal/automation/payload.go:12) is unexported, in no schema, covered only by a key-presence loop (internal/automation/test/automation_test.go:129 asserts 10 of 12 top-level keys) plus one txtar grep. Item's shape is frozen via ShowResult goldens; the genuinely unfrozen surface is the 'sync' key, nested Actor{name,email} and Change{old,new} shapes, and the KIRA_* env var names. Event.FromCategory is matched (match.go:19) but absent from payload and envMirror; Event.Commit is KIRA_COMMIT env-only. Event duplicates ItemID/Number/Type/Title alongside Item *ShowResult (automation.go:34-37); envMirror reads the flattened copy, payload reads ev.Item — a producer filling one copy makes env and JSON disagree.

Fix: export the payload type, add it to schema.topLevelTypes (widen the schema package doc from '--json output' shapes), add a full-payload golden. Add FromCategory and Commit (omitempty) to the payload plus KIRA_FROM_CATEGORY to envMirror (additive under payload_version 1). Delete the flattened Event quartet; envMirror derives nil-safely from ev.Item — caveat: Matches also reads the flattened ev.Type (match.go:13) and Item is nil for sync.completed, so extend the nil-safe derivation to match.go or keep Type.

Verify: full-payload golden test; env-var name assertions; sync.completed event still matches.

Files: internal/automation/payload.go, internal/automation/match.go, internal/core/automation.go, internal/schema/schema.go, internal/automation/test/automation_test.go
<!-- /kira:comment -->
