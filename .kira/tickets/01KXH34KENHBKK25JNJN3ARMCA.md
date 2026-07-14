---
id: 01KXH34KENHBKK25JNJN3ARMCA
number: DX-9
aliases: []
type: ticket
subtype: task
title: "Contract test harness quality: checkStable, scrubbing, repoEnv, shared fixture"
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

<!-- kira:comment id=01KXH34KMFQYX74QHM4VNAPQV5 author=Shivam-Shivanshu ts=2026-07-15T01:24:51+05:30 -->
- contract_test.go:339 'out2, _, _ :=' and plain_test.go:25 ignore code/stderr on the stability rerun — extract checkStable(t, dir, got, args...) failing with stderr on nonzero, used by both.
- Line 414 repeats line 404's shaRE pass (redundant in practice — JSON quoting gives \b boundaries) — delete; ulidRE (:395) lacks \b anchors unlike shaRE so 27-char uppercase tokens partially scrub — anchor (noting the ULID alphabet is all word chars, so not airtight).
- Line 402 ReplaceAll(dir) misses darwin's /private/var Getwd resolution, breaking err-no-store for Mac contributors — EvalSymlinks the fixture dir at creation.
- sprint-close is readOnly:true but SprintClose mutates (internal/core/sprint.go: mutation loop 133-137, pointer removal 141-146) — holds only because the fixture never activates the sprint; mark readOnly:false (or add an activate-then-close case).
- mutateTicketFile (:225-228) continues on ReadFile errors to skip subdirs, masking real failures — IsDir-skip + Fatalf.
- Lines 379-386 parse Hint but assert only Error/Code — per-case wantHint; lines 332-335 never assert empty stderr on success — add.
- kira() pins GIT_*_DATE, gitCmd() does not, gitOutput() sets only HOME — one repoEnv(dir) helper for all three.
- seededRepo (~12 execs) is rebuilt by ~18 read-only invocations — build once (sync.OnceValues) and share. Caveat: read-only kira commands write a sqlite index cache under .kira (index/load.go Load->reindex with discard+force-rebuild on error), so sharing one dir across t.Parallel subtests races cross-process on the cache — pre-warm the index inside the OnceValues builder (run one 'kira list') or copy the fixture per subtest.

Verify: contract suite green and faster; goldens unchanged except where scrubbing tightened.

Files: tests/contract/contract_test.go, tests/contract/plain_test.go
<!-- /kira:comment -->
