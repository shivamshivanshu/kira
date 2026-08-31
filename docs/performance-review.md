# Performance review — 2026-08-31

Implementation results and reproducible commands are in [performance.md](performance.md).
The findings below record the original review before implementation.

The highest-value changes are to reuse the stats database handle, short-circuit
query predicates, skip unused Markdown projection, and reuse existing batches and
snapshots. Several scaling problems come from repeating work already done earlier
in the same operation.

Branch `feat/webui-dashboard` was rebased onto local `main`. The final reviewed
commit is `13124b2`. Main advanced during the review, so the branch was updated a
second time. The initial review added no production-code or test changes; the subsequent implementation is documented separately.

## Measurements

Local machine: Apple M3 Max, darwin/arm64, Go 1.26.5. Microbenchmarks use three
200 ms samples; the figures below are medians. They are directional measurements,
not statistical confidence intervals or promises of whole-application speedups.
Candidate implementations were supplied through temporary Go build overlays.

| Workload | Current implementation | Isolated candidate |
| --- | --- | --- |
| One warm event lookup in a cache with 1,000 event histories | 869.5 µs, 195 allocations | Same queries with a reused handle: 25.4 µs, 82 allocations; about 34× faster |
| Evaluate `state = DONE AND "important"` over 1,000 TODO items with mixed-case titles | 132.6 µs, 64 KB, 1,000 allocations | Short-circuit AND: 3.92 µs, zero allocations; about 34× faster |
| Validate one integer extra on an item with a 1,000-line description | 3.46 ms, 41.5 MB, 2,008 allocations | Parse body sections only if requested by the schema: 0.30 µs, 528 bytes, four allocations |
| Compute progress for 100 nested epics and one leaf ticket | 0.490 ms, 0.431 MB | Baseline only |
| Compute progress for 1,000 nested epics and one leaf ticket | 54.8 ms, 44.0 MB | Baseline only; 10× depth takes about 112× time |
| Render a seeded 1,000-item tree page, excluding data loading | 35.4 ms, 13.6 MB, about 459,145 allocations; 2,390,395 HTML bytes | Baseline only |

The existing `BenchmarkLoadPage`, with a seeded 1,000-item fixture and three timed
iterations, measured tree 62.5 ms, board 64.7 ms, stats 848.9 ms, and manage 57.7 ms.
These measure data loading, excluding template execution. The CPU profile places
62.5% of sampled CPU time under stats' `index.LogEntries` path. That profile also
includes fixture setup and warm-up, so it is supporting evidence, not an exact
breakdown of request latency.

The query, event-cache, epic-chain and page-load measurements were taken at
`93d6a23`, before the second rebase. The relevant hot paths remain unchanged in
`13124b2`; the intervening changes were reviewed. The Markdown projection and
template-rendering measurements were taken at `13124b2`.

## Recommended order

### 1. Open the stats event cache once per request

**High impact; medium effort.**

