---
id: 01KXH96M0AYFGS4FSZX5HNYRSZ
number: TUI-6
aliases: []
type: ticket
subtype: task
title: "clipx follow-up: renderer write hazard and untested fallbacks (from TUI-4)"
state: TODO
priority: P3
labels: []
epic: null
blocked_by: []
created: 2026-07-15T03:10:49+05:30
updated: 2026-07-15T03:10:49+05:30
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01KXH9765YE1NRGNVWKXW1ECZN author=Shivam-Shivanshu ts=2026-07-15T03:11:07+05:30 -->
Follow-up to TUI-4 (DCS byte fix + error surfacing shipped; these were documented, not fixed).

- Writer hazard: clipx.Copy writes OSC52 directly to the same io.Writer bubbletea renders to, from Update handlers, while the renderer flushes from its own goroutine. Pre-existing; routing the copy through the program was found infeasible on bubbletea v1.3.10. Revisit if/when bubbletea exposes a safe write path.
- Untested: XWayland fallback, tool-failure wrapping, Term==nil; terminal write error appended unwrapped; externalTool threads three receiver fields as params (tidy to a method).
<!-- /kira:comment -->
