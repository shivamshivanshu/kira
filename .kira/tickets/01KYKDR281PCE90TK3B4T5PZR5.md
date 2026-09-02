---
id: 01KYKDR281PCE90TK3B4T5PZR5
number: CORE-55
aliases: []
type: ticket
title: "Representation-driven labels and icons"
state: IN_PROGRESS
priority: P3
labels: []
epic: 01KYKDNPSZEGK032FRCJCVNKE1
blocked_by: []
created: 2026-07-28T09:24:25+05:30
updated: 2026-09-02T13:56:36+02:00
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

<!-- kira:comment id=01KYKP53F3YCWZN6QFTNX5ZESY author=Shivam-Shivanshu ts=2026-07-28T11:51:21+05:30 -->
Investigation 2026-07-28, before building.

Footprint is smaller and cost higher than the ticket assumed:

- iconSet.typeGlyph is called in exactly one place, internal/tui/tree.go:214 (the type glyph beside a tree row). Board cards and every CLI surface do not use it.
- The TUI has no access to schemas at all -- zero references to entityschema anywhere under internal/tui. Wiring icons means plumbing the schema map through the model.
- representation.Label is read nowhere in the codebase today.

Two design questions to settle first, neither of which the ticket answers:

1. What is 'icon' -- a registry key or a literal glyph? A key ("icon": "epic") can only select from the glyphs kira already ships, so a custom type cannot have a genuinely new glyph; that is a small feature. A literal ("icon": "@") is flexible but bypasses the nerd/emoji/text degradation in glyph.pick, so it breaks on terminals without the font -- exactly what ui.icons exists to prevent. A third option is a per-mode object, which is the honest shape but more schema surface.

2. Where does 'label' render, and does it change default output? The builtin ticket schema declares label "Ticket" while every surface prints the lowercase type name. Using label in kira show's type: line and the list TYPE column would change default output for every user, the same trap CORE-54 hit. The consistent choice with CORE-54 is to keep default output byte-identical, which means aligning builtin labels to the lowercase type names and leaving the mechanism for custom types only -- but that makes the builtin declarations look redundant.

Recommendation: this is P3 with a thin payoff (one glyph in one view) and real plumbing cost. Worth deferring behind cheaper work, or reducing to just 'label', which needs no TUI plumbing. Not started; moved back to TODO.
<!-- /kira:comment -->
