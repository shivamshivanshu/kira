---
id: 01M1GHBVSX9YS4C1KAPS2CNR9E
number: KIRA-165
aliases: []
type: ticket
title: "kira list and kira board have no --board flag, and board is hardwired to ticket"
state: TODO
priority: P1
labels: []
epic: null
blocked_by: []
created: 2026-09-02T09:47:02+02:00
updated: 2026-09-02T09:47:02+02:00
---

## Description

## Acceptance criteria

<!-- kira:comment id=01M1GHCR55HQRAPQRXX6ECJQZJ author=Shivam-Shivanshu ts=2026-09-02T09:47:31+02:00 -->
From the surface-coherence review.

  $ kira list --board OPS
  kira: unknown flag: --board
  $ kira list --query board=OPS     # works
  $ kira list --field board=OPS     # works

core.ListOpts.Board exists (core/list.go:33) and is filtered (core/list.go:216). Only the cobra adapter is missing, so --board is a create-only flag: you can put a ticket on a board and then have no ergonomic way to look at that board.

Worse on kira board: cli/board.go:56 constructs core.BoardOpts{Epic, Owner, Label, Query, Filter, At} while core.BoardOpts (core/board.go:27-39) declares Type, Epic, State, Owner, Priority, Label, Board, Sprint, Query, Filter, At. Five implemented filters are unreachable from the CLI, and Type is the consequential one — kira board is hardwired to ticket, so a user-defined entity type can never have a board.

The reviewer rated this the highest value-per-hour item it found: roughly 4-6h to wire the flags, completers and txtar coverage, against a whole concept that is currently half-exposed.
<!-- /kira:comment -->
