---
id: 01KYKDP5S4GFJFM0SJREN2XZAE
number: CORE-51
aliases: []
type: ticket
title: "Surface schema-declared fields in show (human and --json)"
state: DONE
priority: P1
labels: []
epic: 01KYKDNPSZEGK032FRCJCVNKE1
blocked_by: []
created: 2026-07-28T09:23:24+05:30
updated: 2026-07-28T09:35:22+05:30
---

A schema-declared custom field round-trips correctly in the item file but
`kira show` does not display it and `kira show --json` omits it entirely, so
the only way to read one back is to open the file or grep for it.

Verified 2026-07-28 (scratch repo, `severity` enum added to `.kira/schema/ticket.json`):

    $ kira create ticket "sev test" --no-edit --field severity=high
    $ grep severity .kira/tickets/*.md
    severity: high                      # persisted correctly

    $ kira show VSCH-1                  # human output: no severity line
    $ kira show VSCH-1 --json | grep -c severity
    0

**Where it lives**: `showResultOf` in `internal/core/show.go` builds
`datamodel.ShowResult` from a fixed field list. Extras live on the item as
`ExtraScalars`/`ExtraLists` (and body-placed fields as body sections);
`entityschema.ProjectItem` already knows how to project an item's declared
fields into a value map -- that is the natural source rather than a second
traversal.

**Design points to settle**:
- JSON shape: nest custom fields under a `fields`/`extra` object, or inline
  them as top-level keys? Inlining risks collision with future built-ins;
  nesting is a shape change consumers must learn. `schema/kira.json` is
  published and generated, so whichever is chosen must be regenerated.
- Human `show`: where do custom fields render -- after the built-in header
  block, or interleaved per the schema's field order?
- Body-placed fields already appear in the body; they should not be printed
  twice.

**Depends on nothing.** Blocks the query ticket only loosely (they can share
a projection helper but do not have to land in order).
