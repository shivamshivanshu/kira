---
id: 01KXH34FAZH6MAD2C0QRVTSWEM
number: CORE-31
aliases: []
type: ticket
subtype: bug
title: "find/rgx: flag ordering, -e pattern capture, scanner.Err, NUL-separated parsing"
state: IN_PROGRESS
priority: P2
labels: []
epic: null
blocked_by: []
created: 2026-07-15T01:24:47+05:30
updated: 2026-07-16T17:47:10+05:30
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01KXH34FGRD4X6ZD5B3RYSS24J author=Shivam-Shivanshu ts=2026-07-15T01:24:47+05:30 -->
- internal/rgx/rgx.go:38-40 prepends enforced flags before passthru: 'kira find --column pat' breaks row parsing and --heading makes every line a RowSeparator (core/find.go:128-130).
- core/find.go:59-95 lists -e/--regexp in rgFlagsTakingValue but never captures the value into Pattern — note 'kira find -e foo' actually fails on BOTH backends because cli/find.go:27 raises 'a search pattern is required' before Store.Find runs; '='-attached forms bypass the map. Fix: capture -e/--regexp (and =-forms) into fa.Pattern and fix the CLI-layer check.
- rgx.go:60-70 and find.go:170-178 both exit the scan loop on ErrTooLong returning partial results with nil error — check scanner.Err in both scanners (or bytes.Split the in-memory output).
- rgx.go:44-56 discards stdout on exit 2 though ripgrep prints matches before erroring — return partial results with the error.
- lineRE (rgx.go:26) couples the wrapper to .md paths, accepts mixed separators, discards the Atoi error; non-parses fall to Line{Text: raw} with Path=="". Fix: have Search own ordering (passthru + enforced flags + '--' + path) and pass --null, splitting at the first NUL — kills the .md coupling, separator laxness, and unquote ambiguity.
- Cleanups: drop the redundant matched bool; share the scanner buffer constants (rgx cannot import core — hoist into rgx or a shared package); note -q passes through (harmless today).

Verify: extend ParseLine tables with rejection cases; add TestSearch behind an Installed() skip.

Files: internal/rgx/rgx.go, internal/core/find.go, internal/cli/find.go
<!-- /kira:comment -->
