---
id: 01M1BYRXGK2J7FAGG5TZ2KT6VV
number: KIRA-109
aliases: []
type: ticket
title: "edit --from-file writes an undeclared frontmatter key and locks the item out of all mutations"
state: IN_PROGRESS
priority: P1
labels: []
epic: null
blocked_by: []
created: 2026-08-31T15:05:09+02:00
updated: 2026-08-31T16:12:02+02:00
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01M1BYRXR15AM9GYJA29TWWE6E author=Shivam-Shivanshu ts=2026-08-31T15:05:09+02:00 -->
Regression from CORE-3 phases 3a/3c. Reproduced by the DX-1 core contextual review.

internal/core/edit.go (from-file path :92-103, editor path via parseFullItem) accepts any buffer. codec.Parse captures every capturable unknown key into Extras (codec/parse.go:68-75,246-254) and codec.Serialize emits all Extras unconditionally (serialize.go:42). guardWritable only ever runs on the LOADED ORIGINAL (mutate.go:27, edit.go:124), never on the incoming item, and entityschema.ValidateExtras validates only schema-DECLARED fields.

Repro: clean item -> kira edit T-1 --from-file f.md where f.md adds 'foo: bar' -> exit 0, 'Edited T-1: foo', file now contains foo: bar -> every later mutation fails exit 3 with 'this item uses fields from a newer kira: foo, hint: upgrade kira'. A typo in the full editor (priorty: high) does the same. Self-inflicted permanent lockout behind a misleading Env-class error.

Before Extras existed, serialize never emitted unknown keys, so this leak is new.

Fix: in validateMutation/finish, reject undeclared unknown keys on the UPDATED item (reuse hostileKeys) as a User-class invalid-item error naming the key, so the editor retry banner surfaces it in-buffer.
<!-- /kira:comment -->
