---
id: 01M1HBDMYXSSS1KYZXJMH3784Q
number: DX-15
aliases: []
type: ticket
title: "Cleanup pass: dead code, stale container naming, and a real test-coverage gap"
state: DONE
resolution: done
priority: P3
labels: []
parent: null
blocked_by: []
created: 2026-09-02T17:22:24+02:00
updated: 2026-09-02T17:22:39+02:00
---
Five read-only audits (low layers, mid layers x2, cli/tui/webui, cross-layer)
reported. Most of what they found was that there was little to find -- no dead
code outside three items, no trivial tests in the low or presentation layers
(one agent read all 221 and cut none), no color literals outside the theme
package, and a clean clock/timezone sweep. The findings acted on:

**A real coverage gap, not just dead code.** `epicProgress` (singular) had zero
production callers -- but the five semantic progress tests, including the two
capability tests written for CORE-63, exercised IT rather than the production
`allContainerProgress`. A bug in the code that actually runs would not have been
caught. The tests are rewritten against the real function, which required
flattening their children-map fixtures into item lists. The dead implementation
and its differential-oracle test are gone, since the direct tests now cover
nested rollup, the non-container child, and cycle detection outright.

**A user-facing message left wrong by the generalization.** A cycle among any
container type printed "epic cycle detected" at three sites. Nothing pinned the
string, so it now says "parent cycle detected".

**Stale naming completed.** `EpicProgress` -> `ContainerProgress`,
`allEpicProgress` -> `allContainerProgress`, `walkEpic` -> `walkDescendants`,
`epicGroupKey` -> `parentGroupKey`, plus the `validateGraph` comment still
claiming it checks "parent-is-epic". `ContainerProgress` appears in no JSON
result shape and not in `schema/kira.json`, so this is internal only.

**Dead code**: `entityschema.NewRefResolver`/`itemRefResolver` (superseded by
`core/refresolver.go`, which resolves through `id.Resolver` so it handles
aliases), `entityschema.BuiltinNames`, and `id.Allocate`. The `RefResolver`
interface stays -- `core` implements it. `id.Allocate`'s three tests were
retargeted onto `id.HighestN` rather than deleted, keeping the 500-trial fuzz
that is the only coverage of its union-of-numbers-and-aliases logic.

**Consistency**: `Batch` now caches the container set once in `BeginBatch`
instead of rebuilding it per mutation, matching how the same file already
caches `enums` for exactly that reason.

**Sharing**: `datamodel.IsDropped` replaces the same predicate written twice.
It takes the dropped slice rather than a `Config`, because the TUI copies that
slice in once and deliberately carries no config further -- a `Config` method,
as first proposed, would not have fitted the call site.

Also: one duplicated `--field key=value` parsing loop, one test that asserted
nothing and guarded an impossible panic, and three comments that restated their
code.