Locations: [stats.go](../internal/core/stats.go#L20),
[eventparse.go](../internal/core/eventparse.go#L168),
[events.go](../internal/index/events.go#L14),
[index.go](../internal/index/index.go#L28).

`Stats` calls `itemMetrics` for every scoped item. That reaches `LogEntries`, which
opens SQLite, configures the connection, checks its schema, queries events and
commit links, and closes the database for every item. Stats discards the commit
links it paid to query. A warm 1,000-item stats request consequently does roughly
1,000 database open/close cycles.

Expose event reads on an already-open `Index`, open it once for the stats operation,
and provide a state-events-only path. Then batch the event-head and event queries,
and prepare event inserts in `replaceEvents` for cache misses. The measured 34×
improvement above changes only handle reuse and retains all three queries.

For a cold cache, also batch history derivation: every stale item currently runs
its own `git log -p` through `deriveEvents`. The repository already has a
multi-file patch walker in `walkPatch`/`rangeEvents`. Preserve per-file head
invalidation, rewritten-history handling and the distinction between committed
and uncommitted items.

Go's [`sql.DB` documentation](https://pkg.go.dev/database/sql#DB) describes the
handle as a connection pool intended for reuse.

### 2. Restore short-circuit evaluation in compound queries

**High benefit for selective filters; small effort.**

Location: [eval.go](../internal/query/eval.go#L136).

`join(l(it, cfg), r(it, cfg))` evaluates both predicates before entering `join`.
The `a && b` or `a || b` inside that function cannot avoid evaluating the right
operand. This repeats title lowercasing, date parsing, extra decoding or blocker
scans even when the result is already known.

Compile AND into `l(it, cfg) && r(it, cfg)` and OR into
`l(it, cfg) || r(it, cfg)`. Continue compiling both operands up front so invalid
fields, references and warnings retain their existing behavior. The candidate
passed the existing query tests. The measured case rejects every item on the
left; gains depend on real filter selectivity and operand cost.

### 3. Make body-section projection lazy and linear

**High allocation benefit for long descriptions; small effort.**

Location: [project.go](../internal/entityschema/project.go#L14).

`ValidateExtras` filters its schema to frontmatter fields and then calls
`ProjectItem`. `ProjectItem` still calls `bodySections`, although none of those
fields need Markdown. Each `sections[current] += line + "\n"` also recopies all
previous text in the section, making long sections quadratic in copied bytes.

Create the section map only when the first body field is encountered. This alone
removes the 41.5 MB allocation cost in the measured extras-validation case; the
candidate passed the existing entity-schema tests. For actual body projection,
use source offsets or a builder per section to make construction linear. Preserve
the existing behavior for repeated headings and whitespace trimming.

### 4. Keep web bulk mutations inside one core batch

**High benefit on large repositories; small-to-medium effort.**

Locations: [bulk.go](../internal/webui/bulk.go#L57),
[mutate.go](../internal/core/mutate.go#L71),
[assign.go](../internal/core/assign.go#L14),
[label.go](../internal/core/label.go#L74).

`resolveBulk` opens a batch, loads and resolves all tickets, then closes it.
`bulkItem` subsequently calls the single-item Store APIs. Move, assign and label
operations each open another batch and reparse the repository. With N tickets and
K selected items, the loading portion is O(KN).

Keep the batch through resolution and application, using its existing `Move`,
`Assign` and `LabelSet` methods. Preserve per-item commits, automation, warnings,
partial-success reporting and sequential WIP validation. Sprint edits and board
moves need separate treatment because they do not have equivalent batch methods.
Do not call a Store method that reacquires the filesystem lock while holding a batch.

### 5. Reuse snapshots throughout a UI refresh

**High benefit across routine navigation; medium effort.**

Locations: [filter.go](../internal/tui/filter.go#L10),
[snapshot.go](../internal/core/snapshot.go#L19),
[page.go](../internal/webui/page.go#L44).

An unfiltered TUI tree refresh independently calls `Tree`, `List` and
`EpicProgress`, refreshing and loading the index three times. A filtered refresh
uses `ListWithMatches`, which reparses all Markdown between two index-backed reads.
Use one `Snapshot` for the tree, fields, matching and progress; add a snapshot
matching API so rows for retained ancestors remain available.

The web stats page creates a snapshot and then calls `Store.Stats`, repeating the
item load. Add `Snapshot.Stats` backed by the same loaded items. Board pages also
filter and sort through both `Snapshot.List` and `Snapshot.Board`; share the
matched set while retaining the board's global WIP-count semantics.

Keep reuse request-scoped so external ticket edits remain visible on refresh.

### 6. Compute all epic progress in one postorder traversal

**High benefit for deep nesting; medium effort.**

Location: [progress.go](../internal/core/progress.go#L31).

`allEpicProgress` walks every epic's descendants separately. Descendants are
revisited once per ancestor; a chain has O(N²) work. Memoize child totals and done
counts in one traversal, with visiting/visited states for cycle detection. Count
non-epic children according to the current semantics. The chain measurements
demonstrate the problem; normal shallow boards will see smaller gains.

The web [treeRows](../internal/webui/page.go#L147) has a related copying problem:
each recursive call returns a flattened slice which is copied into its parent's
slice. Mark retained ancestors in one pass and append each visible row once into
a preallocated output in preorder. Preserve ancestor visibility under filters.

### 7. Reduce full-page HTML and repeated rendering work

**Meaningful UI cost; medium-to-large effort.**

Locations: [tree.html](../internal/webui/templates/tree.html),
[item_copy.html](../internal/webui/templates/item_copy.html),
[pickers.html](../internal/webui/templates/pickers.html),
[viewmodel.go](../internal/webui/viewmodel.go#L64).

A seeded 1,000-item tree emits about 2.39 MB of HTML. Every row includes a copy
menu template, and the jump picker renders all items again. Most navigation
reloads this page. Rendering alone adds about 35 ms and 13.6 MB of allocations in
the probe; browser parsing/layout costs were not measured.

Share one copy-menu component, populate it for the selected item, and lazy-load or
limit the picker results. Consider pagination or progressive tree expansion for
larger repositories. Cache parsed base query parameters per page instead of
reparsing `p.URL` on every `Link` call. Build item-name and selected-ID maps once
instead of doing linear membership scans from template helpers. Preserve URL
escaping, filter parameters and the configured workon reference format.

### 8. Skip the number map when trailer scans are already current

**Moderate benefit on every warm read; small effort.**

Locations: [staleness.go](../internal/index/staleness.go#L45),
[trailers.go](../internal/index/trailers.go#L390).

Even an unchanged index reads every live number and alias into a map before
`scanTrailers` discovers that its watermark already equals HEAD. Load this map
lazily when a trailer or closes scan actually needs it. Check scan-config changes
and the independent landed-ref watermark before deciding it is unnecessary.
This removes an additional full-cache scan without weakening dirty-file detection.

### 9. Resolve query identity only when needed

**Small change with subprocess savings.**

Locations: [list.go](../internal/core/list.go#L87),
[store.go](../internal/core/store.go#L134).

Every query context resolves identity through separate `git config user.name`
and `git config user.email` processes, including plain list and board requests.
Supply a request-scoped lazy identity resolver invoked only by owner/reporter
`@me` predicates, or collect both values in one Git invocation. Inspect the parsed
expression so quoted text containing `@me` does not trigger the lookup.

The web active banner also requests a complete `NowResult`, including history,
blockers and commits, while displaying basic item fields, branch and blocker count.
A smaller snapshot active-item view can retain those fields and avoid history and
commit-list work when an item is active.

### 10. Aggregate sprint counts in one pass

**Moderate benefit as the number of sprints grows; small effort.**

Location: [sprint.go](../internal/core/sprint.go#L80).

`sprintList` scans every item for every configured sprint. Accumulate counts by
sprint key in one pass, then emit rows in configured order: O(N + S), replacing
O(N × S). This is on every successful dashboard snapshot load, including views
that do not display detailed sprint statistics.

## Smaller opportunities

These are source-confirmed extra work, with benefit still to measure individually.

| Location | Opportunity |
| --- | --- |
| [query/eval.go](../internal/query/eval.go#L294), `blockedPred` | A boolean query builds complete open/skipped slices just to test whether any open blocker exists. Use an allocation-free early-exit predicate, preserving unknown/dangling blocker treatment. Reuse the item index within one compilation. |
| [query/eval.go](../internal/query/eval.go#L238), `compileIn` | For sufficiently large scalar IN lists, normalize values once into a set and extract the item's field once. Keep specialized reference, list and numeric semantics. Small IN lists may favor the existing loop. |
| [core/validate.go](../internal/core/validate.go#L144), `validateGraph` | Every mutation builds an ID map and scans/sorts the whole graph for each cycle relation, then only uses cycles containing the changed item. Investigate traversal rooted at that item and reuse batch graph indexes, retaining validation of pre-existing invalid relationships. |
| [codec/comments.go](../internal/codec/comments.go#L145), `JoinComments` | Repeated `AppendComment` recopies the growing body. Use one builder, preserving byte-for-byte comment separators. This also benefits `SplitComments` and merges that reconstruct bodies. |
| [core/log.go](../internal/core/log.go#L35), [merge/body.go](../internal/merge/body.go#L59) | Timestamp sorting parses both timestamps for each comparison. Compute parsed keys once per element, following the existing `core.sortByKey` approach; retain tie and malformed-timestamp ordering. |
| [doctor/check.go](../internal/doctor/check.go#L79) | `People.Names()` allocates a fresh slice for each owner/reporter check. Build a vocabulary lookup once for the doctor run, or directly scan the existing person slice for small vocabularies. |
| [core/comment.go](../internal/core/comment.go#L102) | Replace `ContainsFunc(strings.Split(...))` with a `strings.SplitSeq` loop and return on the first marker. This avoids the line-slice allocation. |

## Go features with practical value

- **`errors.AsType[T]` (Go 1.26):** seven production `errors.As` sites remain in
  query, errx, core, rgx, fzfx, gitx and cli. Convert them when touching those paths.
  The API is type-safe and faster, but most sites are error paths, so the expected
  user-visible performance benefit is small. See the
  [Go 1.26 release notes](https://go.dev/doc/go1.26#errors).
- **Iterator APIs:** `SplitSeq` is already widely adopted. The comment check above
  is a remaining useful case. Indexed parsers should keep slices where they need
  lookahead or random access.
- **Profile-guided optimization:** after the repeated I/O is addressed, gather a
  representative mix of CLI and UI CPU profiles and compare builds with
  `-pgo=off` and `-pgo=<profile>`. A validated profile can live at
  `cmd/kira/default.pgo`. The [official PGO guide](https://go.dev/doc/pgo) stresses
  representative profiles; these narrow microbenchmarks are unsuitable as the
  sole input. No Kira PGO improvement has been measured in this review.
- **Existing adoption:** the module and CI already target Go 1.26. `new(expr)`,
  `B.Loop`, `reflect.TypeFor`, generic slices/maps helpers and integer ranges are
  in use. `go fix -diff ./...` at `93d6a23` produced no changes. Go 1.26's Green Tea
  GC and additional stack-allocation optimizations are already enabled by the
  toolchain; see the [release notes](https://go.dev/doc/go1.26).

## Performance measurement gaps

The existing [perf harness](../tests/perf/harness_test.go#L86) warms both the index
and stats histories before measuring commands or counting Git spawns. It therefore
does not expose cold-history stats' per-item Git calls. Add separate cold-cache,
warm-cache and one-item-changed scenarios when implementing the stats changes.

The [perf workflow](../.github/workflows/perf.yml) compares `BenchmarkCommands`
only. Add mutation latency, bulk operations, page rendering and deep-epic cases to
the performance comparison. Its comment says A/B samples are interleaved, but the
actual steps run all base samples followed by all HEAD samples; alternate them to
reduce machine-load drift. These are benchmark improvements, not test repairs.

Temporary probe sources and build overlays are under `/tmp/kira-review-*`; CPU
and allocation profiles are `/tmp/kira-webui-review.cpu` and
`/tmp/kira-webui-review.mem`. Query short-circuit and lazy-body candidates passed
their existing package tests. The original full-suite run had only three
environment-sensitive theme failures; those passed with `NO_COLOR` removed.
No tests were edited. The full suite was not repeated after the second rebase.
