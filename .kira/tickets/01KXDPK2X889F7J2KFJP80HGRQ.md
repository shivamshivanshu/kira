---
id: 01KXDPK2X889F7J2KFJP80HGRQ
number: KIRA-56
aliases: []
type: ticket
title: "Bump go directive when 1.27 is stable; adopt errors.AsType across 11 sites"
state: WONT_DO
resolution: dropped
labels: []
epic: null
blocked_by: []
created: 2026-07-13T17:47:51+05:30
updated: 2026-07-14T00:27:56+05:30
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01KXEDFMJZERPT7S29S4ENPB64 author=Shivam-Shivanshu ts=2026-07-14T00:27:55+05:30 -->
WONT_DO per backlog-clearing directive: not implementable until Go 1.27 releases. Preserved scope: bump go directive in go.mod, adopt errors.AsType at the ~11 errors.As call sites (rg 'errors.As\(' internal/), re-evaluate green-tea GC + any new stdlib wins. Reopen or refile at 1.27 release.
<!-- /kira:comment -->
