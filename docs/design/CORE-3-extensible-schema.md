# CORE-3 — Extensible entity schema

Status: design approved (brainstorm 2026-07-23). Phase 1 in progress.

## Problem

Entity structure is hardcoded: `datamodel.Item` is a fixed struct, frontmatter
keys are a fixed list, unknown keys are preserved but untyped (`UnknownKeys`),
and body sections (`## Description`, …) are template convention, not typed.
There is no way to add a typed field or a recognized section per type, or to
define a new entity type, without editing Go.

## Goal

Make entity structure **data, not code**: entity *types* are defined by JSON
schema files composed from a fixed set of typed primitives. The engine
validates, stores, renders, and queries any schema-defined entity generically.
Built-in `ticket`/`epic`/`board` become shipped schemas — the same code path as
user-defined types.

## Non-goals (v1)

- Nested object / sub-schema field types.
- Computed / formula / rollup fields (epic progress stays an engine feature).
- Relocating config-backed instances (boards/sprints stay in `config.yaml`).
- Replacing the `Item` struct wholesale (deferred to a late phase; may never be
  worth it — see Phasing).

## Decisions (from brainstorm)

1. **Unified, phased.** One generic schema engine; built-ins ship as schema
   files; correctness is proven by the existing golden/contract suite passing
   **unchanged** against schema-driven built-ins (characterization testing).
   Corollary: **zero data migration** — existing `.kira` files already match
   their built-in schema.
2. **Strongly typed.** `ref<type>` and named `enum`s validate their targets;
   refs bind to the immutable **ULID**, not the `KIRA-n` display number. Engine
   owns referential integrity + a `doctor` check.
3. **Storage model A.** Schema describes *shape*; storage is unchanged.
   *Item entities* (ticket, epic) are per-file under `.kira/tickets/`; *config
   entities* (board, sprint, enum vocab) stay list entries in `config.yaml`.
   The engine models these as two storage backends behind one schema system.
   A follow-up ticket tracks migrating config entities to file-backed storage
   (the "everything is a file-backed entity" end state); out of scope here.
4. **Schemas are repo-tier**, `.kira/schema/*.json`, versioned with the repo,
   shared by the team, available to the merge-driver — **per-project scope**.
   Built-in schemas are embedded (`go:embed`) as the always-present fallback;
   `kira init` **materializes the shipped defaults** into `.kira/schema/` so a
   new repo has editable starting schemas. User-tier `~/.config` may later hold
   personal *representation* prefs only, never the data contract.

## Type system

### Primitives

| Type | Notes |
|---|---|
| `string` | single-line |
| `markdown` | multi-line block; with `placement: body` **this is a custom section** |
| `int` | |
| `number` | float; optional `unit` |
| `bool` | |
| `date` | calendar (`time.DateOnly`) |
| `datetime` | instant w/ offset (RFC3339) |
| `enum` | value set referenced by name (`priority_enum`); reusable |
| `ref` | reference to another entity; `target: <type>`; strongly typed, ULID-bound |

`state` and `resolution` are **compositions**, not new primitives: `state` is an
`enum` bound to the type's workflow (guarded, transition rules); `resolution` is
an `enum` gated by done-category. `user`/`label`/`sprint`/`board` fields are
`enum`/`ref` whose values come from a config list (`source:` pointer) rather than
an inline set.

### Modifiers / attributes (orthogonal to type)

- `list: true` (+ `unique: true` for set semantics) — one collection modifier
  covers labels, aliases, blocked_by, multi-select.
- `required`, `default`, `immutable`, `guarded` (workflow-only mutation),
  `system` (auto-set: id/created/updated), `min`/`max`, `pattern`.

### Representation (the `name` decoupling)

Per field and per schema, separate from data:
- `label` (display name), `icon`, `render` hint (badge | date | markdown | plain),
  `placement` (`frontmatter` | `body` + `section` title), list-column visibility.

This is what lets the CLI/TUI render an unknown entity generically. The schema
`name` is the type id; `representation.label` is what the user sees.

## Schema file format

