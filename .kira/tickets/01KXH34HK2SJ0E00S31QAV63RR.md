---
id: 01KXH34HK2SJ0E00S31QAV63RR
number: CORE-32
aliases: []
type: ticket
subtype: task
title: "core DRY batch: findItem, mutateConfig reuse, renumber, RefPattern, storage constants"
state: DONE
priority: P3
labels: []
epic: null
blocked_by: []
created: 2026-07-15T01:24:49+05:30
updated: 2026-07-16T19:51:53+05:30
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01KXH34HRZ85M74HT26V6173Q3 author=Shivam-Shivanshu ts=2026-07-15T01:24:50+05:30 -->
- Identical resolve-ref triple at log.go:19-26, blame.go, boardmove.go, stats.go, closes.go, tree.go, and inside resolveRef (store.go:74-81) — the sites obtain items/resolver via different load paths (useIndex reads, or ld held under lock), so extract findItem(items, resolver, ref) and have resolveRef use it.
- label.go:23-66 inlines lock/read/parse/write/finalize that mutateConfig encapsulates — teach mutateConfig a no-op signal, rewrite LabelCreate on it.
- reconcile.go:34-37 exact-match alias dedupe vs boardmove.go:52-56 EqualFold+strip — one renumber(it, from, to) with EqualFold semantics.
- Ref grammar: hookrun.go:16 ticketRefRe and config.BoardKeyPattern encode the identical key shape (do NOT feed config keys into hookrun's gate — PrepareCommitMsgHook deliberately regex-checks the branch before loading config as a fast path; share a shape constant at most). index lenientPattern and workon.InferNumber are config-key-driven by design.
- Small: kiracommit.go:79 TrimSuffix/path.Ext -> storage.ULIDFromPath; move.go:58/:128 raw "resolution" -> datamodel.KeyResolution; mergedriver.go:8/init.go:15 restate storage.TicketsPrefix/DirName — build merge-attr/gitattributes lines from constants; workon.go doingTarget re-implements categoryOf; hints.go:12 pointless alias var; storage read.go:54 dead slices.Sort on ReadDir output; SkipNote hardcodes '.kira/tickets/'; fs.go:63 literal "templates"/".cache" with init.go duplicating ".cache" in the generated .gitignore — named constants shared with init.go.
- storage/discover_test.go: do NOT delete — it asserts the ErrStoreNotFound sentinel (errors.Is) while test/store_test.go asserts the errx.ExitEnv code; fold the sentinel assertion into the external test, then remove.

Verify: full suite green; behavior-preserving audit of each extraction.

Files: internal/core/log.go, internal/core/blame.go, internal/core/boardmove.go, internal/core/stats.go, internal/core/closes.go, internal/core/tree.go, internal/core/store.go, internal/core/label.go, internal/core/reconcile.go, internal/core/hookrun.go, internal/core/kiracommit.go, internal/core/move.go, internal/core/mergedriver.go, internal/core/init.go, internal/core/workon.go, internal/core/hints.go, internal/storage/read.go, internal/storage/fs.go, internal/storage/discover_test.go
<!-- /kira:comment -->
