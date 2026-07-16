---
id: 01KXH96KSDZ6JZ13K2W7WKDZ3N
number: CORE-37
aliases: []
type: ticket
subtype: bug
title: "yamlsplice latent edge cases: split-line empty flow list and rune-vs-byte column index"
state: DONE
priority: P3
labels: []
epic: null
blocked_by: []
created: 2026-07-15T03:10:49+05:30
updated: 2026-07-16T19:51:53+05:30
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01KXH9760021C89SSSZEJK9WSA author=Shivam-Shivanshu ts=2026-07-15T03:11:07+05:30 -->
Surfaced by the CORE-14 blind review (pre-existing, not introduced by the consolidation).

1. Split-line empty flow list: openBlockListUnderKey (internal/config/yamlsplice.go) only strips the "[]" when val.Line == key.Line. A config with the empty list on the next line, e.g.

    sprints:
      []

leaves the bracket line orphaned and produces unparseable YAML (yaml: could not find expected ":"). AppendSprint/AddBoard re-parse and bail before writing, so it surfaces as a confusing hard failure on legitimate input, not silent corruption. Only same-line "[ ]" is tested (TestAppendSprintSpacedEmptyFlowList).

2. rune-vs-byte column index: yaml.v3 Node.Column is a rune count but is used as a raw byte index in replaceScalarLine, appendToFlowList, openBlockListUnderKey, and UpdateBoards brace locator. Every current call site has fixed-ASCII preceding same-line content, so it is not reachable today, but a non-ASCII board/sprint/label name sharing a line with a splice target would mis-slice. Add a decode-aware offset plus a non-ASCII round-trip test.
<!-- /kira:comment -->
