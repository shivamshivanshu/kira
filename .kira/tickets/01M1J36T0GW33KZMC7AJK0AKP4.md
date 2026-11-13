---
id: 01M1J36T0GW33KZMC7AJK0AKP4
number: CORE-67
aliases: []
type: ticket
title: "blame, log and changes silently omit every list-valued field"
state: DONE
resolution: done
priority: P2
labels: []
parent: null
blocked_by: []
created: 2026-09-03T00:18:05+02:00
updated: 2026-11-13T22:03:56+05:30
---
`scalarFieldSet()` (`internal/core/eventparse.go:136`) builds the set of fields
the history walker tracks by excluding the non-scalar ones:

    nonScalar := aliases, labels, blocked_by, links, created, updated

So the entire derived-history subsystem -- `kira blame`, `kira log`, `kira
changes`, and `show`'s `history_tail` -- is blind to `aliases`, `labels`,
`blocked_by` and `links`. It reports nothing for them and says nothing about
the omission.

`kira blame --help` promises "Show, per field, its current value and the commit
that last set it." It does not do that for four fields.

Demonstrated on a scratch repo with one ticket and three edits: `kira diff`
reports the `labels` and `blocked_by` changes, while `kira changes` reports only
the `priority` change and `kira blame` prints no row for either field. `kira
log` shows the commit `kira: T1-1 label add alpha` with no event explaining what
it did -- the user sees that something happened and not what.

This matters more since CORE-59: `blocked_by` now drives the move guard and
cycle detection, so it is a field with real behaviour whose history is
invisible.

Related fragility in the same file, worth fixing together or documenting:

- `eventparse.go:88` -- the fence-counting parser is only correct under
  whole-file `-U` context; a narrower context drops every field change
  silently.
- `eventparse.go:196` -- `deriveEventBatch` disables its fast path if the repo
  contains any merge commit, because per-file and multi-file log simplification
  disagree. `rangeEvents`, which feeds `changes` and the index's incremental
  fill, has no such guard.

Options: teach the walker list fields, or narrow the documented promise so it
stops claiming to cover every field. Deriving field history by text-parsing
`git log -p` is the underlying fragility; a YAGNI audit has separately
questioned whether this subsystem earns its keep at all.
