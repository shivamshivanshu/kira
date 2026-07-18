---
id: 01KXH349S7JQV6RX85AVMADK4V
number: DX-6
aliases: []
type: ticket
subtype: task
title: "Consolidate test binary/env/shim infrastructure into testutil"
state: DONE
priority: P2
labels: []
epic: null
blocked_by: []
created: 2026-07-15T01:24:41+05:30
updated: 2026-07-18T17:09:57+05:30
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01KXH349Z4MPHKYRR4M84NPA2R author=Shivam-Shivanshu ts=2026-07-15T01:24:42+05:30 -->
Five kira-binary build sites: contract TestMain:38, integration TestMain:49 (merge_test.go:44 leaks its temp dir on the build-failure panic path — no defer, panic at :52 skips RemoveAll at :56), perf kiraBin (harness_test.go:83-108, only one honoring KIRA_BIN), e2e crash_exit_test.go:11-14 and uninit_exit:13-16 (full compile+link per test, though e2e's kiraBinary() symlink trick already exists). Git-counting shim script duplicated at perf/harness_test.go:195-212 (spawnCount reader :214-231) and e2e/complete_spawn_test.go:61-86, both splicing the LookPath git path unquoted. Hermetic env triple restated: testutil init() + contract baseEnv (EDITOR drift only — kira() at contract_test.go:78 appends EDITOR=true) + perf TestMain (dead re-set) + 3 e2e files + gitx/trailers_test and internal/core/blame_test (extra GIT_AUTHOR_* others lack); contract alone pins commit dates. GitInit runs bare 'git init' so branch name depends on git version (matrix_test pins -b main; closes.go falls back to 'main'); GitInit/InitGitRepo are confusable, take *testing.T not TB, spawn 3 processes, and callers hand-roll Fatalf wrappers (internal/core/mergefile_test.go:45, internal/core/hookrun_test.go:18).

Fix: consolidate in internal/testutil — KiraBin(tb) (sync.Once + KIRA_BIN), GitCountingShim/CountSpawns (single-quoted git path, counter baked in), HermeticGitEnv() []string applied in-process and appended to every spawned cmd.Env, git init -b main with identity (+ optional pinned-date env), testing.TB signatures with self-Fatalf helpers (InitRepoAt/TempGitRepo). Switch all five build sites, both shim copies, and the inline env restatements onto it; move trailers_test to package gitx_test for the import.

Verify: full test suite green; spawn-count tests still deterministic.

Files: internal/testutil/testutil.go, tests/contract/contract_test.go, tests/integration/merge_test.go, tests/perf/harness_test.go, tests/e2e/crash_exit_test.go, tests/e2e/uninit_exit_test.go, tests/e2e/complete_spawn_test.go, internal/gitx/trailers_test.go, internal/core/mergefile_test.go, internal/core/hookrun_test.go
Depends on: testutil: neutralize the user config tier for all test binaries
<!-- /kira:comment -->
