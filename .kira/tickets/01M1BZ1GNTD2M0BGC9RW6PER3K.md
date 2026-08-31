---
id: 01M1BZ1GNTD2M0BGC9RW6PER3K
number: KIRA-134
aliases: []
type: ticket
title: "Index and codec edge cases: stale-schema cache, estimate != semantics, CRLF drift, orphan rows"
state: TODO
priority: P3
labels: []
epic: null
blocked_by: []
created: 2026-08-31T15:09:51+02:00
updated: 2026-08-31T15:09:51+02:00
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01M1BZ1GXX8VWJVWCRDWB5PQQN author=Shivam-Shivanshu ts=2026-08-31T15:09:51+02:00 -->
Grouped from the data blind DX-1 review.
(a) index/read.go:13-23 ReadCached documents 'returns nil, nil if no cache exists' but opens the db read-only with no PRAGMA user_version check and runs the current column list. After a schema bump, shell completion (its only caller) hits 'no such column' until something rebuilds. Contract and behaviour disagree.
(b) query estimatePred requires it.Estimate != nil, so an item with no estimate never matches 'estimate != 5' - yet scalarPred makes an ownerless item match 'owner != alice'. Two conventions for != on absent values; users cannot predict which applies.
(c) codec/parse.go:94-113 splitDocument normalizes CRLF; its near-twin cutAfterFrontmatter (:115-130) does not, so AppendCommentToDocument fails on a CRLF file with a misleading 'cannot locate the closing frontmatter fence'. Currently unreachable only because guardWritable blocks CRLF items upstream.
(d) index/schema.go:65-88 - events, event_heads and commit_links have no FK to items and are not cleared by full() or refresh; deleting a ticket leaves its rows behind until a schema bump. Harmless today (all reads join items) but unbounded growth and a trap for any future scan.
<!-- /kira:comment -->
