---
id: 01M1H3VYABAQF7C83FMYMZJQ9X
number: CORE-61
aliases: []
type: ticket
title: "Route hand-rolled dedup through setx, and remove duplicated logic across layers"
state: TODO
priority: P3
labels: []
epic: null
blocked_by: []
created: 2026-09-02T15:10:23+02:00
updated: 2026-09-02T15:10:23+02:00
---
Four independent audits (low layers, mid layers, cli/tui, and cross-layer)
converged on the same set, which is what gave confidence it was real rather
than one agent's taste.

`internal/setx` is this repo's home for dedup/set helpers, but nine sites
hand-rolled a `map[string]bool` seen-map -- several in files that already
imported and used setx elsewhere, so this was an inconsistency, not merely a
missed reuse: `index/trailers.go` (x2), `datamodel/graph.go`,
`datamodel/item.go`, `entityschema/{validator,loader}.go`,
`config/validate.go` (x2).

`setx.AppendUnique` replaces the "append if not already present" idiom at six
sites (`doctor/collision.go`, `core/label.go`, `core/reconcile.go`,
`id/resolve.go` x3), deleting doctor's local variadic copy -- both its callers
passed exactly one value.

`setx.Deduper` gained `Has`, because one site in `trailers.go` needs
membership WITHOUT marking. Converting it to `Add` would have mutated the set;
the first attempt substituted a linear `slices.Contains` over the accumulated
slice, which is semantically right but turns an O(1) lookup into O(n) inside a
loop over every bare ref in a commit body. `Has` keeps it O(1) and completes
the abstraction rather than working around its gap.

Also removed:
- `core/show.go` -- five lines duplicated verbatim between `Show`'s `at`
  branch and `detail`, now one helper; each caller keeps the different fields
  it adds afterwards.
- `internal/cli` -- one closure triplicated byte-for-byte in edit/assign/label,
  now a package-level function beside `emitMutationWarnings`.
- `codec/parse.go` -- the frontmatter fence grammar was implemented twice
  (`splitDocument` and `cutAfterFrontmatter`), with the same four branches and
  the same CRLF handling. Derived the second from the first. Verified byte-exact
  by differential test against the old implementation over a 16-document
  corpus including CRLF, a missing trailing newline, a bare `---` at EOF, `---`
  inside the body, and `---` inside a YAML block scalar -- `cutAfterFrontmatter`
  feeds comment appends, so a silent difference would corrupt users' files.

Deliberately NOT changed, each judged and rejected: `core/link.go`'s
`toggleRef` (different early-return shape), `merge/set.go`'s `keyUnion` (a
different operation), `diff.go`/`changes.go`'s similar set-partition loops
(different payloads, two call sites -- a callback abstraction would hide
control flow), `validate.go`'s two ref-resolution closures (they look like
copy-paste but one deliberately swallows an error the other reports), and
`core`'s Batch/mutate orchestration, which is already the shared template.
