---
id: 01KYA5J1SM7D368EYGWGWEYT4J
number: CORE-45
aliases: []
type: ticket
title: "Surface ExtraLinks conflicts in merge arbitration"
state: TODO
labels: []
epic: null
blocked_by: []
created: 2026-07-24T19:08:10+05:30
updated: 2026-07-24T19:08:10+05:30
---

Unknown link types round-trip via `Item.ExtraLinks` (CORE-41) and are
set-merged by `merge.linkMerge`, but a divergence on the same unknown link
type never appears in `Result.Arbitrated`: `datamodel.ChangedExtraKeys`
covers only `ExtraScalars`/`ExtraLists`, and `linksField.Changed` compares
only `it.Links`. So `kira resolve` neither reports nor offers a pick for a
conflicting extra link type, unlike a regular `links` conflict. No data
loss (still set-merged), only an informational/pick gap.

Fix: fold `ExtraLinks` into arbitration change-detection and into
`pickFields`/`copyExtra`, mirroring the scalar/list extras handling.

Found in the CORE-4x post-merge review.
