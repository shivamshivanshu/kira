---
id: 01KXH343XY053WEW24MVR4WFER
number: CORE-13
aliases: []
type: ticket
subtype: bug
title: "links field Get renders only keys — invisible link-target changes in diff/automation/merge prompts"
state: DONE
resolution: done
priority: P1
labels: []
epic: null
blocked_by: []
created: 2026-07-15T01:24:35+05:30
updated: 2026-07-15T02:07:01+05:30
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01KXH3443SYSSKDFSTMR12JFYR author=Shivam-Shivanshu ts=2026-07-15T01:24:36+05:30 -->
internal/datamodel/fields.go:127 Get: strings.Join(sorted keys) — for links{relates:[A]} -> {relates:[B]}, core/diff.go:106 emits FieldChange{From:"relates",To:"relates"}, core/automation.go:100 emits Change{Old==New} for a field reported changed, and core/resolve.go:125 prompts '[o]urs=relates [t]heirs=relates' — indistinguishable merge choices. fields.go:128 Copy: dst.Links = src.Links shares the map[string][]string, unlike listField's slices.Clone (pinned by TestFieldDescriptorCopyClonesList) and core/edit.go cloneItem — latent (only call sites are resolve.go pickFields :129/:133 where the source is discarded), fix as hazard hygiene.

Fix: render targets per type in Get (e.g. 'relates:[01B 01C]' over sorted keys) — check contract goldens / automation PayloadVersion first since the string feeds the versioned payload; deep-copy in Copy (maps.Clone + per-slice slices.Clone).

Verify: extend the descriptor tests to links Get and Copy.

Files: internal/datamodel/fields.go, internal/datamodel/fields_test.go
<!-- /kira:comment -->
