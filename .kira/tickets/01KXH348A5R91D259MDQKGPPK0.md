---
id: 01KXH348A5R91D259MDQKGPPK0
number: DX-4
aliases: []
type: ticket
subtype: task
title: "Contract tests: cover the ~20 uncontracted JSON commands and an exit-2 error shape"
state: TODO
priority: P2
labels: []
epic: null
blocked_by: []
created: 2026-07-15T01:24:40+05:30
updated: 2026-07-15T01:24:40+05:30
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01KXH348FY8EFF9E6FJW69588K author=Shivam-Shivanshu ts=2026-07-15T01:24:40+05:30 -->
emitJSON appears in ~30 files in internal/cli but the cases table (tests/contract/contract_test.go:288-327) covers ~20; the golden directory has 42 entries while schema $defs has 66 — BoardResult, BulkOutcome, CommitResult, ConfigInit/SetResult, FilterListResult, HooksStatus/UninstallResult, LabelCreate/ListResult, MergeResult, NowResult, ResolveResult, VersionResult, Completion have no golden. e2e label.txtar asserts label --json via regex — contract-bearing yet unfrozen and blind to renames. TestJSONErrors covers exits 1 and 3 only; errx defines ExitConflict=2 — only exit 2 lacks a contracted JSON error shape (exit 4 is by design: renderError returns ExitCrash on tui.CrashError before the JSON encoder, and its code is pinned by tests/e2e/crash_exit_test.go). board is only exercised plain.

Fix: add table cases + goldens for the deterministic missing commands (board+subcommands, label, now, workon, resolve, bulk multi-id incl. a failing id, config get/set, hooks status, commit, version, merge-file on a conflict fixture, sprint archive/unarchive) and at least one exit-2 error case; document deliberate exclusions in the package comment. Notes: version needs version-string scrubbing; completion emits shell script, not JSON. Reuse seededRepo/diffFixture; add a conflictFixture from the integration diverge recipe.

Verify: goldens generated then re-run green; schema conformance ticket's validator covers the new goldens too.

Files: tests/contract/contract_test.go, tests/contract/testdata
Depends on: Contract test harness quality: checkStable, scrubbing, repoEnv, shared fixture
<!-- /kira:comment -->
