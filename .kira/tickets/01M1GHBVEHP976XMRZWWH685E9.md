---
id: 01M1GHBVEHP976XMRZWWH685E9
number: KIRA-164
aliases: []
type: ticket
title: "Forward compatibility covers only flat scalars and scalar lists"
state: DONE
resolution: done
priority: P1
labels: []
parent: null
blocked_by: []
created: 2026-09-02T09:47:02+02:00
updated: 2026-09-03T00:00:06+02:00
---

## Description

## Acceptance criteria

<!-- kira:comment id=01M1GHCQTF1479T1WAJMF85XB1 author=Shivam-Shivanshu ts=2026-09-02T09:47:31+02:00 -->
The claim that kira round-trips frontmatter it does not understand is true for scalars and scalar lists and false for anything structured. Verified both directions by the durability review agent: a flat scalar and a flat list survive a move; a nested map does not.

This matters more than it looks because it is the input to KIRA-163: a ticket written by a newer kira that uses a structured field becomes unparseable to an older one, which then silently allocates a duplicate number.

Note codec.captureExtras already rejects a mapping-valued key with a ParseError rather than dropping it silently, which is the strictly safer half. The gap is that there is no representation for it at all, so a newer kira cannot use one without breaking older readers.

Decide before 1.0 whether the format promises to carry arbitrary YAML or explicitly promises only scalars and scalar lists. Either is defensible; leaving it undecided is not, because 1.0 is where the promise gets frozen.
<!-- /kira:comment -->
