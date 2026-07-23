---
id: 01KY79HCREGTG4EVS4AEV8PGM6
number: CORE-40
aliases: []
type: ticket
title: "DRY: extract generic dedup/index-by-id/set-from-slice helpers found by generics survey"
state: TODO
priority: P3
labels: []
epic: null
blocked_by: []
created: 2026-07-23T16:19:59+05:30
updated: 2026-07-23T16:19:59+05:30
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01KY79J0BTEGQ9KHGVRKCVKF3B author=Shivam-Shivanshu ts=2026-07-23T16:20:19+05:30 -->
From the Go generics survey (2026-07-23). Repo generics usage is already disciplined (sortByKey, enumValues, storeActionRunE, ptr.*); these are the remaining concrete DRY wins:

1. (medium-high) 'seen-map dedup + ordered append' repeated ~8x with identical shape: cli/complete.go:213, id/resolve.go:100, doctor/doctor.go:107, gitx/gitx.go:150, index/trailers.go:342, merge/set.go:6,33, datamodel/graph.go:27. Extract a generic Deduper[K comparable] in a zero-dep leaf pkg (e.g. internal/setx). ~25-30 LoC.
2. (medium) 'index items by ID' (map[string]*Item) duplicated ~7x: core/store.go:178, doctor/graph.go:92, query/eval.go:255, index/read.go:76, core/validate.go:130, cli/tree.go:73, merge/body.go:47. A concrete datamodel.IndexByID(items) or generic IndexBy[T,K]. ~20 LoC.
3. (low, only if bundling #1) set-from-slice builders: merge/set.go:68 (asSet), doctor/env.go:89 (hookSet) -> toSet[T comparable].

Explicitly NOT worth it (surface-similar but not true dupes): merge/scalar.go threeWayScalar vs threeWayPtr (differ by ==vs ptr.Equal), datamodel/fields.go builders (per-kind formatting). Prefer stdlib slices/maps throughout.
<!-- /kira:comment -->
