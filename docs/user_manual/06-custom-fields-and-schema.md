# Custom fields and the entity schema

## Why this exists

kira ships with two item types — `ticket` and `epic` — whose fields cover what
most teams need: title, state, priority, owner, labels, epic parent, due date,
estimate, and so on. But every team tracks something the built-in set does not
have: a `severity` grading on incidents, a `reviewer` pointing at another
ticket, a "Rollback plan" section that should live in every ticket's body, a
`points` counter that isn't the built-in estimate. Rather than fork kira or
smuggle those values into the description text, you declare them in a JSON
schema file under `.kira/schema/`. kira then stores, type-checks, and
vocabulary-checks them like any built-in field, and exposes them on
`kira create --field` and `kira edit --field`. The schema files are ordinary
tracked files, so the shape of your tickets is versioned and reviewed
alongside the code, exactly like the tickets themselves.

## The schema file format

A schema file is a single JSON object describing one entity type:

```
{
  "name":           string,          // required — the type this schema defines
  "workflow":       string,          // optional — the config workflow key for this type
  "identity":       { "style": string, "prefix": string },
  "fields":         [ { ...one field object, shape below... }, ... ],
  "enums":          [ { "name": string, "values": [string, ...] }, ... ],
  "representation": { "label": string, "icon": string, "list_columns": [string, ...] }
}
```

