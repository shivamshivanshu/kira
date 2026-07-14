---
id: 01KXCWBH6SP32P622FKMYTMRJP
number: KIRA-38
aliases: []
type: ticket
title: "Move Transition UnmarshalYAML out of datamodel into config/codec"
state: WONT_DO
resolution: dropped
priority: P3
labels: []
epic: 01KXCWAN7PK346KK28DTB1BCRQ
blocked_by: []
created: 2026-07-13T10:09:21+05:30
updated: 2026-07-13T22:13:52+05:30
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01KXE5T53GH3K0KEMWBX2PVYHV author=Shivam-Shivanshu ts=2026-07-13T22:13:51+05:30 -->
yaml.v3 dispatches UnmarshalYAML via a method on Transition, so the decoder must live in the type's package; moving it out needs parallel wrapper decode types - bloat against the professionalize goals.
<!-- /kira:comment -->
