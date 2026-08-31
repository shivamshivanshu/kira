---
id: 01M1BZ1G5JTQDG3YCR959168HW
number: KIRA-133
aliases: []
type: ticket
title: "Query and CLI flag drift: changes --since, --json universality, completion gaps"
state: IN_PROGRESS
priority: P3
labels: []
epic: null
blocked_by: []
created: 2026-08-31T15:09:50+02:00
updated: 2026-08-31T23:51:24+02:00
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01M1BZ1GDEBVNTBE08FEWSMAPH author=Shivam-Shivanshu ts=2026-08-31T15:09:51+02:00 -->
Grouped from the cli contextual DX-1 review.
(a) core/diff.go:33 resolves --since via resolveDateOrTreeish; core/changes.go:13 uses bare ResolveTreeish. 'kira diff --since 2026-08-01' works, 'kira changes --since 2026-08-01' fails with raw git noise. Help strings encode the drift (cli/diff.go:52 vs cli/changes.go:39). cli/board.go:72 documents --at as 'a git ref' though it accepts dates.
(b) README.md:19-21 says 'Every command supports --json'. discover and bare config error; 'kira tui --json' silently launches the full-screen TUI (cli/tui.go:34-54 never consults g.json); 'kira --json' on a non-TTY prints human help to stdout, exit 0. README also omits 'board unarchive'.
(c) Completion gaps (cli/complete.go:20-80): no positionals for sprint activate|close, board rename|archive|unarchive|move, board [epic-id], stats [epic-id], tree [id]; no --field key completion (now that schema fields are settable, and the design doc asks for it); and the name-keyed 'type' entry binds to create's --type, which is a SUBTYPE alias, so it suggests ticket|epic where bug|story are valid.
Fix: route Changes through resolveDateOrTreeish; make tui and the auto-TUI path reject --json like discover does; amend README; add the missing completers and bind create --type to the subtype vocab.
<!-- /kira:comment -->
