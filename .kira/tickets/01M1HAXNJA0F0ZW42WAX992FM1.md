---
id: 01M1HAXNJA0F0ZW42WAX992FM1
number: KIRA-170
aliases: []
type: ticket
title: "webui still hardcodes the epic type, so a declared container is invisible to it"
state: DONE
resolution: done
priority: P3
labels: []
parent: null
blocked_by: []
created: 2026-09-02T17:13:40+02:00
updated: 2026-11-13T17:19:13+05:30
---
CORE-63 made "container" a schema-declared capability and replaced nine
hardcoded `TypeEpic` comparisons in core, cli and doctor. It touched **zero**
webui files -- confirmed with `git show 557ac7c --stat`. I scoped webui out
because it had been declared independent, but independent styling is not the
same as webui disagreeing with core about which types group other items.

Remaining hardcoded sites:

- `internal/webui/page.go:111` and `:113` -- progress-bar gate for tree and board
- `internal/webui/templates/tree.html:24`, `board.html:16` -- progress-bar gate
- `internal/webui/templates/tree.html:16`, `detail.html:5` -- glyph choice
  `{{if eq $item.Type "epic"}}◇{{else}}○{{end}}`, the same hardcoded-glyph
  pattern CORE-55 removed from the TUI, which webui never received
- `internal/webui/templates/datalists.html:8` -- the parent picker only offers
  items literally typed `epic`, though CORE-63 removed that restriction in core

`filters.html:84` also mentions "epic" but is the URL query-parameter clear
link -- a web contract, correctly left alone.

Effect on a repo declaring a second container type (say `initiative`): core
computes its rollup correctly, because `EpicProgress` already reads
`entityschema.ContainerTypes`. webui shows a plain circle, no progress bar, and
will not offer it as a parent. Latent today only because `epic` is the sole
container in this repo.

`internal/webui` imports no `entityschema` by design -- it talks to `core`. So
the fix needs a small surface addition (container-ness exposed through `core`,
e.g. on the row/item shape webui already renders) and then all the sites drop
their literal check together.
