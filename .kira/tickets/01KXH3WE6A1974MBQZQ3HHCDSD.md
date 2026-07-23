---
id: 01KXH3WE6A1974MBQZQ3HHCDSD
number: CLI-8
aliases: []
type: ticket
subtype: feature
title: "kira config explain: print this repo's live rules with provenance"
state: DONE
priority: P2
labels: []
epic: null
blocked_by: []
created: 2026-07-15T01:37:52+05:30
updated: 2026-07-23T14:26:56+05:30
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01KXH3X533H3A8DMANZPN5XNPD author=Shivam-Shivanshu ts=2026-07-15T01:38:16+05:30 -->
Spec = section 2 of claude_notes/scratchpad/20260715/kira-hygiene-verdict.md (seams verified by audit): register at internal/cli/config.go:32; core.Explain(cfg) + result struct in datamodel/results.go (schema-register, additive); input is the merged effective config via s.Config() so output is live truth; sections: identity/EffectiveBoards/id.style, workflows per type reusing transitionHint (core/hints.go:40) incl. the blockers_closed leniency wording (dangling/unknown = satisfied+warn), commit linking rendered per marker, vocab incl. system labels (hoist capturedLabel to datamodel), merge/sync/workon; provenance per section default|repo|user via diff against config.Default() + tracked user-tier keys; document KIRA_ICONS here; doctor closing line cross-links; errx hints on commit-link misses. ~200 LoC + one contract test (additive golden).
<!-- /kira:comment -->
