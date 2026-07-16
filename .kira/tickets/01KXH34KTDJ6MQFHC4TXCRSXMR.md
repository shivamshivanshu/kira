---
id: 01KXH34KTDJ6MQFHC4TXCRSXMR
number: DX-10
aliases: []
type: ticket
subtype: task
title: "Perf suite: BenchmarkMutations in A/B compare, determinism dedup, harness hygiene"
state: WONT_DO
resolution: dropped
priority: P3
labels: []
epic: null
blocked_by: []
created: 2026-07-15T01:24:52+05:30
updated: 2026-07-16T18:56:00+05:30
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01KXH34M0684HAQTCXMQ24J7X9 author=Shivam-Shivanshu ts=2026-07-15T01:24:52+05:30 -->
Substantive gap: write-path latency — mutations are spawn-counted but never benchmarked; perf.yml's benchstat A/B compares BenchmarkCommands (reads) only, so a write-path latency regression is invisible. Add BenchmarkMutations (StopTimer around freshFixture) to the compare. (Determinism DOES gate CI: perf.yml sets KIRA_PERF=1 and runs TestSpawnCounts/TestSpawnCountsMutations non-continue-on-error on every PR/push; residual gap is only local 'go test ./...' and the macOS ci.yml leg — worth a sentence in docs, not a fix.)

Hygiene: spawn_test.go:9-26 vs :28-44 identical determinism loops — extract checkSpawnDeterminism. harness_test.go:110-116 repoRoot falls back to '.' (build then fails in tests/perf with confusing noise; GOMOD=/dev/null yields /dev) — take tb and Fatalf. freshFixture/gitShim hand-roll MkdirTemp+registerTmp, accumulating multi-thousand-file fixtures until process exit — tb.TempDir(). bench_test.go:12-17 ignores invalid KIRA_BENCH_SIZE (typo silently benchmarks the wrong corpus for A/B CI) — Fatalf on set-but-invalid. bench_test.go:21 inlines the git-installed skip — split requireGit out of requirePerf and reuse. coldstart prints literal 'min of 11' beside the coldStartSamples const — interpolate. scaling_test.go:38-41 prints growth=n/a whenever minC==0 — report '0->N' growth explicitly (report-only; TestScaling is declared non-gating).

Verify: perf job green; benchstat picks up the mutations benchmark.

Files: tests/perf/spawn_test.go, tests/perf/harness_test.go, tests/perf/bench_test.go, tests/perf/coldstart_test.go, tests/perf/scaling_test.go, .github/workflows/perf.yml
Depends on: Consolidate test binary/env/shim infrastructure into testutil
<!-- /kira:comment -->
