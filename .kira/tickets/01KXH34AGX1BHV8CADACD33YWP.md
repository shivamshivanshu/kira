---
id: 01KXH34AGX1BHV8CADACD33YWP
number: CORE-21
aliases: []
type: ticket
subtype: bug
title: "storage: require valid ULID filenames and direct-child ticket paths"
state: IN_PROGRESS
priority: P2
labels: []
epic: null
blocked_by: []
created: 2026-07-15T01:24:42+05:30
updated: 2026-07-16T16:13:12+05:30
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01KXH34APZEGMQPRX57ZFX967B author=Shivam-Shivanshu ts=2026-07-15T01:24:42+05:30 -->
internal/storage/ticketpath.go:12-14 checks only the .md suffix — a stray README.md flows through LoadAll (SkipNote on every command, or loads as an item), kiracommit/hookrun classification, and the index (bogus 'ULID'); id.ParseULID (ParseStrict) already exists. ticketpath.go:9 IsItemPath matches any depth while read.go:47-49 never recurses; treeish/loader.go:35 uses IsItemPath on ls-tree -r output, so .kira/tickets/sub/x.md is loaded under --at and counted by kira-commit but invisible live.

Fix: require the stem to be a valid 26-char ULID in isItemFilename/ULIDFromFilename, and require a direct child (path.Dir(rel) == TicketsPrefix) in IsItemPath — one change hardens LoadAll, IsItemPath, doctor, index, and treeish alignment. Caveat: doctor currently reports strays (cli/doctor.go:56 includes README.md, Lint runs codec.Parse on it, non-zero exit as schema errors); tightening ULIDFromFilename alone would make doctor's scan silently skip strays — pair with an explicit doctor 'stray file in tickets dir' finding (or keep doctor's scan loose).

Verify: table tests (notes.md, wrong-length, nested, dotfile); a doctor test asserting strays are still reported.

Files: internal/storage/ticketpath.go, internal/cli/doctor.go, internal/doctor/doctor.go
<!-- /kira:comment -->
