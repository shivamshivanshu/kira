---
id: 01KXH34FPWHQQSD6763B2HMQJ9
number: TUI-1
aliases: []
type: ticket
subtype: bug
title: "TUI batch: cardWindow math, synchronous IO in View, notice handling, picker/chord dedup"
state: IN_PROGRESS
priority: P2
labels: []
epic: null
blocked_by: []
created: 2026-07-15T01:24:47+05:30
updated: 2026-07-15T02:22:43+05:30
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01KXH34FWVVX89WX94YDT50RCK author=Shivam-Shivanshu ts=2026-07-15T01:24:48+05:30 -->
Correctness: cardWindow (internal/tui/board.go:139-157) — total=10/capacity=5/focusRow=9 renders 4 cards, no '+N more', a blank padded slot; no test references it. board.go:199 advertises 'n new' with no handler anywhere. boardscreen.go:195-202 back() always jumps to tree vs stats' quit; app.go:291-296 pushes self-referential jump entries. boardscreen.go:69/:170 show raw err.Error() vs firstNonEmptyLine everywhere else. run.go:57-59 records debug.Stack() after Run() returned — the caller's stack, not the panic's. stats.go:79-84 falls back on ANY error, doubling the load and hiding root causes.

Perf: boardscreen.go:216 view()->reload->store.Board and stats.go:59 view()->two full Stats queries, invalidated per treeLoadedMsg tick (only when cfg.UI.Tui.RefreshInterval() > 0); detailhost sync->store.Show re-reads the whole index to resolve a ULID the TUI already has, cache wiped per tick.

Structure: yank.go:43/boardscope.go:26 duplicate picker plumbing with two mutually-exclusive model pointers; pendingG/pendingZ chord logic x4; app.go:123-131 type-asserts all three screens (an invalidate() interface method removes the board/stats asserts; treeScreen's setData seam remains); filter.go:24 names invert core's return values.

Fix: fix cardWindow (grow slots when hidden==0, above-window indicator) with a table test; move board/stats/detail loads onto the existing async tea.Cmd executor (add a core Show-by-ULID entry point skipping the resolve pass); q returns false from board back() (or route through switchView with updated hint); early-return switchView on same view; setNotice helper wrapping firstNonEmptyLine; run with tea.WithoutCatchPanics() so guardRun's recover captures the real stack; implement 'n' as bar-prefill or fix the copy; typed ErrNoActiveSprint fallback in loadStats; picker onCommit callback + single m.picker; one chord helper; invalidate() on the screen interface; rename filter.go locals to rows/matched; Heat.Hot for load errors.

Files: internal/tui/board.go, internal/tui/boardscreen.go, internal/tui/stats.go, internal/tui/app.go, internal/tui/run.go, internal/tui/yank.go, internal/tui/boardscope.go, internal/tui/filter.go, internal/tui/detailhost.go, internal/core/show.go
<!-- /kira:comment -->
