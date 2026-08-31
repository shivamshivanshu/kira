---
id: 01M1BYRZ0NTRPJARH977DQYFDJ
number: KIRA-112
aliases: []
type: ticket
title: "Alias shadows a live display number in the trailer scan: Closes can auto-close the wrong ticket"
state: TODO
priority: P1
labels: []
epic: null
blocked_by: []
created: 2026-08-31T15:05:11+02:00
updated: 2026-08-31T15:05:11+02:00
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01M1BYRZ8DEMYJ700AW0SF5ZES author=Shivam-Shivanshu ts=2026-08-31T15:05:11+02:00 -->
Found independently by BOTH the data blind and data contextual DX-1 reviewers - convergent finding.

internal/index/trailers.go:390-409 (numberToULID) builds one number->ULID map from two queries in the order items then aliases; the second overwrites on key collision. internal/id/resolve.go:94-99 (holdersOf) gives the OPPOSITE precedence: live numbers win over aliases. core/reconcile.go:34-37 confirms renumbering appends the old collided number to the loser's Aliases, so 'alias equals another item's live number' is the normal post-reconcile state.

Scenario: items A and B both hold KIRA-5; doctor reconciles so A keeps live KIRA-5 and B becomes KIRA-12 with alias KIRA-5. A later commit trailer 'Closes: KIRA-5' resolves to B in latestCloses and resolveItemRefs, while kira show KIRA-5 resolves to A. Auto-close closes the wrong ticket and commit links attach to the wrong item.

Fix: make live numbers win - either skip alias rows whose key already exists, or insert aliases first and live numbers second. Match Resolver.holdersOf precedence exactly.
<!-- /kira:comment -->
