---
id: 01KXH34DVYW2WJX24HJJXPFK3B
number: CLI-6
aliases: []
type: ticket
subtype: task
title: "Lazy create subcommand registration — avoid eager Discover+Load per root build"
state: TODO
priority: P2
labels: []
epic: null
blocked_by: []
created: 2026-07-15T01:24:46+05:30
updated: 2026-07-15T01:24:46+05:30
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01KXH34E1RC7N2HZ7HSJQQH90D author=Shivam-Shivanshu ts=2026-07-15T01:24:46+05:30 -->
newRootCmd unconditionally calls newCreateCmd -> createTypes (internal/cli/create.go:30-45): core.Discover(chdirArg()) (walk-up + EvalSymlinks) and config.Load before flags are parsed, purely to name create subcommands; RunE then re-discovers via openStore (cli.go:126, which loads via config.LoadWithUser — same store discovery + full config parse twice, though the loaders differ slightly). commandRunner rebuilds the whole root per command-bar submission; the prepare-commit-msg hook pays it per commit. createTypes falls back to {ticket, epic} on any failure, so this is cost-only, never correctness.

Fix: make the list lazy — a single 'create <type>' positional validated at RunE against cfg.Workflows with a ValidArgsFunction for completion; or peek os.Args for create/help/__complete before Discover+Load; or cache (store, cfg) into globalFlags for openStore reuse. A RunE-time lazy fix also lets you delete chdirArg() entirely (parsed g.chdir is available there — coordinate with the chdirArg bugfix ticket). Drop the unused g parameter.

Verify: spawn/latency check via the perf spawn tests; completion of create subcommands still works.

Files: internal/cli/create.go, internal/cli/cli.go
Depends on: chdirArg misparses -Cpath and '--'; bulk outcome contract untested
<!-- /kira:comment -->
