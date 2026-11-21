# Performance changes

## Cold stats on Kira's history

Ran five paired samples against a local clone of `bdbba3e` with 286 current
tickets, 1,175 commits and 44 reachable merges. Cold runs started with no Kira
cache. Each warm run immediately followed its cold run.

| Workload | Before, median (range) | After, median (range) |
| --- | --- | --- |
| Cold `stats --json` | 4.69 s (4.66–4.72 s) | 1.38 s (1.35–1.38 s) |
| Warm `stats --json` | 52.0 ms (50.8–52.8 ms) | 53.9 ms (51.9–55.6 ms) |
| Cold Git processes | 292 | 99 |

A first-parent merge changed 90 current ticket paths relative to its first
parent. Those tickets retained per-file logs. The other 196 shared a patch
scan. Stats JSON and all 1,116 cached event and event-head rows matched
between binaries. The warm trace showed that rebuilding the per-ticket commit
map took about 30 ms on its own.

The synthetic scaling check used nine Git processes for linear history and
ten for a merge affecting one ticket, at both 100 and 1,000 tickets. A
follow-up that reads project-relative paths directly from Git removed one
process in each case.

Measurements used Linux/amd64, Go 1.27-devel and Git 2.55.0 on a shared host.
The CLI and its Git children ran on CPU 24. OS caches and CPU frequency were
uncontrolled. To reproduce, build both binaries with `go build -o <binary>
./cmd/kira`, check out `bdbba3e` in a disposable clone, and run the following
for each binary from that clone after moving aside its `.kira/.cache`. Use the
same available CPU for every sample. Alternate which binary goes first and
discard one initial pair.

```sh
export KIRA_NOW=2026-11-21T00:00:00Z
export GIT_CONFIG_NOSYSTEM=1 GIT_CONFIG_GLOBAL=/dev/null
/usr/bin/time taskset -c 24 "$binary" stats --json > cold.json
/usr/bin/time taskset -c 24 "$binary" stats --json > warm.json
```

## Allocation reductions

Each change was measured in five paired microbenchmark samples on the same
host with Go 1.27-devel. The patch row combines the two parser changes. Its
input was 1.49 MB of first-parent patches from `bdbba3e`. The item scan used
its warm 286-ticket SQLite index. The match benchmark timed a hit in a
snapshot with an already populated match cache.

| Workload | Before | After |
| --- | --- | --- |
| Parse a Git patch stream | 7.94 ms, 9.84 MB, 68,962 allocations | 6.16 ms, 7.31 MB, 48,267 allocations |
| Match cache hit | 1.18 µs, 432 B, 2 allocations | 93 ns, 0 B, 0 allocations |
| Read indexed items | 668 µs, 297 kB, 9,432 allocations | 602 µs, 231 kB, 6,582 allocations |

The patch parser reuses frontmatter buffers and skips decoding equal blocks.
Snapshot matches use the options struct as their key. Index reads reuse
nullable SQL scan buffers while copying values into each result item.

Across 12 paired CLI samples, `list`, warm `stats`, and
`changes --since HEAD~300` produced identical JSON. Elapsed-time ranges
overlapped for all three commands. With the same source built using Go
1.26.11 and 1.27.4, their ranges also overlapped.

## Earlier performance pass

Results of the performance pass in commit `36c7fae`. Measurements are local to
an Apple M3 Max with Go 1.26.5 on darwin/arm64; they are workload-specific, not
universal speedups.

| Workload | Before | After |
| --- | --- | --- |
| Load a 1,000-item stats page | 848.9 ms | 45.2 ms |
| Load a 1,000-item tree page | 62.5 ms | 31.5 ms |
| Load a 1,000-item board page | 64.7 ms | 31.9 ms |
| Compute progress for 1,000 nested epics | 54.8 ms, 44.0 MB allocated | 0.296 ms, 0.522 MB allocated |
| Evaluate a selective AND query over 1,000 items | 132.6 µs, 1,000 allocations | 3.92 µs, zero allocations |
| Render a 1,000-item tree page, excluding loading | 35.4 ms, 2.39 MB HTML | 22.7 ms, 1.60 MB HTML |

After values are medians of three 200 ms benchmark samples. Page-load baseline measurements used three timed
iterations; comparisons are directional rather than controlled confidence bounds.

## Implementation

- Stats opens one index handle, reads event heads and state events in chunks, and
  writes stale histories with prepared statements in one transaction. It avoids
  loading commit links and reuses an existing snapshot when invoked by the UI.
- Cold history batches now share a first-parent patch scan for paths that
  match the first parent at every first-parent merge. Paths changed at one of
  those merges retain per-file logs because Git can simplify different paths
  through different parents. The scaling check covers 100 and 1,000 tickets
  with linear history and with a merge affecting one ticket.
