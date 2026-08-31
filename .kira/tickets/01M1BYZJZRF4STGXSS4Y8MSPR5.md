---
id: 01M1BYZJZRF4STGXSS4Y8MSPR5
number: KIRA-131
aliases: []
type: ticket
title: "BoardMove and Reconcile bypass the mutation pipeline with no marker saying so"
state: DONE
priority: P2
labels: []
epic: null
blocked_by: []
created: 2026-08-31T15:08:48+02:00
updated: 2026-08-31T16:12:18+02:00
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01M1BYZK7PYK912S762Y0BVZVV author=Shivam-Shivanshu ts=2026-08-31T15:08:48+02:00 -->
core/boardmove.go:48-60 and core/reconcile.go:26-47 write and finalize directly: no validateMutation, no Updated bump, no ChangeSet, therefore no automation event - unlike every other mutation, which funnels through Mutate/commitMutation/commit. Plausibly deliberate (a renumber is not user activity), but nothing records that, so a guard added to the pipeline later will silently not cover these two.
Fix: either a one-line comment declaring the bypass intentional, or route them through commit with a dedicated ChangeSet kind.
<!-- /kira:comment -->
