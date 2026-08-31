---
id: 01M1BYZHF4V3EWXEFS73BCMXPY
number: KIRA-128
aliases: []
type: ticket
title: "showfmt branch form re-implements workon slug rules and drifts from the real branch name"
state: REVIEW
priority: P2
labels: []
epic: null
blocked_by: []
created: 2026-08-31T15:08:46+02:00
updated: 2026-08-31T16:12:15+02:00
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01M1BYZHPRWFD6DM8RS4NR9YGJ author=Shivam-Shivanshu ts=2026-08-31T15:08:46+02:00 -->
showfmt.slugify (showfmt/showfmt.go:88-101) is a byte-for-byte re-implementation of workon.normalize (workon/workon.go:25-40) with the separator hardcoded to '-', and showfmt.branch hardcodes the number-title shape. workon renders the real branch from workon.branch_pattern plus casing config.
Scenario: repo config sets casing snake or a custom branch_pattern -> kira show --format branch (and the TUI yank picker) emits a branch name that does not match the branch kira workon actually creates, so yank-then-checkout fails.
Fix: have showfmt's branch form delegate to workon's renderer with the loaded config, or delete the private slugifier in favour of workon's.
<!-- /kira:comment -->
