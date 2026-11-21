---
id: 01M7Y0CV6ER6QQS6JJ6HC787AK
number: KIRA-171
aliases: []
type: ticket
title: "Speed up stats and improve init config readability"
state: TODO
labels: []
parent: null
blocked_by: []
created: 2026-11-21T08:24:01+05:30
updated: 2026-11-21T08:24:01+05:30
---

## Description

Cold stats starts a Git process per ticket on merged histories. Batch paths
where the first-parent history is equivalent, preserve per-ticket events,
and reduce allocation in patch parsing, snapshot matching, and index reads.
Have Git return paths relative to the project in the last-touch scan.

Move init config explanations onto their own lines, document useful settings,
and apply the layout to Kira's repository config with its existing values.

## Acceptance criteria

- Real-history stats and cached events match before and after.
- Cold stats starts fewer Git processes on Kira's history.
- Existing projects keep their effective settings and config edits preserve comments.
- Build, lint, and regression tests pass.
