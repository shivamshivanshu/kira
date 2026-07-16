---
id: 01KXH3491PATYJK2VTZP07AZCZ
number: CLI-5
aliases: []
type: ticket
subtype: bug
title: "chdirArg misparses -Cpath and '--'; bulk outcome contract untested"
state: IN_PROGRESS
priority: P1
labels: []
epic: null
blocked_by: []
created: 2026-07-15T01:24:41+05:30
updated: 2026-07-16T11:53:15+05:30
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01KXH3497HA6P38SSC2HDJRMP4 author=Shivam-Shivanshu ts=2026-07-15T01:24:41+05:30 -->
chdirArg (internal/cli/create.go:47-62) matches -C x/--C x/-C=x/--C=x but not '-C/other/repo' (pflag-accepted for the registered StringVarP), returning "" — createTypes then discovers from cwd or falls back to {ticket, epic}, silently registering the wrong workflow types for the target repo. It also hijacks '-C' tokens that are other flags' values or appear after '--'. Fix: add the attached-shorthand case (strings.HasPrefix(a, "-C") -> a[2:]) ordered AFTER (or excluding) the exact '-C' and '-C=' cases it also matches, and stop at '--' — or better, pre-parse with a throwaway pflag.FlagSet (a plain string scan also misses grouped bool-shorthand forms like '-xC path').

Separately: runBulk (cli/bulk.go:11-38) emits []BulkOutcome JSON, per-id stderr, and the 'N of M items failed' exit-code contract shared by edit/move/assign/label — the contract table has only single-id cases and grep for BulkOutcome/'items failed' across tests/ returns nothing.

Verify: table unit tests for chdirArg over all arg forms and for runBulk (all-succeed/mixed/all-fail); contract case 'move KIRA-1 KIRA-2 IN_PROGRESS --json' plus one with a failing id (golden outcomes array, exit 1, per-id stderr).

Files: internal/cli/create.go, internal/cli/bulk.go, tests/contract/contract_test.go
<!-- /kira:comment -->
