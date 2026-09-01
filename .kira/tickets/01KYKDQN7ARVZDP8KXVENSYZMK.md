---
id: 01KYKDQN7ARVZDP8KXVENSYZMK
number: CORE-54
aliases: []
type: ticket
title: "Representation-driven list columns from schema"
state: DONE
resolution: done
priority: P2
labels: []
epic: 01KYKDNPSZEGK032FRCJCVNKE1
blocked_by: []
created: 2026-07-28T09:24:12+05:30
updated: 2026-07-28T14:28:05+05:30
---

A schema's `representation.list_columns` is parsed, accepted, and then
ignored: `kira list` renders a fixed column set regardless of what any
schema declares.

Today `datamodel.DefaultListColumns` is the hardcoded
`[number, state, type, priority, title]`, gated by a `ListColumns`
allowlist, with a single shared column slice for every row. The builtin
ticket schema meanwhile declares
`["number", "state", "subtype", "priority", "title"]` -- so wiring this up
is a deliberate output change, not a byte-identical refactor.

**Known obstacles** (from recon):
- `datamodel.ListItem` has no `Subtype` field at all, so the ticket schema's
  declared columns are not even projectable today. The read-projection DTOs
  need widening, or a generic by-name accessor.
- `datamodel.Field(key)`/`FieldDescriptor.Get` exist but operate on `*Item`,
  not on the read DTOs (`ListItem`/`TreeNode`/`BoardColumn`), so there is no
  single "resolve field by name for display" path yet. That accessor is the
  real shared prerequisite -- CORE-55 needs it too.
- `list` mixes types in one table. Per-type column sets mean deciding what a
  mixed ticket+epic listing renders: union of columns, the default board's
  type, or a per-type section. This is the main design fork.
- `board list` is non-tabular hand-formatting and will need its own decision.

**Must not break**: the `ui.list.columns` config override
(`tests/e2e/testdata/script/ui-list-columns.txtar`). Suggested precedence --
schema `list_columns` becomes the per-type default, `ui.list.columns`
overrides it -- but confirm before building.

**Test fallout**: `tests/contract/testdata/golden/plain/list.plain` and
several TUI goldens move. That is expected; they should be re-baselined
deliberately, not force-updated.
