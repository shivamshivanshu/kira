---
id: 01KXH34DFY86KFPNCRYPMJAQFY
number: KIRA-104
aliases: []
type: ticket
subtype: task
title: "errx sweep: config/query/cli error idioms, Env classification, Nearest case-folding"
state: REVIEW
priority: P2
labels: []
epic: null
blocked_by: []
created: 2026-07-15T01:24:45+05:30
updated: 2026-07-15T19:51:08+05:30
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01KXH34DNTHH2SJP1XA18AB2F6 author=Shivam-Shivanshu ts=2026-07-15T01:24:45+05:30 -->
Cross-cutting error-idiom cleanup:
- config validate/load/writers return only fmt.Errorf (hints crammed inline, e.g. validate.go:110); query.Error's Pos is flattened by core/list.go:35 errx.User("%v"); cli/cli.go:39-41 extracts hints only from *errx.Error. Fix: convert config errors to errx.User+WithHint; give query.Error an errx adapter (wrap %w in core).
- errx.Env is the convention for env faults yet mkdir/write/SQLite errors use errx.User at ~8 core sites, ~30 index sites, and config/userinit.go:29 — 'kira index' exits 1 not 3 on a read-only FS. Sweep IO/env wraps to errx.Env in core/index/config.
- Bare fmt.Errorf at cli show.go:24, changes.go:20, discover.go:34, edit.go:43, find.go:28, link.go:52, sync.go:22; validate.go:43 discards the ResolveItemFile error behind a misleading message. Replace with errx.User (+hints); propagate the underlying error in validate.
- errx.go:57 hardcodes 'invalid item' (wrongly singular for hookrun's multi-file case) and duplicates ParseError.Error()'s format — parameterize the prefix (one join helper shared with ParseError).
- suggest.go: distance('done','DONE')=4 so wrong-case typos never hint; threshold 2 makes 2-char inputs suggest anything; the exact-match branch is dead. Fold case in Nearest (return original casing), scale threshold by input length, delete the dead branch, hoist rune conversion with a length prune.

Verify: add errx_test.go covering Invalid hint extraction, WithHint copy semantics, Unwrap traversal (suggest_test.go already covers Nearest/editDistance — only errx.go lacks coverage); an exit-code test that index failures on read-only FS exit 3.

Files: internal/errx/errx.go, internal/errx/suggest.go, internal/config/validate.go, internal/config/userinit.go, internal/core/list.go, internal/cli/show.go, internal/cli/changes.go, internal/cli/discover.go, internal/cli/edit.go, internal/cli/find.go, internal/cli/link.go, internal/cli/sync.go, internal/cli/validate.go, internal/index
<!-- /kira:comment -->
