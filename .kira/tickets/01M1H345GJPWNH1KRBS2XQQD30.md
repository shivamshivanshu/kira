---
id: 01M1H345GJPWNH1KRBS2XQQD30
number: CORE-60
aliases: []
type: ticket
title: "Schema representation text was unvalidated, so one bad label corrupted whole tables"
state: TODO
priority: P2
labels: []
epic: null
blocked_by: []
created: 2026-09-02T14:57:24+02:00
updated: 2026-09-02T14:57:24+02:00
---
`checkSchemaShape` validated name, fields, placement, enums, states and refs,
and never touched `representation` at all. Every renderer measures that text
in display cells and then trusts the measurement for layout, so text that
measures wrongly breaks layout silently and undiagnosably -- nothing in the
output names the schema.

Verified by running the binary against a scratch repo:

`representation.label: "tic\nket"` in `.kira/schema/ticket.json`, then
`kira list`:

    NUMBER  STATE    TYPE  PRIORITY  TITLE
    TST-4   TODO     tic
    ket     P2       probe from-file
    TST-1   PLANNED  epic    -  parent epic

Not just the offending row: every *subsequent* row's columns desynchronise
against the header, because `text/tabwriter` counts a newline as end-of-row.

The same hole in `representation.icon` breaks the tui instead: `lipgloss.Width`
reports the widest line, so "a\nb" measures one cell while occupying two, and
the tree pane wraps rather than clips -- every row doubles and the status bar
is pushed off screen (there is no MaxHeight on the path; `splitPane` pads to
the tallest block).

Fixed in `checkSchemaShape`, which `parseSchema` calls for both the embedded
builtins and user overrides, so the shipped schemas must satisfy their own
rule. Rejected: control characters (C0/DEL/C1, catching newline, tab and ESC),
zero-width format characters (U+200B, U+FEFF and critically U+202E, which
measures nothing and reorders the rest of the line), and a leading combining
mark, which composes onto whatever the renderer emitted before it.

Deliberately NOT a hard error: an over-wide icon. That is a doctor warning
instead, so a schema predating literal glyphs (`"icon": "ticket"`) is reported
loudly rather than refusing to load. Threshold is 3 cells -- the ascii glyphs
this project shipped before CORE-55, and the four live category glyphs
(`[ ]`, `[~]`, `[x]`, `[-]`), are all exactly 3.

Residual, accepted and documented rather than fixed: flag and keycap emoji
measure 2 and 1 cells to lipgloss but are drawn differently by some terminals.
Closing that needs a one-grapheme-cluster rule, which would ban `[E]`.
