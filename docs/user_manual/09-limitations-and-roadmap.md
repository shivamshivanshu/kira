# Current Limitations and Roadmap

kira is under active development, and some things you might reasonably expect
it to do it does not do yet. This page lists those gaps honestly: what you
might try, what actually happens today, and whether a fix is planned. Every
entry here is drawn from kira's own open tickets — kira tracks its own
development in kira, so the tickets behind this page live in `.kira/tickets/`
in the kira repository alongside the code.

## Custom fields are readable, but not yet searchable

You can add a field to the built-in `ticket` or `epic` schema, set it with
`kira create --field severity=high`, and change it later with `kira edit
--field severity=low`. The value is validated against the schema and written
into the item's Markdown file, where it round-trips through edits and merges
intact. `kira show` prints it after the built-in header block, and `kira show
--json` carries it under a `fields` object with its declared type preserved —
an `int` field comes back as a number, a `list` as an array:

```console
$ kira show KIRA-1 --json | jq .fields
{
  "components": [ "api", "web" ],
  "points": 8,
  "severity": "high"
}
```

What does not work yet is filtering. `kira list --query 'severity = high'`
fails with *unknown field "severity"* — the query grammar still knows only the
built-in field names, so custom fields cannot be filtered or reported on from
`list`.

Body-placed fields (`placement: body`) are not part of the `fields` object;
their text is already in the item body, which `show` prints in full.

**Roadmap:** active. Making the query grammar schema-aware is CORE-52, under
the CORE-50 epic.

## Defining your own item type works, but dangling custom refs on it can go unchecked

kira ships two item types, `ticket` and `epic`, but a third type is not a Go
change: give it a `workflows:` entry in `.kira/config.yaml` — the same block
that defines `ticket`'s and `epic`'s states and transitions — and `kira
create <type>`, `edit`, `move`, `list`, `show`, and `doctor`'s state checks
all handle it immediately, with the same built-in field set (title, state,
labels, owner, priority, epic, blocked_by, body sections, and the rest).
No schema file is required for this much to work. Adding a matching
`.kira/schema/<type>.json` on top gives the new type custom fields through
`--field`, exactly like customizing `ticket` or `epic` — see
[Custom Fields and Schema](06-custom-fields-and-schema.md).

What is not yet finished: `kira doctor`'s dangling-reference scan only
follows the built-in `epic`, `blocked_by`, and `links` relationships. A
custom `ref` field — on a new type or on `ticket`/`epic` — is validated when
you set it through `--field`, but if a bad reference reaches the file some
other way (a hand edit, a bad merge) and points at nothing at all, `doctor`
does not flag it. (A *wrong-typed* reference — pointing at an item of the
wrong entity type — is caught; it's specifically the "points at nothing"
case that the dangling-ref scan misses for custom fields.)

If you need a distinct category of work without a new type, the `subtype`
field (`kira create ticket --subtype bug`) or a label are lighter-weight
alternatives — both configurable in `.kira/config.yaml` and both filterable.

**Roadmap:** active. Folding the dangling-reference scan into the same
schema-driven check that already covers ref *type* mismatches, so it walks
every declared ref field generically instead of only the three built-in
ones, is planned work on CORE-3.

## Display settings in a schema have no effect

Each schema file carries a `representation` block: a display `label`, an
`icon`, and a `list_columns` array naming the columns `kira list` should show.
These are accepted by the schema loader without complaint.

They are inert. Changing `"list_columns"` to `["number", "title"]` still
produces the standard NUMBER / STATE / TYPE / PRIORITY / TITLE table. Changing
`"label"` from `"Ticket"` to `"Story"` still prints `type: ticket` in
`kira show`. The fields exist in the file format so that schemas written today
stay valid later, but nothing reads them yet.

Column choice is therefore fixed. `kira list` renders a set layout, and the way
to get a different shape is `--json` plus your own formatting.

**Roadmap:** active, and paired with the two gaps above — rendering driven by a
schema's `representation` block is planned as part of CORE-3.

## Boards and sprints are not tracked items

Tickets and epics are files under `.kira/tickets/`, each with its own git
history, its own comment thread, and its own `kira show`. Boards and sprints
are not. They are entries in `.kira/config.yaml`, managed through
`kira board create` / `rename` / `archive` and `kira sprint create` / `activate`
/ `close`.

The practical consequences:

- `kira show CORE` fails with *resolves to no item*. There is no detail view
  for a board or a sprint; `kira board list` and `kira sprint list` are the
  only views.
- `kira comment CORE -m "..."` fails the same way. You cannot attach a
  discussion thread to a board or a sprint.
- `kira log CORE` fails too. Changes to a board are committed, but as edits to
  `.kira/config.yaml` — so the history is the config file's history, mixed in
  with every other config change, not a per-board timeline.

This is a deliberate split, not a bug: a board is a grouping construct that
lives in configuration, while a ticket is a first-class tracked artifact. It is
worth knowing about because the asymmetry is invisible until you try to comment
on a board.

**Roadmap:** active but unscheduled. CORE-39 tracks moving boards and sprints
onto the same file-backed storage as tickets, which would give them the history
and commentary that tickets have.

A related, deeper change — reworking kira's internal representation of items
entirely — is recorded in the design notes as "maybe never." It is not on a
committed roadmap, and you should not plan around it.

## The generated output schema is missing some value constraints

`kira schema` prints a JSON Schema describing the shape of every `--json`
output, which you can use to validate kira's output in a script or pipeline.

Most enumerated fields in it carry their allowed values. A few do not, and
appear as a plain `"type": "string"` instead: the `severity` and `class` fields
of `kira doctor` findings, the step status in `kira sync` reports, the `kind`
field of `kira log` entries, and the `source_kind` field of `kira blame`
output. Validation against the schema will still pass on correct output — it
just will not catch a bad value in those specific fields, and code generators
reading the schema will produce a string where an enumerated type would be
better.

**Roadmap:** active. DATA-10 tracks extending the enum constraints to cover
these fields.

---

This page reflects kira's open tickets as of the last documentation update. For
the current live state, run `kira list --category todo` in the kira repository
— it lists every open, not-yet-started ticket across all boards, and
`kira list --category doing` shows what is in flight. `kira show <id>` on any
of them gives the full description and discussion.
