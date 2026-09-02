---
id: 01M1HC951ZTX2MYTV6N0QP654Q
number: CORE-64
aliases: []
type: ticket
title: "Types cannot inherit, so every new type copies ticket's whole field list"
state: WONT_DO
resolution: dropped
priority: P3
labels: []
parent: null
blocked_by: []
created: 2026-09-02T17:37:25+02:00
updated: 2026-09-02T23:45:08+02:00
---
`.kira/schema/epic.json` is 36 lines and **17 of its 19 fields are
byte-identical to ticket.json**. Its genuine content is five things: a
different `state` set, a different `blocked_by` target, `container: true`,
`workflow: epic` and `label: epic`.

Schema overlays (the separate proposal) do not fix this. An overlay lets
`.kira/schema/epic.json` declare only what it overrides relative to
`builtin/epic.json` -- it removes *repo-copy* duplication. The duplication here
is *cross-type*: epic repeats ticket's fields.

Removing that needs type inheritance, e.g. `"extends": "ticket"`, after which
epic.json is roughly:

    {
      "name": "epic",
      "extends": "ticket",
      "workflow": "epic",
      "container": true,
      "fields": [ <the state field>, <the blocked_by field> ],
      "representation": { "label": "epic", "icon": "" }
    }

The cost of not having it grows with the goal: every user-defined type (bug,
feature, spike) copies the same 17 fields today, and each copy can rot
independently -- the same class of problem the icon rot in CORE-55 came from.

Design questions to settle first, not to guess:

- how a child overrides one inherited field versus replacing the list,
- how a child *removes* an inherited field,
- whether inheritance is single or multiple, and how deep chains resolve,
- interaction with schema overlays, since a repo file would then layer over a
  builtin that itself extends another type.
