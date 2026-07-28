---
id: 01KYKYKV7PWZ7JHWZ1WT1H0K2D
number: CORE-58
aliases: []
type: ticket
subtype: bug
title: "Draft path silently drops schema-declared fields"
state: TODO
priority: P1
labels: []
epic: null
blocked_by: []
created: 2026-07-28T14:19:13+05:30
updated: 2026-07-28T14:19:13+05:30
---

`create --from-file` and the interactive editor discard any schema-declared
custom field in the draft's frontmatter. No warning, no error, exit 0 -- the
item is created without the value.

Repro, with `severity` declared in `.kira/schema/ticket.json`:

    kira create ticket --from-file - <<'IN'
    ---
    title: "t"
    severity: high
    ---
    IN
    kira show <new> --json | jq .fields    # null -- severity is gone

`--field severity=high` works, because applyFieldEdits runs against the
assembled item. The draft path does not: `draft` (internal/core/draft.go) is
a fixed Go struct, so `parseDraft` decodes frontmatter into it and yaml
discards every key the struct lacks. `codec.Parse` preserves unknown keys as
ExtraScalars/ExtraLists for real items; the draft type has no equivalent.

The write half of the same gap: `serializeDraft` emits a hardcoded frontmatter
key list, so `--print-template` and the editor buffer never offer a declared
custom field. A user editing interactively cannot discover or set one, and if
they add it by hand it is dropped on save.

This is the last place the create path is hardcoded rather than schema-driven.
Body sections are already derived from the schema (templates were removed in
favour of that); frontmatter is not.

Fix direction: give `draft` the same extras carriage `datamodel.Item` has, and
drive `serializeDraft`'s key list off the type's schema rather than a literal
sequence of WriteString calls. Silent drop is the worst part -- even before
the full fix, an unknown draft key should warn rather than vanish.
