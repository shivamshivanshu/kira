---
id: 01M1J36JSS6DJGHR1RRNQGWD4Y
number: CORE-66
aliases: []
type: ticket
title: "The three item-read paths disagree, so --at HEAD orders differently from a clean working tree"
state: DONE
resolution: done
priority: P1
labels: []
parent: null
blocked_by: []
created: 2026-09-03T00:17:58+02:00
updated: 2026-11-13T22:12:43+05:30
---
`core.readRaw` (`internal/core/load.go:53`) dispatches to three loaders that
return structurally different items, chosen per-command by a flag, with no test
asserting they agree:

| path | Body | ExtraLinks / CRLF | Activity |
|---|---|---|---|
| `treeish.Load` (`--at`) | yes | yes | `= Updated` (`treeish/loader.go:90`) |
| `index.Load` (default) | **no** | **no** | max(updated, latest linked commit) |
| `s.load` (fallback) | yes | yes | `= Updated` (`core/store.go:102`) |

`activity` is queryable and sortable (`query/parser.go:116`), so the divergence
is user-visible. Reproduced on this repo with a clean tree, where `--at HEAD`
should be identical to the default read:

    kira list --query 'state != "" order by activity desc'
      CLI-14, KIRA-164, KIRA-163, CORE-64, CORE-59

    kira list --at HEAD --query 'state != "" order by activity desc'
      CLI-14, KIRA-164, CORE-64, CORE-59, CORE-65

Same tree, two orderings, no warning. KIRA-163 is third in one and outside the
top five in the other.

The index path's own doc comment (`index/read.go:30-34`) admits the lossiness:
"the cache stores no Body, ExtraLinks or CRLF, so a rehydrated item looks
rewritable even for a file kira must refuse to rewrite. Never feed these to a
write path." The correctness of every read command currently rests on a
convention enforced by a comment. (Separately verified: no write path does read
from the cache today -- every writer goes through `LoadAll()`. So the data-loss
risk is not live, but nothing enforces it.)

The ordering divergence is the live bug. Either `activity` means one thing and
all three paths compute it identically, or it is documented as
index-path-only and `--at` refuses to order by it.

What is missing regardless: a test that loads the same tree through all three
paths and asserts the items are equal on every field each path claims to
populate. `core/loader_test.go:29` only asserts `len(items)==1` and an ID match.
