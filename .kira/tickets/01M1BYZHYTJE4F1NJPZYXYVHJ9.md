---
id: 01M1BYZHYTJE4F1NJPZYXYVHJ9
number: KIRA-129
aliases: []
type: ticket
title: "Tree screen shows stale state after a board move when auto-refresh is off"
state: REVIEW
priority: P2
labels: []
epic: null
blocked_by: []
created: 2026-08-31T15:08:47+02:00
updated: 2026-08-31T16:12:16+02:00
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01M1BYZJ7492NQR1F4Y0KZ52WG author=Shivam-Shivanshu ts=2026-08-31T15:08:47+02:00 -->
tui/app.go:160-165 - boardMovedMsg only calls bs.applyMove; nothing refreshes tree data or the tree screen's detail cache. refreshEvery defaults to 0 (datamodel/uiprefs.go:14-18 - unset means no auto-refresh), so no tick ever reconciles.
Repro: default config, board view, move a card TODO->IN_PROGRESS with L, then press 1: the tree still renders [TODO] and its detail pane the old state, indefinitely, until a manual r or a command-bar mutation.
Fix: enqueue refreshCmd after a successful move in the boardMovedMsg handler, as commandResultMsg already does.
<!-- /kira:comment -->
