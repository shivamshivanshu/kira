---
id: 01KYKDR281PCE90TK3B4T5PZR5
number: CORE-55
aliases: []
type: ticket
title: "Representation-driven labels and icons"
state: TODO
priority: P3
labels: []
epic: 01KYKDNPSZEGK032FRCJCVNKE1
blocked_by: []
created: 2026-07-28T09:24:25+05:30
updated: 2026-07-28T09:25:48+05:30
---

A schema's `representation.label` and `representation.icon` are parsed and
then ignored. Nothing reads them, so a custom type cannot present itself with
its own glyph or display name -- it inherits whatever the hardcoded rendering
decides.

Today the TUI picks glyphs by switching on the item's concept/type, and only
`TypeTicket` and `TypeEpic` exist as cases (there is no `TypeBoard`). There is
no name-to-glyph registry to look a schema's declared `icon` up in, which is
the missing piece rather than the type switch itself.

**Scope**:
- A registry mapping a schema's declared icon name to a glyph, resolved per
  the existing `ui.icons` mode (`auto|nerd|emoji|text`) so a declared icon
  still degrades correctly on a terminal without nerd fonts. Do not bypass
  that setting.
- `representation.label` used wherever a type is named in output, in place of
  the raw type string.
- An unknown/unmapped icon name must fall back cleanly, not render a missing
  glyph box.

**Depends on CORE-54** for the by-name display accessor on the read-projection
DTOs; the two are the same "make rendering schema-driven" seam and CORE-54
establishes it. Sequence after it rather than in parallel to avoid both
touching the same DTOs.

**Test fallout**: ~10-12 TUI goldens. Re-baseline deliberately.
