---
id: 01KXH34K2KY294QHG71JEB7GTB
number: DX-8
aliases: []
type: ticket
subtype: task
title: "Integration merge tests: shared gitTextMerge oracle, error checks, leaf parallelism"
state: TODO
priority: P2
labels: []
epic: null
blocked_by: []
created: 2026-07-15T01:24:51+05:30
updated: 2026-07-15T01:24:51+05:30
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01KXH34K8KJNFJCBW37R3PDTH8 author=Shivam-Shivanshu ts=2026-07-15T01:24:51+05:30 -->
Oracle fidelity: three copies of the (b,o,t) -> gitx.MergeText adapter with err => ("", true): production internal/core/mergefile.go gitTextMerge, tests/integration/merge_test.go:102-108 (used by assertDriverMatchesEngine, matrix_test.go:508-509 — same package), and internal/merge/test/merge_test.go:32-38 (engine unit tests :149,:200-201). If only one changes, the driver-vs-engine check can pass while they disagree. Fix: export one canonical adapter (core.GitTextMerge or merge.GitTextMerger); delete the local copies.

Quality batch (same files): matrix_test.go:505-508 'bi, _ := codec.Parse(...)' for all three sides — nil ours/theirs would panic merge.Merge (merge.go:18 dereferences ours.Updated) as a misattributed failure; Fatalf on ours/theirs parse errors (nil-tolerant base matches production parseOrNil). diff_test.go:97-99/:137-138 drop add/commit/checkout errors; 'mainBranch, _' would later 'git checkout ""'. 7 sites (4 integration + 3 diff) ignore Discover/Config errors while discoverStore (matrix_test.go:375) does it right — reuse it. Add leaf-level t.Parallel() (each leaf is hermetic); fold TestByteIdenticalSyncVsDriver (:454-470) into the matrix — the cross-path byte comparison needs both worlds, so capture per-path ticketBytes and compare after both parallel leaves finish; assertDriverMatchesEngine folds cleanly into the driver leg. Dedup: writeCommit/diverge vs world.write/commit; mergeItem≈matrixItem — one testItem constructor; delete pass-through wrappers (initGitRepo/coreDiscover/mergeRelPath); generalize setMergePolicyManual into flipConfig (integration_test.go:86-88 flips config via unchecked Replace); move TestListFilters' unique owner/category/state/number-sort assertions into core's list table and delete it (pure core.List unit test in the wrong tier).

Verify: suite green, wall-time drop from parallelism.

Files: tests/integration/merge_test.go, tests/integration/matrix_test.go, tests/integration/diff_test.go, tests/integration/integration_test.go, internal/merge/test/merge_test.go, internal/core/mergefile.go
<!-- /kira:comment -->
