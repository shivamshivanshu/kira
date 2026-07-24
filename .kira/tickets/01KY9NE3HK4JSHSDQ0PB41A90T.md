---
id: 01KY9NE3HK4JSHSDQ0PB41A90T
number: CORE-42
aliases: []
type: ticket
title: "Surface extras-only merge conflicts in arbitrated() reporting"
state: TODO
labels: []
epic: null
blocked_by: []
created: 2026-07-24T14:26:23+05:30
updated: 2026-07-24T14:27:03+05:30
---

## Description

`merge.arbitrated()` walks `datamodel.Fields`, which extra fields aren't
registered in, so an extras-only three-way conflict resolves via last-write-wins
with no `Result.Arbitrated` signal (no CLI/JSON surface) — unlike known fields.
Register extras (or otherwise report them) alongside the schema-driven field
registry in CORE-3 phase 3b.

## Acceptance criteria

## Comments
