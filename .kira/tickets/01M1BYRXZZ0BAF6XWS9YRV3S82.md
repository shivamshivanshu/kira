---
id: 01M1BYRXZZ0BAF6XWS9YRV3S82
number: KIRA-110
aliases: []
type: ticket
title: "Full-editor edit silently destroys comments appended during the editor session"
state: REVIEW
priority: P1
labels: []
epic: null
blocked_by: []
created: 2026-08-31T15:05:10+02:00
updated: 2026-08-31T16:12:03+02:00
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01M1BYRY86N41PFW3J1FJEJVQP author=Shivam-Shivanshu ts=2026-08-31T15:05:10+02:00 -->
Reproduced by the DX-1 core contextual review.

internal/core/edit.go:51-53 detects concurrent change with orig.Updated != snapshot. But Comment is a deliberate byte-suffix append that never bumps updated (comment.go:21-23), so the token is blind to it. commitMutation then writes the stale buffer body wholesale.

Repro: kira edit T-2 (editor open) -> teammate runs kira comment T-2 -m '...' successfully -> user saves a title change -> 'Edited T-2: title, body' and the comment is gone. No conflict raised.

Fix: snapshot the raw file bytes (or a hash) at editor open and conflict on inequality under the lock; or re-append comment blocks present in the reloaded original but absent from the buffer (codec already splits comment blocks).
<!-- /kira:comment -->
