---
id: 01M1BYZBCC1KHKXKE9051N8YKY
number: KIRA-116
aliases: []
type: ticket
title: "kira create silently drops schema-declared extras from templates, editor buffers and --from-file"
state: TODO
priority: P2
labels: []
epic: null
blocked_by: []
created: 2026-08-31T15:08:40+02:00
updated: 2026-08-31T15:08:40+02:00
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01M1BYZBM9A71GADFWAFX3Z7P1 author=Shivam-Shivanshu ts=2026-08-31T15:08:40+02:00 -->
Convergent: found by both the data contextual and core contextual DX-1 reviewers. Gap from CORE-3 phase 3b.
core/draft.go:13-38 uses a fixed draft struct and DecodeFrontmatter without KnownFields, so unknown keys vanish silently; itemFromDraft (create.go:247) never carries Extras. The design doc says create/edit/validate honor declared fields - edit and validate do, create does not.
Repro: schema declares 'severity'; kira create ticket --from-file f.md with 'severity: high' -> created file has no severity, no warning, exit 0. Same field sets fine via kira edit --field. Typing it into the create editor buffer is discarded too, despite the validate loop.
Fix: capture non-builtin frontmatter in the draft (reuse codec's extraValue), thread Extras through itemFromDraft, and reject rather than drop keys that are neither builtin nor declared - matching the rule in KIRA-109.
<!-- /kira:comment -->
