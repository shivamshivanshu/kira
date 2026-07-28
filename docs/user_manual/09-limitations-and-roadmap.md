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

They are also filterable. `--query` resolves a schema-declared name and
compares by its declared type, so `points > 5` is a numeric comparison rather
than a string one:

```console
$ kira list --query 'severity = high'
$ kira list --query 'points > 5'
$ kira list --query 'severity IN (low, high)'
$ kira list --query 'severity IS EMPTY'
```

An ordered comparison is refused on a type that has no order (`severity > low`
errors rather than comparing text). A query spans item types, so a field only
one type declares simply does not match items of the others.

Two gaps remain. A custom field cannot be a `kira list` **column** — the table
is driven by `ui.list.columns`, not the schema (CORE-54). And body-placed
fields (`placement: body`) are neither in the `fields` object nor filterable;
their text is already in the item body, which `show` prints in full.

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
