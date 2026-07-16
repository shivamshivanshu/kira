---
id: 01KXNSNJQZESNQAAA5MCB2H1WX
number: CORE-38
aliases: []
type: ticket
subtype: bug
title: "doctor: resolutionFindings never flags a Done-category state with a nil resolution"
state: TODO
priority: P3
labels: []
epic: null
blocked_by: []
created: 2026-07-16T21:15:34+05:30
updated: 2026-07-16T21:15:34+05:30
---

## Description

`internal/doctor/check.go`'s `resolutionFindings` only flags a non-nil
resolution set on a non-Done state; it never flags the mirror case: a
Done-category state with a nil resolution.

CORE-35 (this session) deliberately does not auto-fill a resolution when a
resolve/merge lands an item in a Done state with none surviving the merge
(auto-filling would fabricate data), so this shape -- Done state, nil
resolution -- can legitimately exist on disk with nothing in `kira doctor`
to catch or surface it.

## Acceptance criteria

- [ ] `resolutionFindings` (or a sibling doctor check) flags a Done-category
      state with a nil resolution as a finding, symmetric to the existing
      non-Done-with-resolution check.
- [ ] Decide the right severity/message -- likely a warning rather than an
      error, since this can be a legitimate transient state right after a
      merge, not necessarily a data-integrity bug.
- [ ] Existing doctor tests plus a new case for the mirror shape.

## Comments