```jsonc
// .kira/schema/ticket.json
{
  "name": "ticket",
  "workflow": "ticket",                        // -> config.workflows.ticket
  "identity": { "style": "sequential", "prefix": "KIRA" },
  "fields": [
    { "name": "title",       "type": "string",   "required": true },
    { "name": "state",       "type": "state",    "workflow": "ticket", "guarded": true },
    { "name": "subtype",     "type": "enum",     "enum": "subtype_enum" },
    { "name": "priority",    "type": "enum",     "enum": "priority_enum" },
    { "name": "owner",       "type": "ref",      "target": "person" },
    { "name": "epic",        "type": "ref",      "target": "epic" },
    { "name": "blocked_by",  "type": "ref",      "target": "ticket", "list": true },
    { "name": "labels",      "type": "enum",     "enum": "label_vocab", "list": true, "unique": true },
    { "name": "due",         "type": "date" },
    { "name": "estimate",    "type": "number",   "unit": "points" },
    { "name": "created",     "type": "datetime", "system": true },
    { "name": "updated",     "type": "datetime", "system": true },
    { "name": "description", "type": "markdown", "placement": "body", "section": "Description" },
    { "name": "acceptance",  "type": "markdown", "placement": "body", "section": "Acceptance criteria" }
  ],
  "representation": {
    "label": "Ticket", "icon": "ticket",
    "list_columns": ["number", "state", "subtype", "priority", "title"]
  }
}
```

Named enums resolve, in order: inline `enum` map in a schema file, then
`config.yaml` vocab (`priorities`, `subtypes`, `resolutions`, `labels`,
`people`) so built-ins reuse today's config without duplication.

## Engine architecture

New package `internal/entityschema` (leaf-ish; depends only on `datamodel`):

- `Schema`, `FieldDef`, `FieldType`, `EnumDef` — the parsed model.
- `Loader` — parse JSON from `.kira/schema/` layered over **embedded built-in**
  schemas (`go:embed` ticket/epic/board.json). User files override/extend by
  `name`.
- `Validator` — pure: given `(Schema, fieldValues)` return `[]Violation`
  (type mismatch, missing required, enum non-membership, ref target
  missing/wrong-type). No IO; ref existence checked against a supplied
  `RefResolver` interface so the validator stays pure/testable.

Integration seam with today's code is `datamodel.Fields []FieldDescriptor`: a
later phase builds the descriptor list *from* a schema instead of the hardcoded
slice. Not in Phase 1.

## Verification strategy

Characterization tests are the acceptance criteria: the built-in schemas must
**accept exactly** today's data and reject violations. If the existing suite
plus a conformance test over every `.kira` item stays green, the engine
faithfully reproduces hardcoded behavior.

## Phasing

- **Phase 1 (this delegation) — engine + built-in schemas + conformance, additive & non-breaking.**
  New `internal/entityschema` package; embedded built-in ticket/epic/board
  JSON; pure validator (type/required/enum; ref shallow — existence deferred);
  conformance test loading every existing `.kira/tickets/*.md` (via current
  codec) and asserting zero violations, plus negative unit tests. `kira init`
  materializes the embedded defaults into `.kira/schema/` (additive: new dir,
  no change to existing init output otherwise). **No changes to `Item`, codec,
  core mutations, cli commands other than init wiring, or tui.** De-risks the
  type system; nothing in the hot path.
- **Phase 2** — strong `ref` integrity (ULID-bound) + `doctor` dangling/wrong-type
  check + completion sourced from schema.
- **Phase 3** — typed *extra* fields on ticket/epic: codec round-trips typed
  frontmatter (replacing untyped `UnknownKeys` for schema-declared keys);
  create/edit/validate honor them.
- **Phase 4** — user-defined *new* item entity types: generic create/edit/list/show.
- **Phase 5** — representation-driven rendering (list columns, show layout, icons)
  from schema.
- **Phase 6 (maybe never)** — replace `Item` struct with a schema-driven value.
  Evaluate whether the hybrid is good enough first.

Each phase is its own ticket + spec + PR. Phases 2-6 are follow-ups, not part of
the current delegation.

## Open items

- Enum resolution precedence when a schema inline enum and a config vocab share
  a name — Phase 1 documents "config vocab wins for built-in names"; revisit.
- Whether `board`/`sprint` schemas are validated in Phase 1 (config-backed) or
  deferred — Phase 1 includes their *schema definitions* and validates existing
  config entries, but adds no new storage.
