---
id: 01M1BZ2XQ0JQMNGEN762DFT0SX
number: KIRA-143
aliases: []
type: ticket
title: "DECISION: --field k=v filtering on kira list"
state: REVIEW
priority: P3
labels: []
epic: null
blocked_by: []
created: 2026-08-31T15:10:37+02:00
updated: 2026-08-31T23:51:34+02:00
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01M1BZ2XZG1QP5Q3HVE2QESBKF author=Shivam-Shivanshu ts=2026-08-31T15:10:37+02:00 -->
The design doc's 'CLI field filtering and flags' section specifies a generic, repeatable, equality-only --field k=v on list that desugars into the query grammar, with --field TAB completing field names and --field state=TAB completing values. Only edit --field exists today; list has no such flag.
Everything it needs now exists: phase 3c made the query grammar schema-aware and threaded the extras spec into both Parse and Options, so this is a thin desugaring layer plus completion, not new machinery.
Decide whether to build it; if yes this is a small, well-specified ticket.
<!-- /kira:comment -->
