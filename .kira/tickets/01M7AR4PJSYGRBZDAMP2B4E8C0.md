---
id: 01M7AR4PJSYGRBZDAMP2B4E8C0
number: CORE-71
aliases: []
type: ticket
subtype: bug
title: "config explain says resolutions_dropped affects epic progress; it does not"
state: DONE
resolution: done
priority: P3
labels: []
parent: null
blocked_by: []
created: 2026-11-13T20:54:12+05:30
updated: 2026-11-13T21:41:39+05:30
---

## Description

`kira config explain` prints

    resolutions_dropped: dropped (a done-category resolution treated as a non-completion outcome for epic progress)

(`internal/core/explain.go:125-126`). Epic progress
(`internal/core/progress.go`, `allContainerProgress`) counts every
done-state child as done, dropped included; only `kira stats` treats
`dropped` separately. `resolutions_dropped` is also no longer a config key.

## Acceptance criteria

- The explain line describes what `dropped` actually affects (stats
  completion, cycle/lead time, throughput), or is removed.
- Docs: drop the entry from 09 Open bugs and the pointer in
  `config explain` (setup-and-schema-commands.md).
