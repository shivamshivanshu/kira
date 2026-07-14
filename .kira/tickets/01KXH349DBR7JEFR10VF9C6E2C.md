---
id: 01KXH349DBR7JEFR10VF9C6E2C
number: KIRA-103
aliases: []
type: ticket
subtype: task
title: "Extract shared stats line formatter (cli/tui drift, codec import leak)"
state: TODO
priority: P2
labels: []
epic: null
blocked_by: []
created: 2026-07-15T01:24:41+05:30
updated: 2026-07-15T01:24:41+05:30
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01KXH349K9SBJH2GF7WNCWVZPQ author=Shivam-Shivanshu ts=2026-07-15T01:24:41+05:30 -->
cli/stats.go:67/:91 vs tui/stats.go:95/:128 carry the same 'X/Y done (Z%)' and 'p50 %s p90 %s n=%d' strings with subtle divergence (scope header, degraded-N, 'no data', and the TUI omits the '(days)' unit on cycle/lead time so its p50/p90 render unitless). codec.EmitFloat (serialize.go:89) is strconv.FormatFloat exported for frontmatter emission; cli/stats.go:91 and tui/stats.go:128 call it purely for terminal output — a change to canonical YAML float emission silently changes UI text. Repo precedent for the extraction: internal/showfmt.

Fix: extract CompletionLine/PercentileLine into a small formatter (extend showfmt or add statsfmt) using strconv directly; cli and tui wrap with their own decoration; drop the codec imports at both stats call sites (note tui package still imports codec via detailfull.go:88 — only the stats-file import goes).

Verify: table-test the formatter once; eyeball cli and tui stats output parity.

Files: internal/cli/stats.go, internal/tui/stats.go, internal/showfmt
<!-- /kira:comment -->
