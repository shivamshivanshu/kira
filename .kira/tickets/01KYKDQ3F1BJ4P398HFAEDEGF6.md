---
id: 01KYKDQ3F1BJ4P398HFAEDEGF6
number: CORE-53
aliases: []
type: ticket
title: "Fold the dangling-ref scan into the schema-driven ref check"
state: DONE
resolution: done
priority: P2
labels: []
epic: 01KYKDNPSZEGK032FRCJCVNKE1
blocked_by: []
created: 2026-07-28T09:23:54+05:30
updated: 2026-07-28T14:28:05+05:30
---

`doctor`'s dangling-reference scan only walks the three built-in
relationships (`epic`, `blocked_by`, `links`), so a schema-declared custom
`ref` field is checked for target *type* but not for target *existence*.

Practical effect: set a custom ref through `--field` and it is validated. But
if a bad reference reaches the file another way -- a hand edit, a bad merge,
a renumber gone wrong -- and points at nothing at all, `doctor` stays silent.
The wrong-*type* case is already caught (CORE-44 wired `ProjectItem` into
`doctor.RefTypeMismatches`, so it sees ref extras); it is specifically the
"points at nothing" case that the dangling scan misses.

**Where it lives**: the hardcoded `refFindings` pass in `internal/doctor`.
`RefTypeMismatches` in `internal/doctor/graph.go` already iterates
`schema.Fields` and filters on `f.TargetsEntity(schemas)` -- that is the
shape the dangling pass should converge on, so one traversal yields both
findings per declared ref field instead of two passes with different
knowledge of what a ref is.

**Watch for**: `Source != ""` fields (owner/reporter/sprint) are config-vocab
refs, not entity refs -- they must not be dangling-checked against the item
store. `entityschema.FieldDef.TargetsEntity` already draws that line; reuse
it rather than re-deriving the condition.

**Origin**: called out in the design doc's Phase 4 list, and re-confirmed as
the real remaining Phase 4 gap in CORE-49.
