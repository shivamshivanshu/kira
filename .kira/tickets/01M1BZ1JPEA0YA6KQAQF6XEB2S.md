---
id: 01M1BZ1JPEA0YA6KQAQF6XEB2S
number: KIRA-138
aliases: []
type: ticket
title: "Document the fragile cross-file invariants the blind reviewers had to reverse-engineer"
state: TODO
priority: P3
labels: []
epic: null
blocked_by: []
created: 2026-08-31T15:09:53+02:00
updated: 2026-08-31T15:09:53+02:00
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01M1BZ1JYBAQ0C8GQXKEMT6NWB author=Shivam-Shivanshu ts=2026-08-31T15:09:53+02:00 -->
Not defects - places where three independent blind reviewers each reported having to read three or more files to establish an invariant that nothing states locally. These are where the next maintainer introduces a bug.
1. core/eventparse.go walkPatch assumes every hunk contains the whole file; that holds ONLY because gitx.logPatch passes --unified=1000000 (gitx.go:123). Drop that flag and events for mid-file frontmatter changes silently vanish. Rated the single most fragile invariant found.
2. Event ordering is an undocumented cross-file contract: cachedEvents yields newest-first, blame.go:37-42 depends on that, history.go:54 reverses to oldest-first, now.go:81 then takes the last element as latest. Three files, both orders, no local statement of which is which.
3. The Extras 'rendered canonical YAML text' encoding is enforced nowhere at the trust boundary: datamodel.ExtraField(...).Set accepts any string and writeExtras splices *v.Scalar VERBATIM into frontmatter, so a caller that skips codec's render step emits invalid YAML with no error. Same for list elements ('already flow-safe' is a promise, not a check).
4. The extras SQL encoding (kind scalar/list plus a NULL-valued list row meaning 'empty list, key present') is documented at both ends but understandable only by reading both.
5. Staleness bookkeeping: why saveMeta only under needsRebuild is safe spans staleness.go, trailers.go and the previous run's save.
6. codec.AppendComment will happily write a body containing a kira:comment end marker and break its own parser; the guard lives in core.validateCommentText, with nothing in codec saying so.
7. index.Items() is not write-safe - Body, UnknownKeys, UnknownLinkTypes and CRLF are absent, so an index-rehydrated item has HasUnknown()==false even for a hostile file. No live defect today (every write path loads from the filesystem) but any future write path fed from the cache silently bypasses guardWritable.
Fix: a sentence at each site, and a doc contract on index.Items().
<!-- /kira:comment -->
