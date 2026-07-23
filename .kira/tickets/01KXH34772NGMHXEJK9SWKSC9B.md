---
id: 01KXH34772NGMHXEJK9SWKSC9B
number: DATA-7
aliases: []
type: ticket
subtype: task
title: "Schema: register missing result types, add conformance validation, harden generator"
state: REVIEW
priority: P2
labels: []
epic: null
blocked_by: []
created: 2026-07-15T01:24:39+05:30
updated: 2026-07-23T14:26:56+05:30
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01KXH347CSHK09NRQKP7Y80RPH author=Shivam-Shivanshu ts=2026-07-15T01:24:39+05:30 -->
Missing registrations (internal/schema/schema.go:17 topLevelTypes): WorkonResult, HooksInstall/ValidateResult, AutomationList/TrustResult, doctor.Report, syncx.Report, plus the ad hoc map[string]string template result (cli/create.go:105) — committed kira.json has 66 $defs with none of these; the hooks surface is half-registered. schema_test.go only checks Generate() matches the artifact; no jsonschema validator exists anywhere in go.mod/tests, so drift is undetectable and a nil-slice-emits-null regression would pass; the golden -update flow blesses any generator bug.

Generator hazards: $defs keyed by bare t.Name() — register() early-returns on a seen name, so a doctor.Report/syncx.Report collision silently skips the second type and both $refs resolve to the first shape; addFields panics on an embedded pointer (latent — no current struct embeds one); the required rule excludes non-omitempty pointer fields although encoding/json always emits them as null (schema looser than reality, not invalid); named string enums (DiffStatus etc.) emit bare strings; no root schema covers the bulk []BulkOutcome shape.

Fix: register the missing result types (typed template result; decide the layer for doctor/syncx report structs); guard test that every exported datamodel *Result type is in topLevelTypes; conformance test loading schema/kira.json with a JSON Schema validator and validating each contract golden against its $def (and raw stdout in TestJSONContract); reflect.Type-keyed collision check; pointer-embed deref; drop the Kind!=Pointer condition from required; enum registry from the existing inventory slices; root oneOf incl. the BulkOutcome array; unit tests over a synthetic struct so regressions fail independent of the golden.

Files: internal/schema/schema.go, internal/schema/kira.json, internal/schema/schema_test.go, tests/contract, internal/cli/create.go, go.mod
<!-- /kira:comment -->
