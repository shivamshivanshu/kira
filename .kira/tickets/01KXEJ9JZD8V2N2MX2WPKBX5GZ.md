---
id: 01KXEJ9JZD8V2N2MX2WPKBX5GZ
number: KIRA-81
aliases: []
type: ticket
subtype: feature
title: "Detached tracker: kira init --target <repo-root> (default .); tracker repo holds .kira, target repo is scanned; all git consumers split tracker-vs-target; zero migration for colocated"
state: WONT_DO
resolution: dropped
priority: P1
labels: [core]
epic: null
blocked_by: []
created: 2026-07-14T01:52:00+05:30
updated: 2026-07-14T21:12:51+05:30
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01KXGMQ57J7WZ620HHDKDDM8A7 author=Shivam-Shivanshu ts=2026-07-14T21:12:51+05:30 -->
WONT_DO (founder): a tracker repo outside the code repo sits off the remote sync path - staleness and invisible divergence when working with remotes, the same failure mode that killed git-notes (and the design doc's own strongest-against argument). Colocated stays the model. Survivors: KIRA-82 (kira commit, colocated) and KIRA-83 (reshaped to colocated hooks).
<!-- /kira:comment -->
