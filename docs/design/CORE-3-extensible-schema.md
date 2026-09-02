# CORE-3 — Extensible entity schema

Status: design approved (brainstorm 2026-07-23). Phases 1-3 done. Phase 4's
headline goal — user-defined new item types — turned out to already work
generically: a `workflows:` entry in `.kira/config.yaml` is enough for
`create`/`edit`/`move`/`list`/`show` and doctor's state checks to handle a new
type, with no schema file and no code change (verified 2026-07-28). What
remains of Phase 4 is narrower than this line used to imply; see the phase list
below.

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
- Per-type identity (per-type ID sequences / display-number prefixes). Early
  Phase 1 schemas declared `identity: {style, prefix}` that nothing generated
  IDs from — every type shares the one project sequence, and display numbers
  come from `project.key` / `boards[].key`. The prefix was not merely inert but
  wrong: a repo initialized with `--key PROJ` still got a schema file asserting
  `KIRA`. Both it and the `sequential` style are gone (KIRA-141, restored in
  error during the origin merge, removed again).

  What survives is `identity.style: config` on `board.json`, which is read by
  `Schema.ConfigBacked()` and marks a type whose instances live in
  `config.yaml` rather than as files. That is a storage backend rather than an
  identity scheme, and the key is misnamed for what it now does.

  The loader tolerates the old keys in schema files materialized by older
  versions. Per-type identity remains open as a future feature.

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
5. **Free transitions** (adopted; landed as KIRA-140). `state` is a plain
   enum; any value → any value is allowed. The transition **graph**
   (`transitions`, `enforce_transitions`, allowed edges) is gone — membership
   in the workflow's declared states is the only check. The `blockers_closed`
   require-guard became a state-independent close check (refuse, or warn under
   `--force`, when entering a `done`-category state with open blockers,
   regardless of path) rather than a per-edge require, and `resolution` is now
   auto-set from the target state's own `resolution:` field on entering a done
   state and cleared on leaving one. This was a deliberate *behavior* change
   and it updated the existing transition-enforcement tests — the
   characterization guarantee covers data *shape*, not transition *policy*.

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

`state` is a **plain `enum`** whose values each carry a `category`
(todo/doing/done). **Transitions are unconstrained** — any value → any value
(see Decision 5). Category is retained (drives board columns, done-detection,
resolution gating, epic progress); only the edges between states go away.
`resolution` is an `enum` gated by done-category. `user`/`label`/`sprint`/`board`
fields are `enum`/`ref` whose values come from a config list (`source:` pointer)
rather than an inline set.

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

### A type is two required halves

A schema file alone does not create an entity type. A type is defined by its
**schema** (`.kira/schema/<type>.json` — the shape) *and* a **`workflows:`
entry** in `.kira/config.yaml` (the state set). `kira
create` refuses a type with no workflow (`no workflow configured for type
"foo"`, hinting at the missing half), and existing items of a workflow-less
type load read-only with an orphan warning. States are deliberately not
inferred from the schema's `state` field: as long as workflows live in config,
inference would give the state set two sources of truth. That duplication
already exists between a schema's declared `state` values and `cfg.Workflows`,
and `doctor`'s `state-drift` check (`internal/doctor/statedrift.go`) is what
keeps it from rotting silently — inference would widen the seam rather than
close it. CORE-39 (config entities become file-backed) is where it gets
resolved for good.

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
- **Phase 2 (done) — schema-driven ref-type integrity.** `doctor` now checks a
  ref field's *target type* against the schema (`RefTypeMismatches`, class
  `ref-type`), generalizing the hardcoded epic-parent check (`NonEpicParents`,
  removed) to every item-target ref field — catching e.g. `blocked_by` pointing
  at the wrong kind. Dangling/self-ref/cycle checks already existed and are
  unchanged. Read-only; drives off the embedded built-in schemas (user
  `.kira/schema` overrides for ref targets deferred to Phase 4, when custom
  item types make doctor load user schemas anyway). Completion-sourced-from-schema
  moved to Phase 4/5 (see the CLI section); write-path `Validate` wiring is Phase 3.
- **Phase 3 (done)** — typed *extra* fields on ticket/epic. Landed as 3a + 3b (3b-i/ii/iii):
  - **3a** — lossless, schema-agnostic round-trip of extra frontmatter.
    `UnknownKeys` (names only, values dropped) replaced by `Item.ExtraScalars`/
    `ExtraLists`; codec parses+serializes them faithfully (scalar rendering
    preserves null/int/bool/quoting byte-identically); `merge.mergeExtras`
    keeps the two maps disjoint; `guardWritable` no longer hard-blocks on
    extras (still blocks on `UnknownLinkTypes`, which stay lossy — CORE-41).
  - **3b-i** — REPLACE `validateItem`'s field-level checks (required/type/enum/
    ref) with `entityschema.Validate` + a `RefResolver` adapter and severity
    policy-mapping; state/sprint/rank/graph/resolution stay config-driven.
  - **3b-ii** — `create`/`edit --field` editing of declared frontmatter extras
    (typed-parsed, round-trip-safe); schema-aware `doctor` lint; a validationCtx
    that builds byID once, caches schemas, resolves enums once per batch.
  - **3b-iii** — writable `placement:body` fields via `--field` (edit a section);
    `datamodel.BodySections`/`UpsertBodySection` single-source the `## ` rule.
  - **Wire shape** — `--json` carries declared extras under a `fields` object
    with their declared type preserved: an `int` comes back as a number, a
    `list` as an array. Storage still keeps canonical YAML text (what leaves
    `points: 42` un-requoted on rewrite); typing happens on the way out.
    `schema/kira.json` stays static by typing `fields` as an open object rather
    than one property per declared field. This supersedes the earlier
    strings-on-the-wire decision (KIRA-145), which is withdrawn.
  - Follow-ups: CORE-41 (unknown link-type round-trip), CORE-42 (extras in merge
    arbitration + ref-extra ULID normalization), CORE-43 (blocked_by target-type
    on write).
