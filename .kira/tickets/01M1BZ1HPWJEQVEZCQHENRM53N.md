---
id: 01M1BZ1HPWJEQVEZCQHENRM53N
number: KIRA-136
aliases: []
type: ticket
title: "Extras coverage gaps: log/blame invisibility, --at schema provenance, immutability, did-you-mean"
state: TODO
priority: P3
labels: []
epic: null
blocked_by: []
created: 2026-08-31T15:09:52+02:00
updated: 2026-08-31T15:09:52+02:00
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01M1BZ1HYF26V0P9047JS90Y90 author=Shivam-Shivanshu ts=2026-08-31T15:09:52+02:00 -->
Grouped follow-ups to CORE-3 phases 3a-3c, from the core and data contextual DX-1 reviews.
(a) core/eventparse.go:91-106 derives events only for datamodel.FrontmatterKeys, and blame.go:46-48 iterates the same set, so an extras change is invisible to kira log/blame/changes even though the commit subject says 'edit severity'. Decide: extend the diff-walker to declared extras (the schema is available at derivation time) or scope this explicitly in the design doc.
(b) core/list.go:87-111 queryExtras uses s.schemas() - the WORKING TREE schemas - while --at items and config come from the historical tree (treeish loads config only, never .kira/schema). 'kira list --at v1.0 -q severity=HIGH' errors unknown-field although every item in that snapshot has it; a scalar-to-list retype decodes old data with today's flags. Fix: load schemas from the tree-ish, or document current-schema semantics for --at.
(c) core/edit.go:202-216 blocks --field writes to Immutable schema extras, but the full-editor and --from-file paths change or delete them freely; restoreImmutable covers builtins only.
(d) core/hints.go:12-17 builds the --field did-you-mean from datamodel.EditableFields only, so a typo'd declared extra gets no suggestion - the design doc explicitly wants schema-sourced suggestions.
<!-- /kira:comment -->
