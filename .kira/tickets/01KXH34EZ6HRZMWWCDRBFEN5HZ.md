---
id: 01KXH34EZ6HRZMWWCDRBFEN5HZ
number: CORE-30
aliases: []
type: ticket
subtype: task
title: "editorx: precedence order, parsed Editor type, error contract"
state: TODO
priority: P2
labels: []
epic: null
blocked_by: []
created: 2026-07-15T01:24:47+05:30
updated: 2026-07-15T01:24:47+05:30
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01KXH34F52BDNFRJH4KPQJMZ5J author=Shivam-Shivanshu ts=2026-07-15T01:24:47+05:30 -->
internal/editorx/editorx.go:34 orders {configured, $EDITOR, $VISUAL} — conventional precedence is configured > $VISUAL > $EDITOR. :54 wraps with %v not %w (lost capability, not live bug — draft.go:121 flattens to errx.User anyway; pays off only if callers start distinguishing :cq/nonzero-exit from missing binary via exec.ErrNotFound). :39 bakes 'set ui.editor or $EDITOR' remediation into the error while core/draft.go:104 rewraps with its own WithHint — CLI users see remediation twice, TUI callers get no errx hint. :42-49 vs :61-70 duplicate the same four-step construction. Triple parse on the TUI edit path: tui/editor.go:28 Command() precheck, store.Edit -> runEditor runs Command() again at draft.go:103, editorx.Edit parses a third time at :43.

Fix: swap precedence to configured > $VISUAL > $EDITOR; %w wrapping (special-case exec.ErrNotFound); shrink the error to 'no editor configured' with remediation in callers (add the TUI hint); make Command return a parsed Editor type with Edit/View methods sharing one builder (kills the triple parse).

Verify: stub-script table test for Edit — path appended, flags preserved, error prefix on nonzero exit (editorx_test currently covers only Command precedence and View assembly).

Depends on the exec-contract ticket: the strings.Fields split at :33-40 is decided there; this refactor builds on whichever splitter lands.

Files: internal/editorx/editorx.go, internal/core/draft.go, internal/tui/editor.go
Depends on: Exec contract: strings.Fields mangles quoted arguments in automation hooks and editor commands
<!-- /kira:comment -->
