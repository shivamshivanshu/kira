---
id: 01KYKDNPSZEGK032FRCJCVNKE1
number: CORE-50
aliases: []
type: epic
title: "CORE-3 follow-through: schema-aware reads and representation"
state: PLANNED
labels: []
epic: null
blocked_by: []
created: 2026-07-28T09:23:08+05:30
updated: 2026-07-28T09:23:08+05:30
---

Remaining work from CORE-3 (`docs/design/CORE-3-extensible-schema.md`),
split into independently shippable tickets.

**What already works** (phases 1-3, shipped): schemas load from
`.kira/schema/*.json` layered over embedded builtins; typed custom fields
validate and round-trip through create/edit/merge via `--field`; body-section
fields via `placement: body`; ref target-type enforcement on write. A
brand-new item type also already works generically off a `workflows:` config
entry alone -- no schema file, no code change (verified 2026-07-28, see
CORE-49).

**The through-line of what's left**: the *write* path is schema-aware, the
*read* paths are not. A custom field persists correctly in the file but is
invisible to `show`, to `--json`, and to the query grammar, and a schema's
`representation` block is parsed and then ignored. Each child ticket closes
one part of that gap.

Phase 6 ("replace the Item struct with a schema-driven value") is explicitly
out of scope and marked maybe-never in the design doc; the hybrid should be
evaluated as good enough first. It is deliberately not a child of this epic.
