---
id: 01KXH33Z81H5B9HBRX6YDE48VY
number: CORE-4
aliases: []
type: ticket
subtype: bug
title: "EmitList emits flow-unsafe plain scalars — silent list corruption"
state: DONE
resolution: done
priority: P1
labels: []
epic: null
blocked_by: []
created: 2026-07-15T01:24:31+05:30
updated: 2026-07-15T02:56:38+05:30
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01KXH33ZDKNX7K9WSVH737R27X author=Shivam-Shivanshu ts=2026-07-15T01:24:31+05:30 -->
EmitList (internal/codec/serialize.go:91-100) wraps EmitScalar output in '['+join+']', but EmitScalar decides plain-vs-quoted via yaml.Marshal in BLOCK context, where ',' is a legal plain scalar. Verified with yaml.v3: EmitList(["a, b"]) emits 'labels: [a, b]' which re-parses as TWO labels (silent data corruption); EmitList(["x]y"]) is unparseable. No charset validation upstream (core/create.go:318-319, core/draft.go:49). Property-test tokens (codec/test/property_test.go:14-19) contain no token that is plain-safe in block context but unsafe in flow (mid-string ',', ']', '['), so 2000 iterations never hit it.

Fix: emit lists via yaml.Node{Kind: SequenceNode, Style: FlowStyle} marshalled once (verified this round-trips); keep the '[]' fast path. Caution: yaml.v3's flow emitter line-wraps at ~80 columns — verify serialize idempotency for long lists.

Same pass, adjacent codec parse edges: frontmatterNodes (parse.go:96-106) last-wins on duplicate keys with no ParseError (Parse vs DecodeFrontmatter disagree); splitDocument (parse.go:90) reports 'missing closing fence' for empty frontmatter and no-final-newline files — report duplicate keys / non-mapping frontmatter as clear ParseErrors, accept the empty-frontmatter and terminal-fence shapes. Add a safe-plain charset fast path to EmitScalar (currently a full yaml.Marshal per scalar per write).

Verify: add 'a, b' / 'x]y' / 'x[y' to the property-test tokens; round-trip tests for the new parse shapes.

Files: internal/codec/serialize.go, internal/codec/parse.go, internal/codec/test/property_test.go
<!-- /kira:comment -->
