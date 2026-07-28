---
id: 01KYKDREZ0ARHHQCZ2MZ08MWPR
number: CORE-56
aliases: []
type: ticket
title: "Allow .kira/schema overrides for ref targets"
state: DONE
priority: P3
labels: []
epic: 01KYKDNPSZEGK032FRCJCVNKE1
blocked_by: []
created: 2026-07-28T09:24:38+05:30
updated: 2026-07-28T12:08:11+05:30
---

Deferred from CORE-3 phase 1: a schema override in `.kira/schema/*.json`
cannot currently redirect what a `ref` field targets -- ref targets resolve
against the built-in expectations, so a repo cannot point a ref at a type it
has defined itself.

This is the last explicitly-deferred item from the design doc's phase 1
notes ("`.kira/schema` overrides for ref targets deferred to Phase 4, when
custom [types exist]"). Now that a brand-new item type works generically off
a `workflows:` config entry (verified in CORE-49), a ref field pointing at
such a type is a reachable configuration, which is what makes this worth
closing.

**Check first**: confirm the gap is still real and characterize it precisely
before building. `FieldDef.Target` and `FieldDef.TargetsEntity(schemas)`
already consult the loaded schema set, so part of this may already work --
the remaining hole may be narrower than the phase-1 note implies. Write the
failing case first (a custom type plus a ref field targeting it, exercised
through `--field` and through `doctor`) and let it define the scope.

**Related**: CORE-53 (the dangling-ref scan must cover whatever targets
become expressible here).
