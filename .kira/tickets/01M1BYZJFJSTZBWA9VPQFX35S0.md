---
id: 01M1BYZJFJSTZBWA9VPQFX35S0
number: KIRA-130
aliases: []
type: ticket
title: "Watermark-persist failure discards already-committed close results"
state: DONE
priority: P2
labels: []
epic: null
blocked_by: []
created: 2026-08-31T15:08:47+02:00
updated: 2026-08-31T16:12:17+02:00
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01M1BYZJQPG8B6YJQZ5NA0NTBW author=Shivam-Shivanshu ts=2026-08-31T15:08:48+02:00 -->
core/closes.go:54-58 - after the moves have been committed, a PersistLandedWatermark error returns (nil, nil, err), dropping the closed set and notes.
Scenario: closes commit successfully but the cache-dir write fails -> kira sync / index --closes reports only an error and the user never learns which tickets were just closed.
Fix: return closed and notes alongside the error, or demote the watermark failure to a warning note.
<!-- /kira:comment -->