- Query AND/OR short-circuit during evaluation while both operands still compile.
  Large scalar IN sets extract a field once, boolean blocker checks return on the
  first open blocker, and `@me` is resolved lazily once per query context.
- Frontmatter-only validation skips Markdown projection. Body sections and
  comment assembly use builders instead of repeatedly copying growing strings.
- All web bulk operations share a core batch, including sprint edits and board
  moves. CLI bulk field edits use the same path. Each item retains its own
  validation, commit and success/error result. Batch indexes track
  successful updates and writes that precede a failed commit.
- Mutation cycle validation visits the changed item's reachable graph and reuses
  batch indexes. Epic progress uses one memoized postorder traversal; web tree
  flattening emits each retained row once. Sprint counts use one aggregation pass.
- The TUI tree and web views reuse request snapshots and matching results. The
  active-item banner reads only its displayed fields, avoiding history queries.
- Rendering parses query parameters and indexes names and selections once. Copy
  buttons share one template. The picker searches compact item data and creates
  at most 50 result links, including matches beyond the first 50 items.
- Fresh index reads defer number/alias maps until a trailer scan actually needs
  them. History/comment sorts parse timestamp keys once. Doctor checks person
  names without allocating a names slice per field. Remaining production
  `errors.As` calls use `errors.AsType`.

The reusable snapshots and lazy values are scoped to an operation, preserving
refresh visibility for external edits. Full tree rows remain available for
keyboard navigation, filtering, ancestor expansion and bulk selection.

## Reproduce the benchmarks

```sh
go test ./internal/query ./internal/core ./internal/webui \
  -run '^$' -bench 'Benchmark(SelectiveQuery|ContainerProgress|LoadPage|RenderPage)$' \
  -benchtime=200ms -count=3 -p=1

go test ./tests/perf -run '^$' \
  -bench 'Benchmark(Mutations|StatsCache)$' -benchtime=3x -count=3

KIRA_PERF=1 go test ./tests/perf -run TestColdStatsSpawnScaling -v
```

Fixtures live in temporary repositories. Mutation and stats-cache benchmarks
exclude fixture copying and cleanup, and separately measure cold, warm and
one-item-changed caches. CI (`.github/workflows/perf.yml`, non-gating) runs no
benchmarks: only the deterministic spawn-count, scaling and cold-start
tripwires in `tests/perf`.

## PGO experiments

`BenchmarkProfileWorkload` mixes item-detail reads, tree/board/stats loading,
template rendering and field mutations. It supplies a repeatable experiment for
the shared core and UI code; it is not a production traffic trace.

Six alternating samples of five workload iterations measured a median of
**324.0 ms without PGO and 328.0 ms with PGO**. Sample ranges overlapped
(315.5–337.5 ms and 313.5–341.0 ms respectively). This experiment did not show a
consistent improvement, so no `default.pgo` or release-build change was added.
The workload and commands remain available for evaluation with real usage profiles.

```sh
go test ./internal/webui -run '^$' -bench '^BenchmarkProfileWorkload$' \
  -benchtime=10s -cpuprofile=/tmp/kira.cpu -o /tmp/kira-profile.test
go test -c -pgo=off -o /tmp/kira-off.test ./internal/webui
go test -c -pgo=/tmp/kira.cpu -o /tmp/kira-pgo.test ./internal/webui
```

Run the two binaries with `-test.run='^$' -test.bench='^BenchmarkProfileWorkload$'
-test.benchtime=5x -test.count=1`, alternating their order across samples. Use
`benchstat` for a longer comparison before adopting a release profile. Go's
[PGO guidance](https://go.dev/doc/pgo) calls for representative workloads; a single
fast microbenchmark is insufficient evidence for changing release builds.

## Validation

- Full suite with the race detector passed using
  `go test -race -exec='env -u NO_COLOR' ./...`.
- The pinned golangci-lint 2.12.2 reported zero issues.
- `go build ./...` and `go vet ./...` passed. All new mutation and stats-cache
  benchmark scenarios completed successfully in a one-iteration smoke run.
- The unrelated theme tests were not changed; their existing expectations require
  `NO_COLOR` to be unset.
- New regression checks cover batched cache invalidation, empty and uncommitted
  histories, per-file equivalence across linear and merged Git history,
  short-circuit behavior, large IN sets, repeated Markdown headings, and epic
  cycles. Existing bulk, snapshot, rendering, contract and integration tests pass.
- Go rendering tests verify escaped copy values; JavaScript tests verify exact
  values in the shared copy picker, picker limits and finding items outside the
  initial result window.
- Interactive browser verification was unavailable: browser discovery returned
  no connected browser sessions.
