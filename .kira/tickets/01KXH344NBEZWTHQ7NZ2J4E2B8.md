---
id: 01KXH344NBEZWTHQ7NZ2J4E2B8
number: CORE-15
aliases: []
type: ticket
subtype: bug
title: "Two-tier config: user-tier UI warnings blamed on repo path; repo ui.editor silently dropped"
state: TODO
priority: P2
labels: []
epic: null
blocked_by: []
created: 2026-07-15T01:24:36+05:30
updated: 2026-07-15T01:24:36+05:30
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01KXH344VATN5RF5Z49W41DE79 author=Shivam-Shivanshu ts=2026-07-15T01:24:36+05:30 -->
internal/config/load.go:40 applies ignorer(warn, repo path) to UIWarnings(cfg.UI) where cfg.UI is the merged result the user tier can seed entirely — a user-config bogus theme slot/column prints '<repo>/.kira/config.yaml: ...; ignored'. load.go:69-73 saves cfg.UI.Editor around the repo unmarshal and restores it: the security why (TestRepoEditorIgnored) is invisible at the site and, unlike the mirror case (user.go:83 'repo-authoritative; ignored'), the repo author gets no signal.

Fix: run UIWarnings per tier — emit user-tier UI warnings through the user-file ignorer in readUserPrefs; assert the path prefix in the existing test. Extract the editor save/restore into a named helper and warn 'ui.editor is personal; set it in ~/.config/kira/config.yaml; ignored' when the repo document carries the key. Plumbing caveat: parseInto is shared by Parse()/Load() which have no warn writer — route the repo ui.editor warning only through LoadWithUser (or nil-safe writer); detecting that the repo document carries the key needs yaml-node inspection or post-unmarshal value comparison (latter misses repo-value==user-value).

Sweep-in: reword 'invalid RFC3339 date' to 'want YYYY-MM-DD' (clarity, also sibling copies internal/core/validate.go:81 and internal/doctor/check.go:102); drop the fileExists pre-check in readUserHooks (TOCTOU; readMapping shows the pattern); merge the two usertier test files onto one harness (duplicated minimalRepo fixtures).

Files: internal/config/load.go, internal/config/user.go, internal/core/validate.go, internal/doctor/check.go
<!-- /kira:comment -->
