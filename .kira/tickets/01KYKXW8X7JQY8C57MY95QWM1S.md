---
id: 01KYKXW8X7JQY8C57MY95QWM1S
number: CLI-11
aliases: []
type: ticket
subtype: feature
title: "No way to materialize schemas or refresh config in an existing repo"
state: TODO
priority: P2
labels: []
epic: null
blocked_by: []
created: 2026-07-28T14:06:21+05:30
updated: 2026-07-28T14:06:21+05:30
---

An existing repo cannot adopt newly-shipped config keys or get its schema
files materialized without hand-editing `.kira/config.yaml`.

Two separate gaps, both reachable only through `kira init`:

1. `entityschema.WriteDefaults` materializes the built-in schemas into
   `.kira/schema/` and is careful never to overwrite an existing file --
   exactly right for an existing repo -- but it is only called from
   `core.initLocked`. A repo created before schema files existed has no
   command to get them. Copying `internal/entityschema/builtin/*.json` by
   hand is the current workaround.

2. `kira init --force` rewrites `config.yaml` from the scaffold
   unconditionally (the `files` map in `core.initLocked`), so it drops
   `boards:`, `sprints:`, `filters:`, and any customized vocabulary. In this
   repo that would orphan the number prefixes of 196 existing tickets, since
   `boardFindings` warns for a prefix with no configured board.

The practical consequence: when kira gains a config key, existing repos
silently keep running on defaults with no path to the documented, commented
scaffold short of a manual diff-and-merge. Hit for real adopting this
session's features into kira's own `.kira/`.

Fix direction: split the two. A standalone way to materialize schemas
(`kira schema init`, reusing WriteDefaults as-is), and a config refresh that
merges scaffold keys the file lacks while preserving every value already
set. The merge wants the same provenance test `kira config explain` already
uses -- a key whose value equals the default can be safely rewritten with
the newer commented form; one that differs must be kept.
