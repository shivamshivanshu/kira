---
id: 01M1BZ2VYQNR47RWDFEC8KSHZT
number: KIRA-140
aliases: []
type: ticket
title: "DECISION: adopt or drop the design doc's free-transitions change (Decision 5)"
state: REVIEW
priority: P2
labels: []
epic: null
blocked_by: []
created: 2026-08-31T15:10:35+02:00
updated: 2026-09-01T13:39:32+02:00
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01M1BZ2W975ZH6F6Z7MXZEQE8Y author=Shivam-Shivanshu ts=2026-08-31T15:10:36+02:00 -->
Owner sign-off needed. This is a documented DEVIATION: docs/design/CORE-3-extensible-schema.md Decision 5 says the enforced state-transition graph is dropped (state becomes a plain enum, any value to any value, and the blockers_closed require-guard becomes a state-independent done-category close check), landing in 'Phase 2+'. Phases 2, 3a, 3b, 3c all shipped WITHOUT it, deliberately.
Reason for deferring: it is a user-visible policy change, not a schema-shape change; CLAUDE.md and README document the enforced graph as a product feature; and it deliberately invalidates the transition-enforcement tests, which are precisely the characterization guarantee that de-risked every other phase. Both plan critics independently agreed deferral was right.
Measured blast radius if adopted: enforcement threads through core/{workflow,move,sync,history,explain,hints}.go and config/{defaults,validate}.go and datamodel/workflow.go; roughly 19 config test hits, core/test/mutations_test.go, contract goldens err-bad-transition.err, err-require-guard.err and config-explain.json; about 10 e2e scripts carrying transitions: fixtures (move, wip-block, blockers-closed, hooks-closes); plus README:25 and CLAUDE.md.
Mitigation already shipped: doctor.StateDrift (warning) reports when a schema's declared states diverge from cfg.Workflows, so the two sources of truth cannot silently rot while this stays open.
Decide: adopt as its own ticket, or amend the design doc to record that the enforced graph is kept and Decision 5 is withdrawn.
<!-- /kira:comment -->
