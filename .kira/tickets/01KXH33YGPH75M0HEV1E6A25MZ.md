---
id: 01KXH33YGPH75M0HEV1E6A25MZ
number: CLI-1
aliases: []
type: ticket
subtype: bug
title: "Command bar: make bridged CLI commands non-interactive"
state: REVIEW
priority: P1
labels: []
epic: null
blocked_by: []
created: 2026-07-15T01:24:30+05:30
updated: 2026-07-15T02:00:55+05:30
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01KXH33YPGNP8WGHS209Y5MXP7 author=Shivam-Shivanshu ts=2026-07-15T01:24:30+05:30 -->
TUI command bar bridges to the normal CLI root (commandRunner, internal/cli/tui.go:58) which runs with terminalPrompter{} (cli/cli.go:127, cli/prompt.go). termx.IsInteractive() is true inside the TUI, so interactive paths block on stdin while tea holds the terminal raw:
- core CommitPrompt reads os.Stdin (core/git.go:41-46).
- ':create foo' hits the interactive-editor branch core/create.go:198-216 (runEditor at :204) with NoEdit=false, spawning an editor on the real terminal outside tea.Exec (contrast tui/editor.go:27-36).
- create.go:159 gates the board picker only on g.json || !Interactive(); create.go:167 ReadLineDefault blocks. The picker list goes to commandRunner's invisible bytes.Buffer via cmd.ErrOrStderr(); the ReadLineDefault prompt itself writes directly to os.Stderr (termx.go), garbling the alt-screen. Either way the TUI appears frozen.

Fix: hidden persistent --non-interactive flag set by commandRunner (alongside --no-color) that (a) injects core's silentPrompter instead of terminalPrompter, (b) implies --no-edit for create/edit (or errx hint 'editor unavailable in command bar'), (c) gates pickBoardIfAmbiguous.

Verify: bridge tests in internal/cli asserting ':create' without --no-edit errors rather than spawning an editor, and that argv like {board, create X, tui} return non-interactive output. Existing tui_test.go only covers a happy-path move.

Files: internal/cli/tui.go, internal/cli/cli.go, internal/cli/create.go, internal/cli/prompt.go
<!-- /kira:comment -->
