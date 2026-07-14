---
id: 01KXH34JPQK0DNJHXR2WC5YHP2
number: TUI-4
aliases: []
type: ticket
subtype: bug
title: "clipx: fix tmux DCS wrapping; surface clipboard copy errors"
state: DONE
resolution: done
priority: P2
labels: []
epic: null
blocked_by: []
created: 2026-07-15T01:24:50+05:30
updated: 2026-07-15T03:04:59+05:30
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01KXH34JWNZ5AWMNEHMA1PMJ2Z author=Shivam-Shivanshu ts=2026-07-15T01:24:51+05:30 -->
internal/clipx/clipx.go:54 combines a literal-ESC prefix with full ESC-doubling of a sequence that itself starts with ESC — reference implementations (go-osc52, vim-oscyank) use one idiom, never both; strict parsers (nested muxers) may not recover. The test self-certifies: internal/clipx/test/clipx_test.go:109-116 strips the prefix then re-encodes with the identical recipe (plus a second circular comparison at :73 asserting term output equals clipx.OSC52 itself); only the plain form is pinned to literal bytes. tui/yank.go:22/:49 '_ = m.clip.Copy(text)' — a missing xclip is invisible. run.go:44 hands the same out to clipx.System and tea.NewProgram; Copy runs from Update handlers while the renderer flushes from its own goroutine (writer hazard). Untested: XWayland fallback, tool-failure wrapping, Term==nil; terminal write error appended unwrapped; externalTool threads three receiver fields as params.

Fix: drop the trailing ESC from the prefix ('\x1bPtmux;' + doubled seq + ST) and pin the tmux form to an exact literal byte string. Surface Copy errors in the yank status flash (or document silent best-effort). Note the writer hazard in the package doc; route emission through the program when feasible. Add the missing matrix cases; wrap the term error symmetrically; make externalTool a method.

Verify: literal-byte test for the tmux form; yank error-flash test with a failing external tool.

Files: internal/clipx/clipx.go, internal/clipx/test/clipx_test.go, internal/tui/yank.go, internal/tui/run.go
<!-- /kira:comment -->
