---
id: 01KXGWCY55YXM3K402GKMAVDH1
number: CORE-3
aliases: [KIRA-102]
type: ticket
subtype: feature
title: "Configurable ticket schema: custom sections + typed custom fields per type"
state: IN_PROGRESS
labels: []
epic: null
blocked_by: []
created: 2026-07-14T23:27:04+05:30
updated: 2026-07-23T15:21:54+05:30
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01KY77VKZTB27RKKZ9FS7CC6WP author=Shivam-Shivanshu ts=2026-07-23T15:50:37+05:30 -->
Phase 1 landed (commit on main): internal/entityschema engine (schema/loader/validator), embedded built-in ticket/epic/board schemas, schema-driven ProjectItem/ConfigVocab, kira init materialization, and a conformance test proving the built-in schemas accept all existing items with zero violations. Additive/non-breaking. Phases 2-6 (ref-integrity enforcement, typed extra-field round-trip, new entity types, schema-driven rendering, Item replacement) remain per docs/design/CORE-3-extensible-schema.md; state-transition relaxation (Decision 5) lands in phase 2+.
<!-- /kira:comment -->

<!-- kira:comment id=01M1BBWHY2EY48Y0J9C7Y302JQ author=Shivam-Shivanshu ts=2026-08-31T09:35:05+02:00 -->
Phase 2 landed (merge cdff347). internal/entityschema now has its first runtime consumer: core.Store lazily loads .kira/schema layered over embedded builtins. entityschema.NewRefResolver(items) maps ULID->type. doctor.NonEpicParents (hardcoded to the epic field) replaced by schema-driven doctor.WrongTypeRefs, which now also catches wrong-typed blocked_by; new doctor.StateDrift (warning) guards schema-vs-cfg.Workflows state divergence while Decision 5 stays deferred. Critical invariant: ref fields carrying a 'source' attribute point at config vocabulary (person, sprint), not items, and are skipped -- otherwise every owner reports as a broken ref; pinned by TestOwnerAndSprintValuesAreNotItemRefs (verified to fail when the guard is removed). Verified: gofmt/build/vet clean, golangci-lint v2.12.2 0 issues, full -race suite + e2e green, kira doctor byte-identical on the real corpus. Deferred out of phase 2: schema-sourced completion (config vocab already wins in ResolveEnums, so no behavior change for builtins).
<!-- /kira:comment -->

<!-- kira:comment id=01M1BSZ5WDA114DG83GCRJJ3TY author=Shivam-Shivanshu ts=2026-08-31T13:41:11+02:00 -->
Phase 3c landed (merge c4f0b67). Schema-declared custom fields are now queryable and visible in JSON. internal/index gains an extras table (item_id, name, ord, kind, value) with schemaVersion 6->7 so stale caches rebuild; kind distinguishes a scalar from a single-element list, and a NULL-valued list row encodes an explicitly empty list so 'key: []' stays distinct from an absent key. internal/query threads a schema-derived []ExtraField spec into both hardcoded tables (parser fields set and eval accessors) via Parse(input, extras) and Options.Extras, covering =, !=, IN, IS EMPTY, ORDER BY and did-you-mean. ListItem/ShowResult gain extras with omitempty (ExtraJSON renders anyOf[string, string[]]), schema/kira.json regenerated. config/validate.go parses saved queries with nil extras and tolerates only unknown-field errors -- that site has no *Store, and rejecting would brick config load for any repo whose saved filter names a custom field; validity is re-checked at every use site with the real spec. Verified end-to-end against a WARM cache: severity=HIGH, story_points>5, components=api all match, field removal leaves no stale row, and '>' on a non-numeric extra errors rather than silently string-comparing. Full suite + e2e green, golangci-lint 0 issues. Known deferred: two schemas declaring the same field name with conflicting traits resolve first-wins by alphabetical schema order.
<!-- /kira:comment -->
