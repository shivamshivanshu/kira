---
id: 01KXGWCY55YXM3K402GKMAVDH1
number: CORE-3
aliases: [KIRA-102]
type: ticket
subtype: feature
title: "Configurable ticket schema: custom sections + typed custom fields per type"
state: DONE
labels: []
epic: null
blocked_by: []
created: 2026-07-14T23:27:04+05:30
updated: 2026-07-28T12:12:19+05:30
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01KY77VKZTB27RKKZ9FS7CC6WP author=Shivam-Shivanshu ts=2026-07-23T15:50:37+05:30 -->
Phase 1 landed (commit on main): internal/entityschema engine (schema/loader/validator), embedded built-in ticket/epic/board schemas, schema-driven ProjectItem/ConfigVocab, kira init materialization, and a conformance test proving the built-in schemas accept all existing items with zero violations. Additive/non-breaking. Phases 2-6 (ref-integrity enforcement, typed extra-field round-trip, new entity types, schema-driven rendering, Item replacement) remain per docs/design/CORE-3-extensible-schema.md; state-transition relaxation (Decision 5) lands in phase 2+.
<!-- /kira:comment -->

<!-- kira:comment id=01KYKQBFPY5Q21S32EQPC4T767 author=Shivam-Shivanshu ts=2026-07-28T12:12:19+05:30 -->
Closing: phases 1-3 shipped, and the remaining work is tracked as the CORE-50 epic (CORE-51 show, CORE-52 query, CORE-53 dangling refs, CORE-54 list columns, CORE-56 ref targets all done; CORE-55 labels/icons open with design questions recorded on it). CORE-49 corrected this doc's phase-4 framing. Phase 6 remains explicitly maybe-never.
<!-- /kira:comment -->