- **Phase 4** — what actually remains for user-defined types and custom fields.
  Generic create/edit/move/list/show for a new type already works off a
  `workflows:` config entry alone (no schema file, no code change), so the open
  work is the parts that are still blind to schema-declared fields:
  - *(done, CORE-53)* The dangling/self-reference scan drives off the item's
    schema rather than a hardcoded `epic`/`blocked_by`/`links` list, so a custom
    ref field gets dangling, self-reference and wrong-type checks together.
    CORE-56 extends the target-type check to refs aimed at a config-only type.
  - *(done, CORE-51 + CORE-52 + KIRA-143)* `show` prints declared frontmatter
    fields after the built-in header and carries them under `fields` in
    `--json`; the `query` grammar resolves a declared name and compares by its
    declared type, with `list --field key=value` as the equality shorthand.
    Body-placed fields (`placement: body`) remain outside `fields` and
    unfilterable — their text is already in the body `show` prints.
  - `.kira/schema` overrides for ref targets (deferred here from Phase 1).
- **Phase 5 (part done)** — representation-driven rendering from schema. List
  columns landed: `representation.list_columns` names the table's columns when
  `ui.list.columns` is not pinned, and a schema-declared field can be one of
  them (CORE-54). Show layout and icons are still hardcoded.
- **Phase 6 (maybe never)** — replace `Item` struct with a schema-driven value.
  Evaluate whether the hybrid is good enough first.

Each phase is its own ticket + spec + PR. Phases 2-6 are follow-ups, not part of
the current delegation.

## CLI field filtering & flags (Phase 4/5)

Filtering by a schema field — built-in or custom — resolves to **one engine**:
the `query` grammar (`state = TODO AND severity = HIGH`), made schema-aware
(field names + enum values validated/completed against the schema). Everything
else desugars into it; no parallel filter path.

- **Generic `--field k=v`** on `list` (repeatable, ANDed, equality-only)
  desugars to a query equality predicate. Works for any schema field or entity.
  Richer operators (`<`, `IN`, `NOT`) stay in the query expression. Unknown
  field → `errx` with a did-you-mean from schema field names; bad enum value →
  error listing the schema's allowed values (same vocab as `entityschema`).
  `--field <TAB>` completes field names; `--field state=<TAB>` completes values.
- **Native flags** (`--state`, `--priority`, …) are **kept as ergonomic
  aliases** that desugar to the same predicate — `--field state=TODO` ≡
  `--state TODO` ≡ query `state = TODO`. Uniform *mechanism*, not uniform
  *surface*.

Which fields get a native flag: a **curated, compile-time set** of the built-in
commonly-filtered fields (state/priority/owner/label/sprint/epic/subtype/due),
collision-checked against reserved/global flags. Custom fields use `--field`
only. Native flags are *not* auto-derived from the schema because cobra
registers flags before argv is parsed, but the schema lives in the repo (opened
only after `-C`/cwd resolves) — a chicken-and-egg — and user field names can
collide with reserved flags. The flag's *values* still come from the schema
(completion/validation); only its *existence* is compile-time.

Escape hatch (later, opt-in): a field may declare `representation.flag` to
request a native flag, dynamically registered after the repo resolves, guarded
against reserved-name collisions. Not v1.

## Open items

- Enum resolution precedence when a schema inline enum and a config vocab share
  a name — Phase 1 documents "config vocab wins for built-in names"; revisit.
- Whether `board`/`sprint` schemas are validated in Phase 1 (config-backed) or
  deferred — Phase 1 includes their *schema definitions* and validates existing
  config entries, but adds no new storage.

## Extras in item history (`log`, `blame`, `changes`)

The frontmatter diff walker reports schema-declared **scalar** extras next to
the builtin scalar fields, so `kira log`/`blame`/`changes` explain an
`edit severity` commit rather than showing it as an empty change. Two limits
are deliberate:

- **List extras get no events**, matching the builtin list fields (`labels`,
  `blocked_by`, `aliases`, `links`), which the walker has always skipped.
- **Declared extras are unioned across schemas**, not scoped to the item's
  type: the walker keys off a file path, and a file's history spans commits
  whose type or schema may differ from the working tree's.

Derivation now depends on the schema, so the cached events carry the index
schema version and are dropped when it changes. Declaring a *new* extra does
not retroactively re-derive events for items whose file has not changed since
the last derivation; `kira index --full` does.

Time-travel reads (`--at`) resolve extras against the schemas of the tree-ish
they read, loaded in the same batched `cat-file` pass as its items and config —
never the working tree's, which may declare a field the snapshot did not (or
type it differently).
