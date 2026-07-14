---
id: 01KXH342VKD61KFX5VP3A5GHEX
number: CORE-12
aliases: []
type: ticket
subtype: bug
title: "Exec contract: strings.Fields mangles quoted arguments in automation hooks and editor commands"
state: DONE
resolution: done
priority: P1
labels: []
epic: null
blocked_by: []
created: 2026-07-15T01:24:34+05:30
updated: 2026-07-15T02:47:11+05:30
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01KXH34318NZBGJCJDBJH8P3EQ author=Shivam-Shivanshu ts=2026-07-15T01:24:34+05:30 -->
Exec strings are split with strings.Fields in exactly two places: internal/automation/fire.go:81 (hook Run) and internal/editorx/editorx.go:33-40 (Command, feeding both Edit and View). 'run: notify-send "Item done"' executes with args ["Item, done"]; 'sh -c "kira sync && ..."' equally broken; EDITOR='"/opt/My Editor/bin/ed" --wait' splits into garbage (git/less solve this via sh -c). Config validation only checks non-empty; nothing documents argv-only semantics.

Fix: decide the contract once — exec via sh -c (safe: RecursionGuardEnv prevents kira-in-kira and the trust gate already treats run as arbitrary code), or a shared quote-aware splitter that rejects unbalanced quotes at config load; apply the same choice in editorx (git-style sh -c with path appended). If Windows is a target, prefer the quote-aware splitter or gate sh -c per platform.

Verify: runHook table tests driving a bytes.Buffer — quoted arg preserved, sleep+50ms timeout kill, false exit propagates, prefix on both streams, recursion-guard no-op (note: the recursion guard at fire.go:18 and envMirror at fire.go:39,62 live in Fire(), so those tests must drive Fire, not runHook alone). This half currently has zero coverage.

Files: internal/automation/fire.go, internal/editorx/editorx.go, internal/config/validate.go
<!-- /kira:comment -->
