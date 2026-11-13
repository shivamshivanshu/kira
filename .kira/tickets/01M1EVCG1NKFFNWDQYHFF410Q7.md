---
id: 01M1EVCG1NKFFNWDQYHFF410Q7
number: KIRA-160
aliases: []
type: ticket
title: "Equal extras values with different YAML spellings compare unequal"
state: WONT_DO
resolution: dropped
priority: P3
labels: []
parent: null
blocked_by: []
created: 2026-09-01T18:03:40+02:00
updated: 2026-11-13T17:19:14+05:30
---

## Description

## Acceptance criteria

<!-- kira:comment id=01M1EVDA05NMWS2QC9JMZSX779 author=Shivam-Shivanshu ts=2026-09-01T18:04:06+02:00 -->
Verified against yaml.v3. Extras store the rendered YAML text, and YAML has several spellings per typed value, so two byte-different stored forms can be the same value:

  42 / 0x2a / +42   all decode to int 42
  true / True / TRUE  all decode to bool true
  null / Null / ~     all decode to nil

Consequence: branches holding 'tags: [true]' and 'tags: [True]' merge to '[true, True]', and kira diff shows '[true] -> [True]' for a semantic no-op.

NOT fixing by canonicalising on store: that would rewrite a user's 0x2a as 42, which is exactly the round-trip fidelity KIRA-157 exists to provide. If this is ever worth addressing, canonicalise the COMPARISON in merge's set operations only, leaving storage verbatim.

Reachable by hand-edit only — kira's own writer emits one spelling per value. Found by the C++-perspective reviewer after it correctly withdrew a broader claim that 42 vs "42" needed canonicalising; those two are genuinely different values and the diff display now distinguishes them.
<!-- /kira:comment -->
