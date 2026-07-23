---
id: 01KXH34GTRYSDYT1Z84YYMMQCJ
number: CLI-7
aliases: []
type: ticket
subtype: task
title: "CLI consistency batch: --quiet, crash-vs-json ordering, tree columns, ShortSHA, discover output, storeActionRunE"
state: REVIEW
priority: P2
labels: []
epic: null
blocked_by: []
created: 2026-07-15T01:24:49+05:30
updated: 2026-07-23T14:26:55+05:30
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01KXH34H0Y82R5NN7RVHC3TJ8T author=Shivam-Shivanshu ts=2026-07-15T01:24:49+05:30 -->
- cli.go:83 declares --quiet persistent; only index.go:28 consumes it — route human output through a g.quiet-aware helper or demote --quiet to index-local with fixed help.
- renderError (cli.go:33-36) returns ExitCrash before the jsonMode branch — move the crash branch below jsonMode (hint from CrashError.LogPath).
- tree.go:82 hardcodes DefaultListColumns and tab-count padding while flat list honors cfg.UI.List.Columns — pass resolveColumns into renderTreeGroups; derive header padding from len(cols).
- SHA truncation implemented three times at two widths: cli/log.go:49 [:8] vs tui/detailfull.go:168 [:7] vs commit.go:36 inline 7 — one gitx.ShortSHA.
- complete.go:58 completeStatic({ticket,epic}) though types are workflow-driven — completeVocab over cfg.Workflows keys; complete.go:144-151 findCached duplicates core's unexported itemMatchesAny predicate (single-ref variant) — export core.RefMatches.
- discover.go:31-54 never consults g.json (config.go:22-30 same); dispatchAction show drops StderrNotes/Skew; edit prints nothing; the no-fzf fallback prints candidates then exits 0 having performed no action; discover.go:90-92 concatenates os.Executable() unquoted into fzf's $SHELL -c preview — errx.User for --json (or typed shape + golden), share newShowCmd's output block, print editLine, error or numeric-pick when fzf is absent, shell-quote the preview path.
- storeActionRunE (board.go:75-91) used by 5 subcommands while sprint/label/config/commit/blame/log/now/stats/tree/changes/index/comment/automation/workon repeat the identical 10-12 line body — move to cli.go and adopt where RunE maps 1:1.

Verify: existing cli suites + contract goldens green; a crash-in-json test asserting JSON error shape ordering.

Files: internal/cli/cli.go, internal/cli/tree.go, internal/cli/log.go, internal/cli/commit.go, internal/cli/complete.go, internal/cli/discover.go, internal/cli/config.go, internal/cli/board.go, internal/gitx/gitx.go, internal/tui/detailfull.go, internal/core
<!-- /kira:comment -->
