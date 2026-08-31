---
id: 01M1BYZCD7GRFHEY7JVCVNR1GG
number: KIRA-118
aliases: []
type: ticket
title: "TUI board screen performs store IO inside View(), against its own DONE ticket"
state: IN_PROGRESS
priority: P2
labels: []
epic: null
blocked_by: []
created: 2026-08-31T15:08:41+02:00
updated: 2026-08-31T16:12:06+02:00
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01M1BYZCN6DTS1CTGJ69GCMVS3 author=Shivam-Shivanshu ts=2026-08-31T15:08:41+02:00 -->
Convergent: found by both cli reviewers.
tui/boardscreen.go:227-228 - view() calls ensureLoaded, which calls m.store.Board(...) synchronously (:62-78). REVIEW.md:287 states the invariant 'no store I/O in View()'. Ticket TUI-1 (DONE) prescribed moving board/stats/detail loads onto the async tea.Cmd executor; stats got it (stats.go:69-80), board did not. activate() returns nil, so the View side effect is the real load trigger.
Scenario: invalidate() runs on every treeLoadedMsg, so with a refresh interval set, each tick makes the next board frame block the render loop on a full index read. On a large or cold store the UI freezes per tick.
Fix: mirror statsScreen - activate()/invalidate() dispatch an async boardLoadCmd, result arrives via a boardLoadedMsg, view() stays pure and renders a loading state.
<!-- /kira:comment -->
