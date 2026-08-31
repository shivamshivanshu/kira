---
id: 01M1BY3M83GSSPXCC6KG0GG5EH
number: KIRA-108
aliases: []
type: ticket
title: "Automation trust grant is committable: arbitrary code execution on clone"
state: REVIEW
priority: P0
labels: []
epic: null
blocked_by: []
created: 2026-08-31T14:53:32+02:00
updated: 2026-08-31T15:17:02+02:00
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01M1BY47AQH15H5A47MKW76PC1 author=Shivam-Shivanshu ts=2026-08-31T14:53:51+02:00 -->
REPRODUCED end-to-end 2026-08-31 on main. Arbitrary command execution from cloning a hostile repo.

Mechanism: automation.Trusted(cacheDir, cfg) (internal/automation/trust.go:30) grants execution when a file at <cacheDir>/automation.trust contains sha256(json(cfg.Automation)). core passes s.fs().CacheDir() = .kira/.cache/ (internal/core/automation.go), which is INSIDE the git worktree. .kira/.gitignore lists .cache/ (written by core/init.go:66), but an attacker controls their own repo's ignore rules and can 'git add -f'. Nothing checks whether the trust marker is tracked.

Exact repro:
1. attacker: kira init; append an automation hook with 'run: touch /tmp/PWNED' to .kira/config.yaml; kira automation trust; git add -f .kira/.cache/automation.trust; commit.
2. victim: git clone; kira create ticket --title innocent
3. /tmp/PWNED exists. The victim ran no trust command.

Impact: any mutating kira command in a freshly cloned repo runs attacker-chosen shell via automation.Fire -> sh -c, with the victim's environment and credentials. Strictly worse than KIRA-107 (arbitrary file write); this is code execution. Kira's premise is that tickets and config travel inside the repo, so cloning an untrusted repo is the expected workflow, not an exotic one.

Fix must move the grant outside the worktree so a clone cannot carry it: store it under .git/ (via gitx GitPath, which is already used for hook installation and is not cloned) or in an XDG state dir keyed by repo identity. Belt and braces: refuse a trust marker that git reports as tracked.

Note the two DX-1 platform reviewers disagreed here: the contextual one recorded the trust file as safely gitignored, the blind one called the attack. The blind reading was right - gitignore stops accidents, not adversaries. Found by the DX-1 review program.
<!-- /kira:comment -->
