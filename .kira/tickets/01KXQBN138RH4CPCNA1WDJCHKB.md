---
id: 01KXQBN138RH4CPCNA1WDJCHKB
number: DX-12
aliases: []
type: ticket
subtype: task
title: "Add CLAUDE.md with project, package map, and CI/CD context"
state: DONE
resolution: done
priority: P3
owner: shivam
reporter: shivam
labels: []
epic: null
blocked_by: []
created: 2026-07-17T11:49:05+05:30
updated: 2026-07-17T11:49:11+05:30
---

## Description

Added a repo-root `CLAUDE.md` covering: project summary, the fact that this
repo dogfoods kira for its own tickets (boards, workflow states, auto-commit
mode), build/test/lint commands, a dependency-ordered map of every
`internal/` package plus `cmd/`, `schema/`, `tests/`, the CI/CD pipeline
(`build-test` matrix, `lint`, `tag`, `release`/GoReleaser, perf workflow),
and three gotchas hit this session (no Go `-O2`/`-O3` equivalent, don't
render timestamps via `time.Local()`, resolve symlinks before `filepath.Rel`
against a git toplevel).

## Acceptance criteria

- CLAUDE.md accurately reflects current package boundaries and CI jobs.

## Comments
