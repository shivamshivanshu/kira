---
id: 01M1BYZEE036HDF20TKGW96J43
number: KIRA-122
aliases: []
type: ticket
title: "kira show dumps raw comment markers where the TUI renders them"
state: IN_PROGRESS
priority: P2
labels: []
epic: null
blocked_by: []
created: 2026-08-31T15:08:43+02:00
updated: 2026-08-31T16:12:10+02:00
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01M1BYZEP0Y0BD8K031EBKRKN7 author=Shivam-Shivanshu ts=2026-08-31T15:08:43+02:00 -->
cli/show.go:99-101 prints r.Body verbatim; core/show.go:103-104 sets Body to the raw marker-laden text AND a parsed Comments slice the human renderer never uses. The TUI does it right (tui/detailfull.go:90 renders codec.Description(res.Body) plus a formatted Comments section).
Repro: kira show TUI-1 prints '<!-- kira:comment id=... author=... -->' machine markers around every comment; the same ticket in the TUI shows a clean Comments section. Clear CLI/TUI divergence on identical data.
Fix: in renderShow print codec.Description(r.Body) and format r.Comments the way the TUI does. --json already carries both.
<!-- /kira:comment -->
