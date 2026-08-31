---
id: 01M1BYZFZ3F1WS67VRPPPVJW8X
number: KIRA-125
aliases: []
type: ticket
title: "kira config explain reports user-tier provenance for repo-set ui and workon values"
state: TODO
priority: P2
labels: []
epic: null
blocked_by: []
created: 2026-08-31T15:08:45+02:00
updated: 2026-08-31T15:08:45+02:00
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01M1BYZG71A3VEAF5GEE9MHV83 author=Shivam-Shivanshu ts=2026-08-31T15:08:45+02:00 -->
core/explain.go:38-47,243,252 - tierOf(differs, userTier=true) hardcodes the ui and workon sections as user-tier whenever they differ from the default. But config/set.go:27-30 lets 'kira config set ui.icons|ui.background|workon.*' write those keys into the REPO .kira/config.yaml, and config/load.go:34-43 has repo values override the user tier.
Scenario: kira config set ui.icons emoji (written to the repo file and committed) -> kira config explain tags the ui section 'user'. The provenance the command exists to report is wrong, and a teammate hunting for where a setting comes from is sent to the wrong file.
Fix: compute provenance during LoadWithUser by recording which tier actually supplied each section, instead of inferring it from the section's identity.
<!-- /kira:comment -->
