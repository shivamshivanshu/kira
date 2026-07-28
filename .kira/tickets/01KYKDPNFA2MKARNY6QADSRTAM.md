---
id: 01KYKDPNFA2MKARNY6QADSRTAM
number: CORE-52
aliases: []
type: ticket
title: "Make the query grammar schema-aware so custom fields are filterable"
state: TODO
priority: P1
labels: []
epic: 01KYKDNPSZEGK032FRCJCVNKE1
blocked_by: []
created: 2026-07-28T09:23:40+05:30
updated: 2026-07-28T09:25:47+05:30
---

`kira list --query` rejects any schema-declared field outright, so a custom
field cannot be filtered or reported on from the CLI at all.

Verified 2026-07-28 (scratch repo with a `severity` enum field declared in
`.kira/schema/ticket.json` and set on an item):

    $ kira list --query 'severity = high'
    kira: query: unknown field "severity" (available: activity, blocked,
      blocked_by, board, category, created, due, epic, estimate, label, links,
      owner, priority, rank, reporter, resolution, sprint, state, subtype,
      type, updated) at position 0

**Where it lives**: `internal/query` holds a fixed field inventory used for
both parse-time validation (the "available:" list above) and evaluation. It
needs the loaded schemas to extend that inventory per item type, and an
evaluator path that reads the value off the item's extras rather than a
built-in struct field.

**Design points to settle**:
- The field set is per *type*, but a `list` query spans types. Is an unknown
  field an error only when no loaded schema declares it? What does
  `severity = high` mean for an `epic` that has no `severity` -- no match, or
  an error? (Suggest: no match, matching how absent built-ins behave.)
- Typed comparison: `priority > P2` works off the configured ordered vocab.
  A custom enum with declared `values` has an order too -- reuse it, or
  restrict custom enums to equality only in v1?
- The "available:" hint should list schema fields once they are valid, so the
  error stays useful.
- `date`/`datetime`/`int`/`number` custom fields need the same comparison
  semantics as their built-in counterparts, not string compare.

**Related**: CORE-51 (same underlying "read paths are schema-blind" gap; a
shared projection helper may serve both).
