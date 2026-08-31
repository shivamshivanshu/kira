---
id: 01M1BZFD1Z2XJ38Q8H19Z93NXF
number: KIRA-147
aliases: []
type: ticket
title: "webui: reject a bridge selection value that starts with a dash"
state: REVIEW
priority: P3
labels: []
epic: null
blocked_by: []
created: 2026-08-31T15:17:26+02:00
updated: 2026-08-31T16:12:18+02:00
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01M1BZFDA605NQ2AB4JK0Q4BWB author=Shivam-Shivanshu ts=2026-08-31T15:17:26+02:00 -->
From the pre-merge security review. webui/actions.go:214-222 commandArgs substitutes the form's 'selected' value for '.' verbatim, and only argv[0] is checked for a leading dash (cli/server.go:141).
Self-attack only today because every POST is CSRF-gated, but selected='--from-file=x' against a command line like 'comment . -m hi' yields [comment, --from-file=x, -m, hi], leaving the per-command flag blocklist in PersistentPreRunE as the only defence.
Fix: reject a selected value beginning with '-'; it can never be a valid item ref.
<!-- /kira:comment -->
