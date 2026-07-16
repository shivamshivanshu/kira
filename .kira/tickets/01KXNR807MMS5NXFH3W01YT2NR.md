---
id: 01KXNR807MMS5NXFH3W01YT2NR
number: DX-11
aliases: []
type: ticket
subtype: task
title: "golangci-lint: doc-comment enforcement disabled and 130 findings still unfixed"
state: REVIEW
priority: P3
labels: []
epic: null
blocked_by: []
created: 2026-07-16T20:50:40+05:30
updated: 2026-07-16T21:09:50+05:30
---

## Description

golangci-lint was broken in CI for a while (v1.64.8 can't load a config
targeting go.mod's `go 1.25.0` -- Go 1.25 support only landed in golangci-lint
v2.4.0). Fixing the version (bumped to v2.12.2 + action v9) let the linter
actually run for the first time in a long time, and it surfaced 674 real
findings repo-wide -- golangci-lint's default `--max-issues-per-linter=50`
was silently truncating the reported count on every prior run, so this had
been invisible.

Of the 674:
- 555 were `revive`'s `exported`/`package-comments` rules (every exported
  symbol across the codebase needs a Go doc comment). This is now disabled
  in `.golangci.yml` (`enable-default-rules: true` + `exported`/
  `package-comments` set to `disabled: true`) rather than writing ~555
  comments in one pass -- see "how to fix" below for the real options.
- 119 were `errcheck` (mostly unchecked `Close`/`Rollback`/`Fprint*`/`Remove`
  calls) and were fixed across 6 dispatched batches covering internal/id,
  internal/config, internal/cli, internal/core+gitx+automation, internal/index,
  and internal/storage+testutil+e2e+misc.
- The remaining ~130 findings (119 errcheck + 11 misc revive: var-naming,
  unused-parameter, redefines-builtin-id) are still unfixed, in packages that
  were never in scope for any of those 6 batches -- confirmed via
  `golangci-lint run --max-issues-per-linter=0 --max-same-issues=0 ./...`.
  CI's `lint` job will still fail on these until they're fixed.

## Acceptance criteria

- [ ] Fix the remaining ~130 findings (`errcheck` + misc `revive`) so
      `golangci-lint run ./...` is clean repo-wide.
- [ ] Decide the doc-comment question and act on it -- options:
  1. Re-enable `exported`/`package-comments` and write the ~555 doc
     comments (biggest lift, but matches Go/godoc convention and every
     other repo linter rule already enforced here).
  2. Leave it disabled permanently and remove the "for now" framing from
     the `.golangci.yml` comment/commit history (accept that internal/-only
     packages don't need godoc-quality comments since nothing outside this
     module can import them).
  3. Re-enable only for specific higher-traffic/most-reused packages
     (e.g. internal/core, internal/datamodel) via a per-path revive
     exclusion, splitting the difference.
- [ ] Re-run `golangci-lint run --max-issues-per-linter=0
      --max-same-issues=0 ./...` after any config change to confirm the
      *true* count, not the capped default -- this session's original
      108-finding estimate was itself wrong because of the cap, and cost a
      round of re-scoping once discovered.

## Comments

<!-- kira:comment id=01KXNSAWDGP80ENNP68J3SJ0RS author=Shivam-Shivanshu ts=2026-07-16T21:09:43+05:30 -->
All 3 quick-win batches landed and verified: internal/cli (85 errcheck findings across 22 files), automation/index/tui/core (25 findings: errcheck + unused-parameter + var-naming), tests/contract+integration+perf (20 errcheck + 1 redefines-builtin-id). golangci-lint run ./... now reports 0 issues repo-wide (confirmed with --max-issues-per-linter=0 --max-same-issues=0 and a cleared cache too). Full test suite green (1285 tests). Remaining open item: the doc-comment policy decision (3 options in the description) is still unresolved.
<!-- /kira:comment -->

<!-- kira:comment id=01KXNSG9XXE56TZBNWJQVNZQZ3 author=Shivam-Shivanshu ts=2026-07-16T21:12:41+05:30 -->
Decision made: leave exported/package-comments disabled permanently (option 2). Rationale: everything here lives under internal/, so no other module can ever import these packages -- the usual payoff of Go doc comments (public API discoverability via go doc/godoc/pkg.go.dev) doesn't apply. Both acceptance criteria are now satisfied -- closing.
<!-- /kira:comment -->
