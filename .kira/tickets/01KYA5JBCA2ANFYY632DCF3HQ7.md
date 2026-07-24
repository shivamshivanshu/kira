---
id: 01KYA5JBCA2ANFYY632DCF3HQ7
number: CORE-46
aliases: []
type: ticket
title: "Deduplicate error on unresolvable ref extra in create path"
state: TODO
labels: []
epic: null
blocked_by: []
created: 2026-07-24T19:08:20+05:30
updated: 2026-07-24T19:08:20+05:30
---

In the create/`validateAssembled` path, an unresolvable schema-declared
ref extra is reported twice: once by `normalizeRefExtras` (via the shared
erroring `resolve` closure) and once by `validateItem`'s `CodeRef`
(`refResolver.Exists` returns false). The edit path returns early after
`normalizeAndCheckRefs`, so it is unaffected. Both messages are correct;
it is a cosmetic duplicate only.

Do NOT fix by suppressing `CodeRef` for ref extras: `refResolver.Exists`
checks target-type too, so `CodeRef` is the authoritative existence+type
check on write. Prefer making `normalizeRefExtras` normalize-only (drop
its error append), leaving `CodeRef` the single reporter — mind the
self-reference edge (`ulid == it.ID`) that the erroring path currently
catches.

Found in the CORE-4x post-merge review.
