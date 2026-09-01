---
id: 01M1BZFCH0MAZBYZC2EPVMYFZ7
number: KIRA-146
aliases: []
type: ticket
title: "webui has no user-level auth: any local account on the host has full dashboard control"
state: WONT_DO
resolution: dropped
priority: P2
labels: []
epic: null
blocked_by: []
created: 2026-08-31T15:17:25+02:00
updated: 2026-09-01T13:07:40+02:00
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01M1BZFCSCTQPT0SNQEBVTKQ49 author=Shivam-Shivanshu ts=2026-08-31T15:17:26+02:00 -->
From the pre-merge security review of the web dashboard. Loopback-only binding is the sole access control (webui/security.go:23-57 does Host and CSRF checks only; cli/server.go:66 help says only 'Only loopback addresses are supported').
Attack: on a shared or multi-user machine, another local user runs curl http://127.0.0.1:7432/, scrapes the CSRF token out of the HTML - it is a per-process capability handed to anyone who can connect - and then POSTs any action: edit or move tickets, drive the command bridge (sync --push, workon branch switch, config set commit.mode), all under the dashboard owner's git identity. templates/page.html:12 also leaks the absolute repo path via data-workspace.
Fix: print a URL carrying a random secret path segment (or set a cookie on first visit from that tokenized URL) and 403 anything without it. At minimum, state the single-user-machine assumption in kira server --help and the README.
<!-- /kira:comment -->
