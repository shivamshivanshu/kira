---
id: 01M1BYZDDENWX36500JCGFPNRR
number: KIRA-120
aliases: []
type: ticket
title: "kira init accepts invalid project keys from the prompt and from the derived default"
state: IN_PROGRESS
priority: P2
labels: []
epic: null
blocked_by: []
created: 2026-08-31T15:08:42+02:00
updated: 2026-08-31T16:12:08+02:00
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01M1BYZDNDER7EREQYXG25Q7GC author=Shivam-Shivanshu ts=2026-08-31T15:08:42+02:00 -->
core/init.go:44-56 - ValidBoardKey runs only on the --key flag, before prompting. The interactively typed key and the derived default are unvalidated, and deriveKey does not truncate to the pattern's 10-char cap. config/validate.go validates boards[].key but never project.key, so the post-scaffold Load check passes.
Repro: kira init with no --key in a directory named my-superlong-project -> project key MYSUPERLONGPROJECT (18 chars, violates ^[A-Z][A-Z0-9]{1,9}$). Scaffold succeeds; later board create/adoption fails with 'project.key is not a valid board key', and re-init needs --force. The interactive path also accepts lowercase and garbage.
Fix: validate after prompting and truncate in deriveKey, before writing files.
<!-- /kira:comment -->
