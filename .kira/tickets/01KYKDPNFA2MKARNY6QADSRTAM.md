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
updated: 2026-07-28T09:36:51+05:30
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

<!-- kira:comment id=01KYKEET0RG1YMG9XJXG2RVDH9 author=Shivam-Shivanshu ts=2026-07-28T09:36:51+05:30 -->
Investigation 2026-07-28, before any code.

The crux is that field validity and field *type* are resolved at parse time, but a custom field's type is only knowable from the schema at compile time. Three sites in internal/query consult the static `fields` map:

- parser.go:306 -- decides whether a word starts a field predicate at all
- parser.go:325 -- `startsFieldPredicate()` then raises unknownFieldErr, which is the error users hit today
- parser.go:208 -- ORDER BY validation
- eval.go:61 -- Match(), the flag-filter entry point (--label etc), built-ins only

`Parse(input string)` takes no options, so schemas cannot reach it; `Compile(input, Options)` parses first and then compiles. parsePredicate also runs typeCheckValue() at parse time, switching on the field name to decide whether the value is a date or a number -- which cannot work for a field whose type lives in the schema.

Suggested shape: make the parser purely syntactic for unknown words (a word followed by a comparison operator parses into a predicate node with the value left as a raw token), and move both the unknown-field error and the typed conversion into the compiler, where Options can carry the loaded schemas. The existing unknownFieldErr moves to compilePred's fallthrough, which already exists as the default branch.

Semantics to settle when picking this up (also in the ticket body):
- a query spans types, so an unknown field should error only when *no* loaded schema declares it; an item whose type does not declare it should simply not match rather than error
- equality and inequality for all custom scalar types is the v1 core; int/number/date/datetime can reuse estimatePred/datePred for ordered compare; custom enums are probably equality-only in v1 since declared `values` order is not the same concept as the configured priority order
- ORDER BY on a custom field: parser.go:208 rejects it today. Deciding to leave that for later is fine, but do it deliberately.
- the 'available:' list in the error should include schema fields once they are valid, or the message becomes misleading

Not started; moved back to TODO.
<!-- /kira:comment -->
