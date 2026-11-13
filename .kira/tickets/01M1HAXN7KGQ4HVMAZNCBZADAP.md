---
id: 01M1HAXN7KGQ4HVMAZNCBZADAP
number: CLI-13
aliases: []
type: ticket
title: "tabwriter sizes columns by rune count, misaligning any non-ASCII cell"
state: DONE
resolution: done
priority: P3
labels: []
parent: null
blocked_by: []
created: 2026-09-02T17:13:40+02:00
updated: 2026-11-13T21:50:05+05:30
---
`text/tabwriter` sizes columns by rune count, not terminal display width -- its
own documentation says it "assumes all Unicode code points have the same
width". Every column to the right of a wide cell therefore shifts.

Verified by running the binary, not by reading. With
`ui.list.columns: [number, title, state]` so TITLE is not the last column:

    NUMBER  TITLE  STATE
    TW-1    ab     TODO        <- STATE at display column 15
    TW-2    标题标题   TODO     <- STATE at display column 19
    TW-3    cd     TODO

`标题标题` is 4 runes but 8 display cells, so tabwriter pads it as if it were 4
wide and the row runs 4 cells long.

Same class as CLI-12, but CLI-12's fix (`ansi.StringWidth` in `renderShow`)
does not reach these: they go through stdlib tabwriter rather than hand-rolled
padding.

Call sites, all via `newTabWriter` (`internal/cli/output.go:36`):
`cli/list.go:110` and its tree-grouping variant, `cli/tree.go:49,75`,
`cli/blame.go:27`, `cli/config.go:125`, `cli/label.go:64`.

Not a one-liner: it needs either a width-aware replacement for tabwriter's
column engine, or pre-padding each cell so rune count matches display width.

No existing fixture contains non-ASCII, so no golden changes today -- which is
itself the gap. A fix must add a CJK or emoji fixture, or the bug simply
returns.
