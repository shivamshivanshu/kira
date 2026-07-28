---
id: 01KYKDV966X444A0GAYNC4ESTV
number: CORE-57
aliases: []
type: ticket
title: "create --from-file silently discards every draft flag"
state: TODO
labels: []
epic: null
blocked_by: []
created: 2026-07-28T09:26:11+05:30
updated: 2026-07-28T09:26:11+05:30
---

`kira create --from-file` returns the parsed file as the draft and throws
away the flag-applied base, so `--parent`, `--priority`, `--owner`,
`--reporter`, `--label`, `--subtype`, `--rank`, `--due`, `--estimate` and
`--sprint` are all accepted on the command line and then silently ignored.
No warning, no error, exit 0.

`--board` is unaffected (board resolution runs separately via
`resolveBoardKey`), which makes the failure more confusing: some flags
appear to work.

Repro:

    kira create ticket --board CORE --parent CORE-50 --priority P1 --from-file - <<'IN'
    ---
    title: "child"
    ---
    body
    IN
    kira show <new> --json    # "epic": null, "priority": null

Hit for real while creating the CORE-50 epic's six child tickets: all six
were created with `--parent` and `--priority` and all six came out
unparented and unprioritised, needing a follow-up `link`/`edit` per ticket.

**Cause**: `draftForCreate` (`internal/core/create.go`) builds `base` via
`applyFlags`, then in the `opts.FromFile != ""` branch returns the parsed
draft and drops `base` entirely. The `NoEdit` branch returns `base`, and the
editor branch seeds the editor with it -- only the from-file branch discards.

**Fix direction**: decide precedence explicitly rather than by accident. The
file is the authoritative content, so the natural rule is "file wins for any
field it sets; flags fill in fields the file leaves empty" -- i.e. overlay
the parsed draft onto `base` instead of replacing it. The alternative,
rejecting draft flags alongside `--from-file` with a clear error, is also
defensible and is a smaller change; either is better than silent drop.

Note `--field` is not affected: it is applied after, via `applyFieldEdits`
on the assembled item.
