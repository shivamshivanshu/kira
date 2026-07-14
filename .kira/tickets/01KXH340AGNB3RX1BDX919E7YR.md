---
id: 01KXH340AGNB3RX1BDX919E7YR
number: CORE-7
aliases: []
type: ticket
subtype: bug
title: "edit --field state bypasses move guards and leaves stale resolution"
state: TODO
priority: P1
labels: []
epic: null
blocked_by: []
created: 2026-07-15T01:24:32+05:30
updated: 2026-07-15T01:24:32+05:30
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01KXH340G6HEJANJ4PVR9ZXBDQ author=Shivam-Shivanshu ts=2026-07-15T01:24:32+05:30 -->
datamodel/fields.go:40 gives state a default Set (.withoutSet() exists, used for blocked_by at :48), so applyFieldEdit (core/edit.go:153) accepts it; validateMutation (validate.go:37-40, state-existence check at validate.go:35-39) only checks the state exists. Move clears resolution when leaving done (move.go:63) and demands --force off-graph (move.go:36-41); the edit path does neither — 'edit --field state=TODO' on a DONE item leaves 'resolution: done', the exact invariant TestMoveResolutionLifecycle asserts. Meanwhile item.go:106 marks state mutable=false so MutableFields excludes it (config/validate.go:322) while EditableFields includes it — two hand-maintained source tables (datamodel/fields.go:37-54 vs item.go:99-122) disagree. cli/edit.go:32 sanctions out-of-band edits for resolution but offers no --state flag: the state hole is unintentional. Tests exploit the hole: internal/core/test/fixture_test.go:58, internal/core/test/mutations_test.go:408.

Fix: add .withoutSet() to the state descriptor so --field state is rejected with a hint to use 'kira move' (preferred), or route edit-path state changes through Move's guard chain honoring Force. Add a validateItem rule that resolution may only exist on done-category states. Then derive FrontmatterKeys/MutableFields from the Fields descriptors so one table owns per-field capability. Route the fixture positionTo helper through Move --force or a test seam (tests currently depend on the hole).

Verify: pin with a test that --field state is rejected (or guard-routed) and that resolution cannot survive on non-done states.

Files: internal/datamodel/fields.go, internal/datamodel/item.go, internal/core/edit.go, internal/core/validate.go, internal/core/test/fixture_test.go, internal/core/test/mutations_test.go
<!-- /kira:comment -->
