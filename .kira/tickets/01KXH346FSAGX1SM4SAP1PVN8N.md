---
id: 01KXH346FSAGX1SM4SAP1PVN8N
number: CLI-3
aliases: []
type: ticket
subtype: bug
title: "Doctor: honor hooksPath/worktrees via repo.GitPath; move env gathering behind core"
state: TODO
priority: P1
labels: []
epic: null
blocked_by: []
created: 2026-07-15T01:24:38+05:30
updated: 2026-07-15T01:24:38+05:30
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01KXH346NH3VDS7B0PJX6XZSW2 author=Shivam-Shivanshu ts=2026-07-15T01:24:38+05:30 -->
Correctness: installedHooks (internal/cli/doctorenv.go:91-103) reads gitDir/hooks/<name> and TicketAttrRegistered (:33) reads gitDir/info/attributes from a local 'rev-parse --git-dir', while core honors hooksPath (core/hooks.go:195-215; 'hooks install --into-hooks-path' exists) and gitx.Repo.GitPath resolves linked worktrees — husky/lefthook repos and worktrees get 'hooks missing' from doctor while 'kira hooks status' says installed.

Layering: cli.go:1 declares 'every command is a thin argv/flag adapter over internal/core', yet gatherEnv spawns gitx, classifies hooks, probes index; readTicketFiles (doctor.go:41-66) re-implements storage LoadAll's directory scan, ULID filename filter, and the exact errx.User("reading tickets: %v") message (returns raw content where LoadAll decodes — near-duplicate, not literal); validate.go:24-46 reuses readTicketFiles (not gatherEnv).

Fix: replace gitDir()+manual joins with repo.GitPath("hooks")/GitPath("info/attributes"). Move readTicketFiles and gatherEnv behind core (s.DoctorReport(cfg)/s.ValidateFiles(...) or doctor.Gather(fs, repo)), adding a storage (s *FS) ItemFilenames()/RawItems() helper LoadAll also builds on; cli keeps flag parsing and rendering only.

Verify: test with core.hooksPath set (and in a linked worktree) asserting doctor and 'hooks status' agree.

Files: internal/cli/doctorenv.go, internal/cli/doctor.go, internal/cli/validate.go, internal/core/hooks.go, internal/storage/read.go, internal/doctor
<!-- /kira:comment -->