`name` and `fields` carry the weight, and `representation.list_columns`
chooses what `kira list` renders. `identity`, and `representation`'s `label`
and `icon`, are parsed and accepted but nothing reads them yet — see
[Current limits](#current-limits).

Here is a real, working schema: the built-in `ticket` schema with a `severity`
enum field and a `reviewer` ref field added at the end. This is the exact file
used for every worked example below.

```json
{
  "name": "ticket",
  "workflow": "ticket",
  "identity": { "style": "sequential", "prefix": "KIRA" },
  "fields": [
    { "name": "title", "type": "string", "required": true },
    { "name": "state", "type": "state", "required": true, "system": true, "guarded": true,
      "states": [
        { "key": "TODO", "category": "todo" },
        { "key": "IN_PROGRESS", "category": "doing" },
        { "key": "REVIEW", "category": "doing" },
        { "key": "DONE", "category": "done" },
        { "key": "WONT_DO", "category": "done" }
      ] },
    { "name": "subtype", "type": "enum", "enum": "subtype", "source": "subtypes" },
    { "name": "priority", "type": "enum", "enum": "priority", "source": "priorities" },
    { "name": "resolution", "type": "resolution", "enum": "resolution", "source": "resolutions" },
    { "name": "rank", "type": "string" },
    { "name": "owner", "type": "ref", "target": "person", "source": "people" },
    { "name": "reporter", "type": "ref", "target": "person", "source": "people" },
    { "name": "labels", "type": "enum", "enum": "label", "source": "labels", "list": true, "unique": true },
    { "name": "epic", "type": "ref", "target": "epic" },
    { "name": "blocked_by", "type": "ref", "target": "ticket", "list": true, "unique": true },
    { "name": "sprint", "type": "ref", "target": "sprint", "source": "sprints" },
    { "name": "due", "type": "date" },
    { "name": "estimate", "type": "number" },
    { "name": "aliases", "type": "string", "list": true, "unique": true, "required": true, "system": true },
    { "name": "created", "type": "datetime", "required": true, "system": true, "immutable": true },
    { "name": "updated", "type": "datetime", "required": true, "system": true },
    { "name": "description", "type": "markdown", "placement": "body", "section": "Description" },
    { "name": "acceptance", "type": "markdown", "placement": "body", "section": "Acceptance criteria" },

    { "name": "severity", "type": "enum", "enum": "severity" },
    { "name": "reviewer", "type": "ref", "target": "ticket" },
    { "name": "notes", "type": "markdown", "placement": "body", "section": "Notes" }
  ],
  "enums": [
    { "name": "severity", "values": ["low", "medium", "high"] }
  ],
  "representation": {
    "label": "Ticket",
    "icon": "ticket",
    "list_columns": ["number", "state", "subtype", "priority", "title"]
  }
}
```

Two things to notice before the reference table:

- The three added fields sit in the same `fields` array as the built-ins. There
  is no separate "custom fields" list — the built-in fields are declared in
  exactly the same syntax you use for yours.
- `enums` is a sibling of `fields`, not nested inside a field. A field says
  *which* vocabulary it uses (`"enum": "severity"`); the `enums` array supplies
  the values.

### Each field's shape

```
{
  "name": string, "type": string,
  "list": bool, "unique": bool, "required": bool,
  "immutable": bool, "guarded": bool, "system": bool,
  "enum": string, "states": [{ "key": string, "category": string }],
  "target": string, "source": string,
  "placement": "frontmatter" | "body", "section": string
}
```

`name` and `type` are mandatory; everything else defaults to absent/`false`.
Two fields in one schema may not share a `name`.

#### `type`

| Type | Accepts | Notes |
|---|---|---|
| `string` | any text | The plain workhorse. |
| `markdown` | any text | Same validation as `string`; the distinction is intent, and it is the natural type for `placement: body` fields. |
| `int` | a whole number | `--field points=eight` is rejected at parse time. Stored unquoted in YAML. |
| `number` | any number | Integers and decimals both. `estimate` is the built-in example. |
| `bool` | `true`/`false` (also `1`/`0`, `t`/`f`, `T`/`F`, and the all-caps/capitalized spellings) | Stored unquoted. `--field signoff=yep` is rejected. |
| `date` | a calendar date, `YYYY-MM-DD` | `due` is the built-in example. |
| `datetime` | a full RFC 3339 timestamp | `created`/`updated` are the built-in examples. |
| `enum` | text drawn from a named vocabulary | Requires `enum` naming the vocabulary. |
| `ref` | an item reference (or a config-vocab key) | Requires `target`. See `target`/`source` below. |
| `state` | one of the declared `states` keys | Requires `states`. Only meaningful on the type's own state field. |
| `resolution` | text drawn from a named vocabulary | Behaves like `enum`; requires `enum`. Distinct type so kira can tell "this is the resolution field" apart from an ordinary enum. |

#### Attributes

| Attribute | Applies to | Meaning |
|---|---|---|
| `list` | string-valued types only (`string`, `markdown`, `enum`, `state`, `resolution`, `ref`, `date`, `datetime`) | The field holds multiple values. Written as a comma-separated `--field` value (`--field components=api,web`) and stored as a YAML flow sequence. Declaring `list` on `int`, `number`, or `bool` is rejected when the schema file loads. |
| `unique` | list fields | Duplicate values are **rejected**, not silently removed. See the note below. |
| `required` | any | The item is invalid without a value. Enforced on **every** write, not only at create. |
| `immutable` | any | Not settable through `--field`, ever — not even at create. |
| `guarded` | any | Not settable through `--field`; the value is meant to change through a dedicated command instead. |
| `system` | any | kira owns this value. Not settable through `--field`. `created`, `updated`, and `aliases` are the built-in examples. |
| `enum` | `enum`, `resolution` | Names the vocabulary to check membership against. |
| `states` | `state` | Inline `[{key, category}]` list. `category` must be one of `todo`, `doing`, `done`. |
| `target` | `ref` | Which entity type the reference points at (`ticket`, `epic`, …), or, for a config-backed ref, which config vocabulary. |
| `source` | `ref` | Names the config vocabulary that backs this reference. Its presence switches validation from "an item of that type must exist" to "this key must exist in that config vocabulary". |
| `placement` | any | `frontmatter` (the default) or `body`. |
| `section` | `placement: body` | The Markdown `## ` heading the value lives under. Required whenever `placement` is `body`; the schema file fails to load without it. |

**On `unique`.** For a schema-declared field, a duplicate is a hard error:

```console
$ kira edit APP-1 --field components=api,api,web
kira: invalid item: field "components": duplicate value "api" in a unique list
```

The three *built-in* set-valued fields — `labels`, `blocked_by`, `aliases` — are
the exception: kira silently de-duplicates those, keeping first-occurrence
order, so a duplicate heals rather than blocks a write. Do not rely on that
behavior for your own fields.

**On `required`, `immutable`, `guarded`, `system`.** For a field *you* declare,
`immutable`, `guarded`, and `system` all have the same practical effect: the
field is not settable via `--field` at all, and kira reports it the same way.

```console
$ kira create ticket "Imported" --no-edit --field origin=import
kira: invalid item: --field: unknown or immutable field "origin"
  hint: editable fields: acceptance, components, description, due, epic, escalation,
        estimate, labels, notes, owner, points, priority, rank, reporter, resolution,
        reviewer, severity, sprint, state, subtype, title
```

The three differ only in *why* the field is off-limits, and that distinction is
live for the built-in fields, where kira implements the real semantics behind
each: `created` is `immutable` and is restored from the previous version on
every write; `state` is `guarded` and changes only through `kira move`'s
workflow checks; `updated` and `aliases` are `system` and are
recomputed by kira. Use the attribute that describes your intent, but treat all
three as "read-only to `--field`" for a field of your own.

`required` on a custom field is worth a warning: it is checked on every
mutation of an existing item, not only at creation. Adding `"required": true`
to a field that existing tickets do not have makes **every one of those tickets
unwritable** until you backfill a value:

```console
$ kira move APP-3 IN_PROGRESS
kira: invalid item: field "team": required, missing
```

#### `enum` — the two routes to a vocabulary

`enum` names a vocabulary. There are two places a vocabulary can come from, and
a field does not care which:

1. **Inline** — an entry in this schema file's `enums` array. `severity` above
   is inline; its allowed values are `low`, `medium`, `high` and they live in
   the schema file.
2. **Config-backed** — a vocabulary kira derives from `.kira/config.yaml`. The
   available names are `label` (from `labels.known`), `person` (from
   `people.known`), `priority` (from `priorities`), `subtype` (from `subtypes`)
   and `resolution` (from `resolutions`). Pointing a field at one of these
   means the field's allowed values are whatever the config says, and change
   when the config changes.

```json
{ "name": "escalation", "type": "enum", "enum": "priority" }
```

```console
$ kira edit APP-1 --field escalation=P1
Edited APP-1: escalation

$ kira edit APP-1 --field escalation=URGENT
kira: warning: field "escalation": value "URGENT" is not in the "priority" vocabulary
Edited APP-1: escalation
```

If a name appears both inline and in config, **config wins**. If a name matches
no vocabulary at all, the field is treated as an open vocabulary and is not
membership-checked.

Whether an out-of-vocabulary value is a warning or an error is a config
decision, not a schema one — see the strictness discussion in the worked
example below.

#### `states` and its relationship to `workflows:`

`states` is an inline list of `{key, category}` pairs on a `type: state` field.
It is **descriptive**. The list of states an item may actually occupy comes
from the `workflows:` block in `.kira/config.yaml` — not from the schema. kira validates an item's state
against `workflows.<type>` and deliberately ignores the schema's `states` array
when doing so. Editing the schema's `states` to nonsense changes nothing:

```console
$ # schema states edited to [{ key: NONSENSE, ... }, ...]
$ kira create ticket "State check" --no-edit
Created APP-2 State check
$ kira move APP-2 IN_PROGRESS
Moved APP-2: TODO -> IN_PROGRESS
```

The practical rule: **a type is usable if and only if it has a `workflows:`
entry in config.** A schema file adds fields to a type; it does not bring a
type into existence. Keep the schema's `states` in sync with your workflow as
documentation, but change behavior in `config.yaml`.

#### `target` and `source` on a `ref`

A `ref` field always needs `target`. What `target` means depends on whether
`source` is present:

- **No `source`** — `target` is another entity *type*, and the value must
  resolve to an existing item **of that type**. `epic`, `blocked_by`, and our
  `reviewer` field work this way. The reference is normalized to the target's
  immutable id on write, so it survives a renumber.
- **With `source`** — the reference is backed by a config vocabulary rather
  than an item file, and validation becomes a membership check against that
  vocabulary. `owner` and `reporter` (`target: person`, `source: people`) and
  `sprint` (`target: sprint`, `source: sprints`) are the built-in examples.
  Nothing needs to exist as a file on disk; the key just has to be in config.

The difference in practice is what a bad value gets you. An entity-target ref
reports a missing or wrong-typed *item*; a config-backed ref reports a value
outside a *vocabulary*.

#### `placement` and `section`

`placement: frontmatter` (the default) stores the value as a YAML key in the
item file's frontmatter. `placement: body` instead stores it as prose under a
named `## ` heading in the item's Markdown body — which is exactly how the
built-in `description` and `acceptance` fields work. `section` names that
heading and is mandatory for a body-placed field.

A body-placed value may not itself contain a `## ` heading line, because that
would create an ambiguous second section:

```console
$ kira edit APP-1 --field notes='## Sub
x'
kira: invalid item: --field "notes": a body-section value cannot contain a "## " heading line
```

## How schema files layer over the built-ins

kira embeds the `ticket`, `epic`, and `board` schemas in the binary. At startup
it loads those, then reads every `*.json` file in `.kira/schema/` and overlays
each one **by its `name` key**.

Two consequences matter:

1. **The overlay is whole-schema replacement, not a per-field merge.** A file
   whose `"name"` is `"ticket"` replaces the entire built-in ticket schema. If
   your file declares only `severity`, then `title`, `state`, `priority`, and
   every other built-in field simply stop being declared — and kira will report
   them as unknown. You must therefore start from a **complete copy** of the
   built-in schema and append to it. There is no diff or patch form.
2. **The `"name"` inside the file decides which type is overridden, not the
   filename.** `.kira/schema/mystuff.json` containing `"name": "ticket"`
   overrides `ticket`. Keeping the filename equal to the name is a strong
   convention, not a rule.

### Getting a starting-point copy

`kira init` materializes the built-in schemas into `.kira/schema/` for you:

```console
$ kira init --key APP
Initialized kira in .kira (project key APP)

$ ls .kira/schema/
board.json  epic.json  ticket.json
```

These files are byte-identical to the built-ins, so an untouched
`.kira/schema/` changes nothing — it is purely a starting point for editing.
That is the intended workflow: `kira init`, then edit `ticket.json` in place.

If a schema file goes missing later, `kira init --force` restores it, and
crucially **never overwrites a schema file that already exists** — your edited
`ticket.json` is left alone:

```console
$ rm .kira/schema/board.json
$ kira init --force
Initialized kira in .kira (project key APP)
$ ls .kira/schema/
board.json  epic.json  ticket.json
```

Be aware that `--force` *does* rewrite `.kira/config.yaml` and the templates
back to their scaffolded defaults. In a repo with a customized config, it is
safer to copy the one missing schema file from another clone than to reach for
`--force`.

## Worked example: adding a custom field to `ticket`

### 1. Get a copy of the built-in schema

```console
$ kira init --key APP
Initialized kira in .kira (project key APP)
$ ls .kira/schema/
board.json  epic.json  ticket.json
```

### 2. Add the field and its vocabulary

Open `.kira/schema/ticket.json` and append to the existing `fields` array,
then add a sibling `enums` array:

```json
    { "name": "acceptance", "type": "markdown", "placement": "body", "section": "Acceptance criteria" },
    { "name": "severity", "type": "enum", "enum": "severity" }
  ],
  "enums": [
    { "name": "severity", "values": ["low", "medium", "high"] }
  ],
```

Nothing else in the file changes; every built-in field stays exactly where it
was.

### 3. Set it on a new ticket

```console
$ kira create ticket "Checkout returns 500 on retry" --no-edit --field severity=high
Created APP-1 Checkout returns 500 on retry
```

```console
$ cat .kira/tickets/01KYHXN2EE1P46KRKNXWF17C8C.md
---
id: 01KYHXN2EE1P46KRKNXWF17C8C
number: APP-1
aliases: []
type: ticket
title: "Checkout returns 500 on retry"
state: TODO
labels: []
epic: null
blocked_by: []
created: 2026-07-27T19:23:56+05:30
updated: 2026-07-27T19:23:56+05:30
severity: high
---

## Description

## Acceptance criteria
```

`severity: high` sits in the frontmatter alongside the built-in keys.

### 4. Try a value outside the vocabulary

What happens depends on `labels.strict` in `.kira/config.yaml`, which is the
strictness setting a schema-declared enum field inherits when no more specific
config vocabulary governs it. With the scaffolded default of
`labels.strict: false`, an unknown value is a warning and the write proceeds:

```console
$ kira create ticket "Bad value" --no-edit --field severity=critical
kira: warning: field "severity": value "critical" is not in the "severity" vocabulary
Created APP-2 Bad value
```

Set `labels.strict: true` and the same command is rejected:

```console
$ kira create ticket "Bad value" --no-edit --field severity=critical
kira: invalid item: field "severity": value "critical" is not in the "severity" vocabulary
```

`--force` downgrades that error back to a warning for a one-off exception.

A typo'd *field name* is always an error, with a suggestion:

```console
$ kira edit APP-1 --field sevrity=high
kira: invalid item: --field: unknown or immutable field "sevrity"
  hint: did you mean `--field severity=...`?
```

### 5. Clear it

An empty value removes the key entirely rather than writing a blank:

```console
$ kira edit APP-1 --field severity=
Edited APP-1: severity
```

```console
$ head -14 .kira/tickets/01KYHXN2EE1P46KRKNXWF17C8C.md
---
id: 01KYHXN2EE1P46KRKNXWF17C8C
number: APP-1
aliases: []
type: ticket
title: "Checkout returns 500 on retry"
state: TODO
labels: []
epic: null
blocked_by: []
created: 2026-07-27T19:23:56+05:30
updated: 2026-07-27T19:24:07+05:30
---
```

The `severity` line is gone.

## Worked example: a ref field and a body-placed field

Add two more entries to the same `fields` array:

```json
    { "name": "reviewer", "type": "ref", "target": "ticket" },
    { "name": "notes", "type": "markdown", "placement": "body", "section": "Notes" }
```

Both can be set in one `edit` (repeat `--field` per key):

```console
$ kira edit APP-1 --field severity=medium --field reviewer=APP-2 \
      --field notes='Only reproduces when the HTTP connection is reused.'
Edited APP-1: body, reviewer, severity
```

The resulting file shows both placements at once — `reviewer` and `severity` in
frontmatter, `notes` as a `## Notes` section in the body:

```console
$ cat .kira/tickets/01KYHXN2EE1P46KRKNXWF17C8C.md
---
id: 01KYHXN2EE1P46KRKNXWF17C8C
number: APP-1
aliases: []
type: ticket
title: "Checkout returns 500 on retry"
state: TODO
labels: []
epic: null
blocked_by: []
created: 2026-07-27T19:23:56+05:30
updated: 2026-07-27T19:24:07+05:30
reviewer: 01KYHXN2K84E652M9PBJA867KA
severity: medium
---

## Description

## Acceptance criteria

## Notes

Only reproduces when the HTTP connection is reused.
```

Note that `reviewer` stores the target's immutable id, not `APP-2`, even though
you typed `APP-2`. That is deliberate: display numbers can be renumbered after
a merge, ids cannot, so the link never goes stale.

### Ref validation in practice

`target: ticket` is enforced on write. Pointing `reviewer` at an epic
(`APP-3` below) fails, as does pointing it at nothing at all:

```console
$ kira edit APP-1 --field reviewer=APP-3
kira: invalid item: field "reviewer": references a non-existent ticket "01KYHXNDFECKVTBVGAMJAQATD6"

$ kira edit APP-1 --field reviewer=APP-99
kira: invalid item: field "reviewer": references a non-existent ticket "APP-99"
```

Self-reference is caught separately:

```console
$ kira edit APP-1 --field reviewer=APP-1
kira: invalid item: field "reviewer": an item cannot link to itself
```

If a wrong-typed reference reaches a file some other way — a hand edit, a bad
merge — `kira doctor` reports it with a clearer message, alongside a warning
for any frontmatter key the schema does not declare:

```console
$ kira doctor
error   ref-type        APP-1 reviewer: reviewer points at APP-3 of type epic, expected ticket
warning schema          APP-1 owning_squad: unknown top-level field "owning_squad"
info    hooks           no tracked hooks in .kira/hooks
info    index-freshness index not built yet; it builds on first read
problems found: 1 error, 1 warning, 2 info
see `kira config explain` for the rules these checks enforce
kira: doctor: 1 problem found
```

That second warning is the reason to declare a field rather than just typing it
into a file: an undeclared key is tolerated but flagged forever.

## Current limits

Be aware of what this mechanism does and does not do today.

**New types work, but only as far as config takes them.** A type exists because
it has a `workflows:` entry in `.kira/config.yaml`; add one and
`kira create <type>`, `kira move`, `kira list`, and the board all handle it.
Pair it with a `.kira/schema/<type>.json` and its custom fields work through
`--field` too, including `ref` fields, which `kira doctor` checks for both
dangling and wrong-typed targets exactly as it does the built-in `epic` and
`blocked_by`. What you do *not* get is per-type presentation or storage: items
of every type live in `.kira/tickets/`, and there is no way to give a new type
its own rendering.

If you only need a distinct category of work rather than a whole type, the
`subtype` field (`kira create ticket --subtype bug`) or a label are
lighter-weight alternatives — both configurable in `.kira/config.yaml`, and
both filterable from `kira list`.

**`representation.label` and `.icon` are inert.** `list_columns` is wired up:
it names the columns `kira list` renders, custom fields included, with
`ui.list.columns` taking precedence over it. `label` and `icon` are accepted
and validated as JSON but nothing reads them yet -- `kira show` and the TUI use
fixed built-in rendering. Changing those two in a schema file changes nothing
on screen. The same is true of `identity` (`style`/`prefix`) — id generation is
driven by config, not by the schema.

**Declared extra fields are not `kira list` columns.** A custom field
round-trips through the item file, is validated on every write, is shown by
`kira show`, is carried in `--json` under a `fields` object, and is filterable
with `--query`. What it cannot yet be is a column in the `kira list` table —
that is driven by `ui.list.columns`, not by the schema.

To see one per item, use `--json`:

```console
$ kira list --query 'severity = high' --json | jq -r '.items[] | .number'
APP-1
```

None of the above is a bug — these are parts of the system that are declared but
not yet built. See `09-limitations-and-roadmap.md` for the fuller picture of
what is planned.
