---
id: 01M1EVCH7MTBPERY0SZQ2JG0X0
number: KIRA-162
aliases: []
type: ticket
title: "Bundle entityschema's injected decoders behind one parameter"
state: TODO
priority: P3
labels: []
epic: null
blocked_by: []
created: 2026-09-01T18:03:41+02:00
updated: 2026-09-01T18:03:41+02:00
---

## Description

## Acceptance criteria

<!-- kira:comment id=01M1EVDABZX5C8ET44N7WD0BTE author=Shivam-Shivanshu ts=2026-09-01T18:04:07+02:00 -->
ProjectItem, CustomFields and ValidateExtras each grew a second injected decoder, and all 21 call sites pass the identical pair codec.DecodeExtraScalar, codec.DecodeScalar. A third decoder would force the same 21-file sweep again.

Two reviewers converged on this independently. Either shape works and both keep entityschema free of a codec import, since Go interfaces are structural:

  type ExtraDecoder interface { DecodeExtraScalar(string) any; DecodeExtraList([]string) []string }
  // or
  type Decoders struct { Scalar func(string) any; ListElement func(string) string }

Confirmed safe as a purely mechanical sweep: no test anywhere defines a stub decoder, so collapsing to a concrete codec.Decoder{} costs no test flexibility. The only non-literal call sites are ValidateExtras forwarding its own params, which the change absorbs.

Deferred from the review pass because it touches 21 files the reviewed commits never did.
<!-- /kira:comment -->
