---
id: 01M1H346B8JE2EABZY5BRHZHWZ
number: CLI-12
aliases: []
type: ticket
title: "show padded its label column by byte length, misaligning non-ASCII labels"
state: DONE
resolution: done
priority: P3
labels: []
epic: null
blocked_by: []
created: 2026-09-02T14:57:25+02:00
updated: 2026-09-02T15:10:55+02:00
---
`renderShow` sized the label column with `len(label)` (bytes) and padded with
`fmt.Fprintf("%-*s")`, which pads by rune count. Neither is the column count a
terminal uses, so the three measures only coincide for ASCII.

Labels are not always ASCII: they include schema custom-field names and
configured link-type names, both user-authored.

Verified by running the binary -- a schema field named 类 on a scratch repo,
`kira show` with `cat -A`:

    id:·······01M1H1TK...      <- value at column 10
    category:·todo             <- value at column 10
    类:········x               <- value at column 11

Every other row's value starts at column 10; the CJK-labelled row starts at
11. Now measured and padded with `ansi.StringWidth`, matching what
`internal/tui` already does. No golden covered a non-ASCII label, so nothing
needed regenerating.
