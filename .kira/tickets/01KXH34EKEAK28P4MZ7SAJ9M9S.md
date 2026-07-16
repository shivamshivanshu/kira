---
id: 01KXH34EKEAK28P4MZ7SAJ9M9S
number: CORE-29
aliases: []
type: ticket
subtype: bug
title: "Automation validation batch: timeout<=0, dead match keys, trust-hash pinning"
state: REVIEW
priority: P2
labels: []
epic: null
blocked_by: []
created: 2026-07-15T01:24:46+05:30
updated: 2026-07-16T18:29:54+05:30
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01KXH34ESC3EADZ2S4GKK3FV6C author=Shivam-Shivanshu ts=2026-07-15T01:24:46+05:30 -->
- validate.go:208 checks only the parse error of timeout; fire.go:85-86 builds an already-expired context for d <= 0 (fire.go's 'timeout, _ :=' discard is safe since load validates; the live gap is d <= 0). Fix: reject d <= 0 in TimeoutDuration.
- match.go:13-21 fields are only populated for EventItemStateChanged (core/automation.go:69-74); validateAutomationHooks never inspects h.Match — dead-match set is exactly to/from on item.created plus any match field on sync.completed (match.type on item.created works; baseEvent sets ev.Type). Fix: reject never-populated match keys per event (or populate To/ToCategory for created events).
- trust.go:16 hashes json.Marshal(cfg.Automation) — Go field names, not a pinned contract; trust.go:26/31 compares byte-for-byte so an editor-added trailing newline breaks trust with only the generic nag. Fix: pin the hash with explicit json tags (or hash the raw yaml block) plus a fixed-config hash-stability test; TrimSpace in GrantedHash; unexport GrantedHash (one same-file caller).
- fire.go:28 prints len(matching) as 'hooks defined' with wrong pluralization — fix wording. core/automation.go:55 casts "sync" instead of a SourceSync constant — add it.

Verify: hash-stability test with a fixed config; validation table for match keys per event type.

Files: internal/config/validate.go, internal/datamodel/automation.go, internal/automation/fire.go, internal/automation/trust.go, internal/automation/match.go, internal/core/automation.go
<!-- /kira:comment -->
