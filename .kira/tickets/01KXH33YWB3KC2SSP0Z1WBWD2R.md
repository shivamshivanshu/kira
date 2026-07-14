---
id: 01KXH33YWB3KC2SSP0Z1WBWD2R
number: CLI-2
aliases: []
type: ticket
subtype: bug
title: "Gate auto-TUI on the command's writer; extract shared shouldAutoTUI"
state: DONE
resolution: done
priority: P1
labels: []
epic: null
blocked_by: []
created: 2026-07-15T01:24:30+05:30
updated: 2026-07-15T02:00:56+05:30
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01KXH33Z25Q3Q5NSECC1QMC10D author=Shivam-Shivanshu ts=2026-07-15T01:24:30+05:30 -->
Two auto-TUI gating defects, same mechanism — land together.

1) ':board' from inside the TUI launches a second tea.Program. cmdbar (tui/cmdbar.go:103) only blocks argv[0]=="tui" (wrong layer — any future auto-launching command has the same hole). board.go:50 decides shouldLaunchBoardTUI via cfg.UI.AutoTUI && !plain && !g.json && noFilters && termx.IsTerminal(os.Stdout) — os.Stdout IS still a TTY inside the running TUI and AutoTUI defaults true (config/defaults.go:72), so tui.Run starts inside a tea.Cmd goroutine: input stolen, alt-screen corruption, outer model blocked. boardWidth() (board.go:19-27) likewise reads os.Stdout/COLUMNS instead of the buffer it renders into. Fix: gate on writerIsTTY(cmd.OutOrStdout()) — move writerIsTTY from internal/tui/icons.go:52 into termx — and make boardWidth take the writer; with commandRunner's bytes.Buffer the gate falls back to RenderBoardPlain.

2) Bare 'kira' redirected to a file writes alt-screen ANSI. runTUI (cli/tui.go:31-49, root RunE cli/cli.go:75-77) checks only auto && !cfg.UI.AutoTUI; tui.Run defaults Output to os.Stdout with tea.WithAltScreen unconditionally (run.go:81). 'kira --json' should fall back to cmd.Help() (matching the auto_tui:false path, tui.go:46, and ui-auto-tui-off.txtar). Fix: one shared gate shouldAutoTUI(g, cfg) = cfg.UI.AutoTUI && !g.json && termx.IsTerminal(os.Stdout) used by both sites; board keeps its extra !plain && noFilters conjuncts; explicit 'kira tui' stays ungated (passes auto=false deliberately).

Verify: cli test commandRunner(g)([]string{"board"}) with AutoTUI=true returns plain text and does not hang; e2e txtar (non-tty by construction) asserting bare 'kira' prints help, not TUI bytes.

Files: internal/cli/board.go, internal/cli/tui.go, internal/tui/icons.go, internal/termx/termx.go, internal/tui/cmdbar.go, internal/tui/run.go
<!-- /kira:comment -->
