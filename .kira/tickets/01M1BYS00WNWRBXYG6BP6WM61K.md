---
id: 01M1BYS00WNWRBXYG6BP6WM61K
number: KIRA-114
aliases: []
type: ticket
title: "kira discover ignores --non-interactive; :discover from the TUI command bar hijacks or hangs the UI"
state: DONE
priority: P1
labels: []
epic: null
blocked_by: []
created: 2026-08-31T15:05:12+02:00
updated: 2026-08-31T16:12:05+02:00
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01M1BYS08W6W7VJJD2PXBQT3MR author=Shivam-Shivanshu ts=2026-08-31T15:05:12+02:00 -->
From the DX-1 cli blind review.

internal/cli/discover.go:31-57 RunE never consults g.nonInteractive or g.prompter().Interactive(). Every sibling with an interactive step guards it: create.go:119 pickBoardIfAmbiguous checks Interactive(), edit.go:59 and comment.go:20 reject $EDITOR under nonInteractive. discover is the drifted twin. The TUI command bar bridges through commandRunner (cli/tui.go:72-83) which forces --non-interactive and only blocks argv[0]=='tui' (tui/cmdbar.go:103).

Scenario: in the TUI type ':discover'. With fzf installed, fzfx.Pick execs fzf with Stderr=os.Stderr, drawing its finder over the alt-screen and fighting bubbletea for the tty. Without fzf, pickNumbered prints into the captured buffer (invisible) then blocks on fmt.Fscan(os.Stdin) against bubbletea's input reader - the queued command never completes, busy never clears, and the TUI is stuck on 'finishing in-flight change' until double ctrl+c. Scripted kira --non-interactive discover misbehaves the same way.

Fix: reject at the top of RunE when g.nonInteractive, mirroring the edit/comment errors. That covers the command-bar bridge for free.
<!-- /kira:comment -->
