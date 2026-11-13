---
id: 01M7AR4PD1M0FRQPK7P07DKFRQ
number: CORE-70
aliases: []
type: ticket
subtype: bug
title: "Config-file mutations that change nothing fail with a raw git commit error"
state: TODO
priority: P2
labels: []
parent: null
blocked_by: []
created: 2026-11-13T20:54:12+05:30
updated: 2026-11-13T20:54:12+05:30
---

## Description

Under `commit.mode: auto`, a mutation of `.kira/config.yaml` that leaves the
file unchanged still calls `git commit`, which fails with nothing staged:

    $ kira config set commit.mode auto        # already auto
    kira: git commit -m config set commit.mode -- .kira/config.yaml: exit status 1
    $ kira init --key ACME --force            # store already as init writes it
    kira: git commit -m kira: init -- .kira .gitattributes: exit status 1
    $ kira board unarchive ACME               # board not archived
    kira: git commit -m board unarchive ACME -- .kira/config.yaml: exit status 1

All exit 1. Item mutations short-circuit with `<NUMBER>: no changes`; the
config writers do not. `Store.finalize` (`internal/core/git.go`) commits
unconditionally.

## Acceptance criteria

- A config-file mutation that changes nothing reports a no-op and exits 0
  (or a user error naming the no-op), never a raw git error.
- Docs: drop the entry from 09 Open bugs and the pointer in
  `config set` (setup-and-schema-commands.md).
