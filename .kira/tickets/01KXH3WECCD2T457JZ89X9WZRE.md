---
id: 01KXH3WECCD2T457JZ89X9WZRE
number: CORE-34
aliases: []
type: ticket
subtype: task
title: "Convention knobs batch: captured label, commit caps, link-type vocab, prompt-mode hint, inert Comments heading"
state: TODO
priority: P2
labels: []
epic: null
blocked_by: []
created: 2026-07-15T01:37:52+05:30
updated: 2026-07-15T01:37:52+05:30
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01KXH3X5953MDTAJ6BA5SP3ET9 author=Shivam-Shivanshu ts=2026-07-15T01:38:16+05:30 -->
From audit verdicts (claude_notes/scratchpad/20260715/kira-hygiene-verdict.md section 1): (1) captured label -> knob: name it in config (e.g. create.here_label, empty disables) instead of invisible magic at core/create.go:19,116 + validate.go:21 system-label exemption follows the configured name; (2) kiracommit caps 5 subject numbers / 20 trailers (kiracommit.go:14-17) -> config knobs, batch users hit them silently; (3) link types relates/duplicate_of (datamodel/item.go:54-59) -> config vocab, default unchanged; (4) commit.mode=prompt silently degrades to no-commit when non-interactive (core/git.go:38-55) -> keep behavior, add errx hint + scaffold doc line; (5) drop the inert '## Comments' heading from the default draft body (core/draft.go:141-146) — implies parsing that never happens. DEFERRED with reasons, no ticket: item-types config (audit says bigger lift, defer), automation event vocab (speculative until a real 4th event exists). All knobs additive; frozen --json contract untouched except additive Explain/schema regs.
<!-- /kira:comment -->
