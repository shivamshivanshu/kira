---
id: 01M1BYZEYPK0CXJBB0ZBJGHJRY
number: KIRA-123
aliases: []
type: ticket
title: "Board cards wrap instead of truncating in narrow columns, misaligning the board"
state: TODO
priority: P2
labels: []
epic: null
blocked_by: []
created: 2026-08-31T15:08:44+02:00
updated: 2026-08-31T15:08:44+02:00
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01M1BYZF6HEP45VHCTEM05PG88 author=Shivam-Shivanshu ts=2026-08-31T15:08:44+02:00 -->
tui/board.go:124-143 renderCard width-budgets only the title (fitWidth(it.Title, w-2-...)); the fixed priority/glyph/number segments are not budgeted, so the assembled line can exceed w and the Width(w) style (board.go:99) wraps it onto a second line.
Repro: kira board --plain at width 80 with this repo's 5 states renders cards as '!   [~] CORE-' / '40' across two lines, and rows in adjacent columns stop aligning. The same renderer serves the interactive board, where the extra line silently pushes column content past its allotted height.
Fix: truncate the finished card line to w (fitWidth on the joined segments, or truncate the number before the title) so a card is always exactly one line.
<!-- /kira:comment -->
