---
id: 01M1BZ1J6C6NN5HKMF64R9WQ8M
number: KIRA-137
aliases: []
type: ticket
title: "Small correctness and clarity nits across core, cli and tui"
state: DONE
resolution: done
priority: P3
labels: []
epic: null
blocked_by: []
created: 2026-08-31T15:09:52+02:00
updated: 2026-09-01T10:25:59+02:00
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01M1BZ1JE39NSAS4033BMZ158D author=Shivam-Shivanshu ts=2026-08-31T15:09:53+02:00 -->
Grouped low-severity items from the DX-1 blind reviews.
core: Batch.LabelSet has no blank-name guard though LabelCreate does (label.go:16-19 vs :85-93), so 'label add ""' can commit an empty label; closes.go:62-71 reopenedSince treats any field edit after the closing commit as a reopen and silently suppresses the auto-close (rename it touchedSince or compare against the last state event); sync.go:122 hardcodes '.kira' where kiracommit.go:29 uses storage.DirName; sprint.go:166 returns errx.User for an active-pointer Remove failure where every sibling IO failure is errx.Env.
cli/tui: cmdbar.go:48-49 has an unreachable ctrl+c branch (app.go:180-182 intercepts first); app.go:95-99 guards cfg != nil though run.go:42 and two screens deref it unguarded; cli.go:154-161 stripGlobalPrefix locates the subcommand with slices.Index over the whole argv, so '-C find find pat' matches the flag VALUE and silently drops the chdir; treescreen.go:51 dropMissing tests membership in the FILTERED item set, permanently purging jump history for items merely hidden by the active filter.
<!-- /kira:comment -->
