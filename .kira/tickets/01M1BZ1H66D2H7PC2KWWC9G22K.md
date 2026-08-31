---
id: 01M1BZ1H66D2H7PC2KWWC9G22K
number: KIRA-135
aliases: []
type: ticket
title: "Platform hardening: rename detection, entityschema placement validation, WIP doctor check, misc"
state: DONE
priority: P3
labels: []
epic: null
blocked_by: []
created: 2026-08-31T15:09:51+02:00
updated: 2026-08-31T23:51:27+02:00
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01M1BZ1HEE7Y201GCG8PMGNPT3 author=Shivam-Shivanshu ts=2026-08-31T15:09:52+02:00 -->
Grouped from the platform contextual and blind DX-1 reviews.
(a) gitx/staged.go:4 uses --diff-filter=ACM; rename detection is on by default so an R entry matches no letter and ValidateStaged never sees it. git mv a ticket to a stray name and the pre-commit hook passes where the same content added fresh is blocked. Fix: add --no-renames.
(b) entityschema/loader.go:128-173 checkSchemaShape never validates Placement or the body/Section pairing. A schema with 'placement: front-matter' loads clean, the field is silently inert, edit rejects it as undeclared and doctor warns 'unknown field' - no signal points at the typo. Fix: reject unknown placements, require section when placement is body.
(c) doctor has no WIP-limit rule though the write path enforces one (core/move.go:111,153-169). A hand edit, a three-way merge, or sync auto-resolve can leave a wip-capped state over its limit permanently, and doctor reports clean.
(d) automation/fire.go:36 formats the payload timestamp with time.Now(), bypassing the timex seam - nondeterministic hook-stdin goldens.
(e) gitx/index.go:110-122 parsePorcelainPaths splits on ' -> ' before unquoting, so a quoted path containing that sequence is mangled.
(f) termx/termx.go:37-46 ReadLineDefault makes a fresh bufio.Reader per call, dropping buffered stdin between consecutive prompts (piped 'y\ny\n' answers only the first).
(g) internal/config/templates/config.yaml documents editor precedence as EDITOR then VISUAL; editorx.Command:44-51 checks VISUAL first.
(h) gitx rev/branch wrappers pass caller strings positionally with no -- guard; seed.Run stages .kira then commits with no pathspec, sweeping unrelated staged changes; testutil.KiraBinary leaks a temp dir per test process.
<!-- /kira:comment -->
