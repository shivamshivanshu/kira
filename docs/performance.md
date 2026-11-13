# Performance changes

This implements the concrete code improvements identified in a performance
review. Measurements are local to an Apple M3 Max with
Go 1.26.5 on darwin/arm64; they are workload-specific, not universal speedups.

| Workload | Before | After |
| --- | --- | --- |
| Load a 1,000-item stats page | 848.9 ms | 45.2 ms |
| Load a 1,000-item tree page | 62.5 ms | 31.5 ms |
| Load a 1,000-item board page | 64.7 ms | 31.9 ms |
| Compute progress for 1,000 nested epics | 54.8 ms, 44.0 MB allocated | 0.296 ms, 0.522 MB allocated |
| Evaluate a selective AND query over 1,000 items | 132.6 µs, 1,000 allocations | 3.92 µs, zero allocations |
| Validate an extra with a 1,000-line description | 3.46 ms, 41.5 MB allocated | 0.275 µs, 528 bytes allocated |
| Render a 1,000-item tree page, excluding loading | 35.4 ms, 2.39 MB HTML | 22.7 ms, 1.60 MB HTML |

Before values and methodology are in the review. After values are medians of
three 200 ms benchmark samples. Page-load baseline measurements used three timed
iterations; comparisons are directional rather than controlled confidence bounds.

## Implementation

- Stats opens one index handle, reads event heads and state events in chunks, and
  writes stale histories with prepared statements in one transaction. It avoids
  loading commit links and reuses an existing snapshot when invoked by the UI.
- Cold histories are derived in one Git patch scan when the history is linear.
  A scaling check reports **seven Git processes for both 100 and 1,000 items**.
  Histories containing merges keep per-file derivation because Git path
  simplification can choose different parents for different pathspecs. Warm reads
  still use the batched cache in that case.
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
  -run '^$' -bench 'Benchmark(SelectiveQuery|EpicProgress|LoadPage|RenderPage)$' \
  -benchtime=200ms -count=3 -p=1

go test ./tests/perf -run '^$' \
  -bench 'Benchmark(Mutations|StatsCache)$' -benchtime=3x -count=3

KIRA_PERF=1 go test ./tests/perf -run TestColdStatsSpawnScaling -v
```

Fixtures live in temporary repositories. Mutation and stats-cache benchmarks
exclude fixture copying and cleanup, and separately measure cold, warm and
one-item-changed caches. CI alternates base and HEAD samples, compares mutation
and bulk latency as well as read commands, and runs the same query, schema,
deep-epic and rendering benchmarks against both revisions.

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
