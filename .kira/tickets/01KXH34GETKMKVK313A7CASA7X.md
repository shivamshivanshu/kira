---
id: 01KXH34GETKMKVK313A7CASA7X
number: TUI-3
aliases: []
type: ticket
subtype: bug
title: "Board screen: applyMove reloads unscoped board and leaves stale raw snapshot"
state: IN_PROGRESS
priority: P1
labels: []
epic: null
blocked_by: []
created: 2026-07-15T01:24:48+05:30
updated: 2026-07-15T02:01:16+05:30
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01KXH34GMPT1Q08CZC4E5N7M6X author=Shivam-Shivanshu ts=2026-07-15T01:24:48+05:30 -->
internal/tui/boardscreen.go:76-79 applyScope() renders scopedBoard(s.raw, s.scope), but applyMove (:182) does s.board.load(msg.board) where msg.board is the full unfiltered store.Board(cfg, BoardOpts{}) (cmd.go:71), and never updates s.raw — after a move under an active scope, other boards' cards appear; a later 'b' picker re-filters a stale pre-move snapshot. No test exercises the applyMove/boardMovedMsg reload path at all (board_render_test.go:226-261 only verifies the move request reaches core.Move and on-disk state; grep for applyMove|boardMovedMsg in *_test.go is empty).

Fix: in applyMove — s.raw = msg.board; s.applyScope(); s.board.focusByID(msg.cardID); s.syncPeek(m).

Verify: move-under-scope test asserting other boards' cards stay hidden, s.raw reflects the post-move board, and s.board.focusByID survives the scoped reload.

Files: internal/tui/boardscreen.go, internal/tui/board_render_test.go
<!-- /kira:comment -->
