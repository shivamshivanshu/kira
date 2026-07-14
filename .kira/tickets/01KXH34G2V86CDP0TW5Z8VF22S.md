---
id: 01KXH34G2V86CDP0TW5Z8VF22S
number: TUI-2
aliases: []
type: ticket
subtype: bug
title: "Command bar '.' and yank target the tree screen regardless of active view"
state: REVIEW
priority: P1
labels: []
epic: null
blocked_by: []
created: 2026-07-15T01:24:48+05:30
updated: 2026-07-15T02:56:40+05:30
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01KXH34G8T4VS144423KE8H51Y author=Shivam-Shivanshu ts=2026-07-15T01:24:48+05:30 -->
focusedNumber() (internal/tui/cmdbar.go:147-156) calls m.treeScreen() — a screens-map lookup regardless of m.view — never m.current(); boardScreen tracks its own focus (boardscreen.go:209-214) which is ignored. On the board (and stats view), ':move . DONE' moves whatever the invisible tree cursor last pointed at — wrong-target mutation. yank.go:8 type-asserts *treeScreen — view-aware and fails safe (silent no-op on board/stats, never wrong-target) — but globalKeys advertises y/Y everywhere (app.go:34,183-188) with no feedback. The screen interface (app.go:40-48) already defines focusedID() for exactly this.

Fix: route both through the active screen — m.current().focusedID() or add a focused-item accessor to the screen interface (boardScreen supplies board.selected(); statsScreen returns none and drops y/Y from its hints). Note: focusedID() substitutes a ULID rather than a KIRA-n number; the CLI resolver accepts full ULIDs (id/resolve.go:100-107) so this works, but TestCommandBarForwardsTokenizedArgv asserts the number form ("KIRA-100") and must be updated — or the accessor should return the display number for tree/board.

Verify: cmdbar test asserting '.' resolves the board's focused card when view==viewBoard; a yank-on-board test.

Files: internal/tui/cmdbar.go, internal/tui/yank.go, internal/tui/app.go, internal/tui/boardscreen.go
<!-- /kira:comment -->
