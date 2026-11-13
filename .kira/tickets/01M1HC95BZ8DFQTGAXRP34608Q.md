---
id: 01M1HC95BZ8DFQTGAXRP34608Q
number: CORE-65
aliases: []
type: ticket
title: "An epic cannot be blocked by a ticket, which looks unintended"
state: DONE
resolution: done
priority: P3
labels: []
parent: null
blocked_by: []
created: 2026-09-02T17:37:25+02:00
updated: 2026-11-13T17:19:13+05:30
---
`blocked_by` declares a different ref target per type:

    ticket.json:  { "name": "blocked_by", "type": "ref", "target": "ticket", "list": true, "unique": true }
    epic.json:    { "name": "blocked_by", "type": "ref", "target": "epic",   "list": true, "unique": true }

`core.validateGraph`'s `checkTargetType` enforces `target.Type != f.Target`, so
an epic may only be blocked by another epic, and a ticket only by another
ticket. An epic blocked by a ticket -- a very ordinary situation -- is
rejected.

Nothing found so far suggests this is deliberate; it reads like each schema
copied the pattern with its own type name substituted, which is consistent with
the cross-type duplication described in the `extends` ticket.

Not yet verified by running the binary -- this is read from the schema
declarations and the enforcement path. Confirm before fixing.

If it is unintended, the question is what the target should be: any item type,
or a capability the way `parent` now uses `targets_container`.
