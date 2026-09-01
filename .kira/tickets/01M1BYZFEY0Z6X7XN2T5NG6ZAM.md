---
id: 01M1BYZFEY0Z6X7XN2T5NG6ZAM
number: KIRA-124
aliases: []
type: ticket
title: "termx.IsTerminal treats /dev/null as a TTY"
state: DONE
resolution: done
priority: P2
labels: []
epic: null
blocked_by: []
created: 2026-08-31T15:08:44+02:00
updated: 2026-09-01T10:25:58+02:00
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01M1BYZFQ0QPXST9065XQCZ10X author=Shivam-Shivanshu ts=2026-08-31T15:08:44+02:00 -->
termx/termx.go:14-17 returns Mode()&os.ModeCharDevice != 0. /dev/null is a character device, so 'kira </dev/null' or '>/dev/null' counts as interactive. This drives IsInteractive() (prompts, cli/prompt.go:25) and WriterIsTTY (auto-TUI launch, cli/tui.go:58).
Scenario: a cron job or CI step runs 'kira ... </dev/null >/dev/null' and kira decides it is interactive - launching the TUI or blocking on a confirm prompt nobody can answer.
Fix: use term.IsTerminal(f.Fd()) from charmbracelet/x/term, which the package already imports for Width.
<!-- /kira:comment -->
