---
id: 01M1BZFE2WCSCAYXG9S5P6PEB2
number: KIRA-149
aliases: []
type: ticket
title: "webui: a hung network git operation wedges the whole dashboard"
state: TODO
priority: P3
labels: []
epic: null
blocked_by: []
created: 2026-08-31T15:17:27+02:00
updated: 2026-08-31T15:17:27+02:00
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01M1BZFEBSZW27DXQ7Q83K8P33 author=Shivam-Shivanshu ts=2026-08-31T15:17:27+02:00 -->
From the pre-merge security review. webui/actions.go:20-21 holds a single mutex across mutate -> RunCommand -> root.Execute, cli/server.go:96 deliberately leaves WriteTimeout unbounded, and bridge commands have no timeout.
Availability only, and local only: a sync against a stalled remote blocks every subsequent request, GETs included, indefinitely.
Fix: wrap bridge and sync execution in a generous context.WithTimeout, or accept it and document the behaviour.
<!-- /kira:comment -->
