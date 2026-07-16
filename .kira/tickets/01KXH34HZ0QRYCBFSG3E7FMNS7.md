---
id: 01KXH34HZ0QRYCBFSG3E7FMNS7
number: CORE-33
aliases: []
type: ticket
subtype: task
title: "datamodel cleanup: estimate error wording, ResolutionDropped constant, dead json tags, vocab tests"
state: DONE
priority: P3
labels: []
epic: null
blocked_by: []
created: 2026-07-15T01:24:50+05:30
updated: 2026-07-16T19:51:53+05:30
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01KXH34J4VV50WE6ZKD0A6QRNN author=Shivam-Shivanshu ts=2026-07-15T01:24:50+05:30 -->
- fields.go:150 '--field estimate: invalid number' surfaces from 'kira move' via transition set: values — reword to 'field %q: invalid number %q' (CLI adds --field context).
- internal/config/defaults.go:17/:50 hardcode literal "dropped" vs :52 datamodel.ResolutionDropped — a constant change fails the default config's own validation with no compiler catch; use the constant at both sites.
- Item/Config are never JSON-marshaled (results.go is the sole contract): drop the dead json:"-" tags (keep load-bearing yaml:"-").
- results.go:478-496: HooksStatusResult carries []HookState while install/validate carry []HookStatus — rename the Go types (JSON tags unchanged) or fold to one.
- Config.Sprint's only caller is its own HasSprint wrapper — fold in.
- Test gaps (targeted, not re-covering what config tests already hit): People.Canonical (git-alias commit attribution, sole caller store.go:97), Person map form with git: aliases, EnumVocab map form ({values, strict}), and FieldDescriptor.Set error/empty-clear/splitCSV-trim paths — table tests: {estimate abc -> error, "" -> nil, labels 'a, b,,' -> [a b]}.

Verify: new table tests green; grep confirms no remaining json.Marshal of Item/Config.

Files: internal/datamodel/fields.go, internal/datamodel/results.go, internal/datamodel/config.go, internal/datamodel/vocab.go, internal/config/defaults.go
<!-- /kira:comment -->
