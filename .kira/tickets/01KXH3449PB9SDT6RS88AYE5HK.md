---
id: 01KXH3449PB9SDT6RS88AYE5HK
number: CORE-14
aliases: []
type: ticket
subtype: bug
title: "config yaml writers: SetScalar silent failures and comment-aware splice fixes"
state: IN_PROGRESS
priority: P1
labels: []
epic: null
blocked_by: []
created: 2026-07-15T01:24:36+05:30
updated: 2026-07-15T02:22:44+05:30
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01KXH344FF03J85KX9NAGJR3TS author=Shivam-Shivanshu ts=2026-07-15T01:24:36+05:30 -->
SetScalar's only guard (internal/config/set.go:76-80) is Parse-succeeds, not round-trip (sibling writers all re-parse-and-compare: boardwrite.go:40, sprintwrite.go:25, labelwrite.go:57). Confirmed silent failures: (1) 'git: {}' — descend/insertUnder splices a top-level landed_ref: line yaml ignores; success reported, value unset. (2) renderToken passes kindLiteral raw so 'manual # x' silently becomes 'manual'. (3) UpdateBoard's LastIndexByte('}') deletes a comment containing '}' undetected (boardwrite.go:135). Error-not-corruption UX: bare 'git:' appends a duplicate block (raw yaml error); appendToFlowList's ']'-in-comment (labelwrite.go:65-66) and openBlockListUnderKey's '[ ]' (sprintwrite.go:73) produce spurious failures caught by round-trip checks; AddBoard with no version: line tells the user to hand-edit (boardwrite.go:159 -> validate.go:110). bumpVersionToBoards hand-splices where replaceScalarLine exists (:165) — quoted-version corruption is unreachable (Version is int, initial Parse rejects), reuse anyway. Generic helpers scattered with three drifting single-line-guard variants.

Fix: after the final Parse in SetScalar, re-descend and require the leaf scalar to equal the requested token; handle/reject non-block-mapping intermediates with a targeted error; reject '#' in kindLiteral. Fix the splice edges (depth/quote-aware bracket+brace scans, replaceScalarLine reuse, auto-insert 'version: 2', regex bracket strip). Extract shared node/line helpers + flowScalar/singleLineScalar into one yamlsplice.go with a single one-line guard.

Verify: table tests for each edge plus the two UpdateBoard refusal paths.

Files: internal/config/set.go, internal/config/labelwrite.go, internal/config/sprintwrite.go, internal/config/boardwrite.go
<!-- /kira:comment -->
