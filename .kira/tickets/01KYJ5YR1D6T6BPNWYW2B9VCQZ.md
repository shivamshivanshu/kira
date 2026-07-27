---
id: 01KYJ5YR1D6T6BPNWYW2B9VCQZ
number: CORE-49
aliases: []
type: ticket
title: "CORE-3 design doc status line is stale re: Phase 4 scope"
state: TODO
labels: []
epic: null
blocked_by: []
created: 2026-07-27T21:49:01+05:30
updated: 2026-07-27T21:49:01+05:30
---

`docs/design/CORE-3-extensible-schema.md`'s Status line reads "Phases 1-3
done; Phase 4 next", framing Phase 4 (user-defined new entity types) as
entirely not-yet-built.

Verified while writing docs/user_manual/ that this overstates the gap: a
brand-new item type already works generically today with only a
`workflows:` entry in `.kira/config.yaml` -- no `.kira/schema/<type>.json`
required. `create`/`edit`/`move`/`list`/`show`/`doctor`'s state checks all
handle it immediately, with the full built-in field set.

The real remaining gap is narrower: `doctor`'s dangling-reference scan only
follows the built-in `epic`/`blocked_by`/`links` relationships, not an
arbitrary custom `ref` field declared on a type via `.kira/schema/`. Custom
fields are also not yet surfaced by `show`/`list`/the query grammar
(that part of the original Phase 4/5 framing still holds).

Action: revisit the CORE-3 design doc's phase breakdown and status line to
reflect what's actually done vs. actually remaining, so Phase 4 tracks the
narrower real gap rather than the wholesale "new types don't work" framing.

Found while writing docs/user_manual/09-limitations-and-roadmap.md.
