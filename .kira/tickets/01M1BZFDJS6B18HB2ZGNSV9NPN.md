---
id: 01M1BZFDJS6B18HB2ZGNSV9NPN
number: KIRA-148
aliases: []
type: ticket
title: "gitx: pass --end-of-options before user-supplied refs"
state: DONE
priority: P3
labels: []
epic: null
blocked_by: []
created: 2026-08-31T15:17:27+02:00
updated: 2026-08-31T16:12:19+02:00
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01M1BZFDTRTZ8RFKAZP3HMFWAZ author=Shivam-Shivanshu ts=2026-08-31T15:17:27+02:00 -->
From the pre-merge security review, and it applies beyond the webui. gitx/treeish.go:16 runs 'rev-parse --verify --quiet <ref>^{commit}' with no --end-of-options, so a ref beginning with '-' is parsed as a git flag. Reachable through diff --since, show --at and changes --since, including via the dashboard command bar.
gitx/show.go:19 already does this correctly - ShowPatch passes --end-of-options - so this is drift between two sibling call sites rather than a missing idea. Exploitation is unlikely given the ^{commit} suffix and rev-parse's flag set, and every path is user-initiated, hence P3.
Fix: add --end-of-options before the ref, matching ShowPatch. Sweep the other gitx rev and branch wrappers at the same time.
<!-- /kira:comment -->
