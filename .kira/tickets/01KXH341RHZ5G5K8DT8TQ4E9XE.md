---
id: 01KXH341RHZ5G5K8DT8TQ4E9XE
number: CORE-9
aliases: []
type: ticket
subtype: bug
title: "id: live-number collisions last-write-win; bare-token candidates non-distinguishing; weak allocation"
state: REVIEW
priority: P1
labels: []
epic: null
blocked_by: []
created: 2026-07-15T01:24:33+05:30
updated: 2026-07-15T02:14:53+05:30
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01KXH341YAG8VGB0ZQ5YVHY0QN author=Shivam-Shivanshu ts=2026-07-15T01:24:33+05:30 -->
internal/id/resolve.go:71 liveByNumber[live]=u overwrites on collision (last in snapshot slice order wins); holdersOf (:86-91) returns the single winner while the alias path correctly reports ambiguity (full-token alias-two-holders path is already correct and tested, resolve_boards_test.go:47) — defects are confined to live-number collisions and bare-token candidates. resolveBare (:136-157) counts holders but returns fulls as candidates — one non-distinguishing candidate or inflated counts. HashNumber (number.go:55-58, 30 bits) has no uniqueness check against the snapshot (~0.05% collision at 1k items, feeding the last-wins bug); an all-digit suffix inflates HighestN to ~6 digits. Allocation policy duplicated in substance: allocateNumber (core/create.go:336-342) branches on cfg.ID.Style with a Snapshot, id.AllocFor on a bool with a precomputed next int. id.go:13-15 seeds math/rand with UnixNano for immutable identities. AmbiguousError says 'prefix %q' for non-prefix cases and leaks 'id:' into user output. SortKey (six core sort sites) and KeyOf (four consumers) have zero direct tests; NotFoundError.Suggestion is asserted nowhere (errx.Nearest itself IS tested).

Fix: make liveByNumber a map[string][]string and return live holders (preserving live-beats-alias precedence) so >1 yields AmbiguousError; build bare candidates from deduped holders via liveNumberByULID. Snapshot-aware hash allocator (retry on collision, never all-digit) used by both create and seed via one AllocFor entry point. crypto/rand.Reader for entropy. Neutral ambiguity wording without the 'id:' prefix; one splitLastDash helper (i>0) shared by KeyOf/ParseNumber/addBare.

Verify: SortKey/KeyOf/Suggestion table tests; a two-live-holders resolution test asserting AmbiguousError.

Files: internal/id/resolve.go, internal/id/number.go, internal/id/id.go, internal/core/create.go
<!-- /kira:comment -->
