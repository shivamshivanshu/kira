---
id: 01M1BYZCXMSJN14AEMFN81PBYV
number: KIRA-119
aliases: []
type: ticket
title: "kira resolve writes the item file before checking it is the conflicted path"
state: DONE
resolution: done
priority: P2
labels: []
epic: null
blocked_by: []
created: 2026-08-31T15:08:42+02:00
updated: 2026-09-01T10:25:58+02:00
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01M1BYZD5J6Y6J0225XWAH2J0F author=Shivam-Shivanshu ts=2026-08-31T15:08:42+02:00 -->
core/resolve.go:79-86 calls WriteItemRaw(res.Item.ID, ...) - writing to the canonical path for the FRONTMATTER id - and only then does 'if rel != path' mark the entry skipped. The write has already happened.
Scenario: a conflicted file whose filename ULID does not match its frontmatter id (hand-copied or corrupted item). Resolve clobbers the OTHER item's file, reports the conflict merely as skipped, and leaves the working tree silently modified.
Fix: compute the expected rel path and skip before writing.
<!-- /kira:comment -->
