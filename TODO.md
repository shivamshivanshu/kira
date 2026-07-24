# TODO

- Extend lossless round-trip to unknown link types (`links:`) — `codec.optLinks`
  still drops target IDs for unrecognized link types, so `guardWritable` still
  hard-blocks writes on them (CORE-3, same round-trip fix as phase 3a).
- Surface extras-only merge conflicts in `merge.Result.Arbitrated` —
  `arbitrated()` walks `datamodel.Fields`, which extras aren't in, so an
  extras-only conflict resolves via last-write-wins with no user signal
  (CORE-3, with the schema-driven field registry in phase 3b).
