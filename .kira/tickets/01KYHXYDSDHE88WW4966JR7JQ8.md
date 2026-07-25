---
id: 01KYHXYDSDHE88WW4966JR7JQ8
number: CORE-48
aliases: []
type: ticket
title: "show --json blocks field is always empty (never derived)"
state: TODO
labels: []
epic: null
blocked_by: []
created: 2026-07-27T19:29:02+05:30
updated: 2026-07-27T19:29:02+05:30
---

`ShowResult.Blocks` (the reverse `blocked_by` edge — "which items does this
one block") is part of the published `--json` shape but is never actually
populated. `internal/core/show.go:98` hardcodes:

    Blocks:        []string{},

and nothing else in the codebase assigns `.Blocks` (confirmed: no other
assignment site exists). The index-enrichment step just below it derives
`LinkedCommits`/`HistoryTail` from the index but never touches `Blocks`, and
a `kira index --full` does not change the outcome.

Repro:

    kira link <blocked> --blocked-by <blocker>
    kira show <blocked> --json | jq .blocked_by   # correctly lists <blocker>
    kira index --full
    kira show <blocker> --json | jq .blocks       # [] -- should list <blocked>

`tests/e2e/testdata/script/link.txtar` asserts `"blocks": []` on the blocker
and comments it as "derived (empty pre-index)" -- the test currently pins the
buggy value as expected, and nothing ever derives it post-index either.

Anyone scripting against the documented `--json` shape gets a silently empty
array for a field that looks meaningful. Worth deciding whether to actually
implement `blocks` (derive it from other items' `blocked_by` at show/index
time) or drop it from the schema if it's not planned.

Found while writing docs/user_manual/08-recipes-and-examples.md (kira
user-manual project), verifying a "what blocks this ticket" recipe.
