---
id: 01M1EAAMCE4TE1F9FGNFG36DTF
number: KIRA-153
aliases: []
type: ticket
title: "linkMerge drops an extra link type whose list is empty"
state: WONT_DO
resolution: dropped
priority: P2
labels: []
epic: null
blocked_by: []
created: 2026-09-01T13:05:33+02:00
updated: 2026-09-01T13:50:14+02:00
---

## Description

## Acceptance criteria

<!-- kira:comment id=01M1ECWE95EFR0VFH0W6ZJ5C9S author=Shivam-Shivanshu ts=2026-09-01T13:50:14+02:00 -->
Not a bug — the gate is unreachable. Verified by round-tripping both shapes through codec.Parse:

  tags: []                  -> ExtraLists["tags"] = []string{}   (preserved)
  links: {supersedes: []}   -> ExtraLinks           = nil          (dropped)

optLinks (internal/codec/parse.go:352) does 'if len(targets) == 0 { continue }', so an empty link list of any type is discarded at parse and never reaches Item.ExtraLinks. linkMerge's 'len(merged) > 0' gate can therefore only drop something that cannot exist in a parsed item.

This was filed by analogy with KIRA-115, which WAS real: extra *lists* do survive as an empty non-nil slice, so gating their presence on length genuinely deleted a key on unrelated merges. Extra *links* do not survive, so the same gate is inert here.

Noting the asymmetry rather than removing it: an empty extra list is arguably meaningful (an explicitly empty set, distinct from absent) whereas an empty link type carries nothing. Also note writeLinks emits extraLinks unconditionally while gating builtin links on length — also unreachable today, for the same reason.
<!-- /kira:comment -->
