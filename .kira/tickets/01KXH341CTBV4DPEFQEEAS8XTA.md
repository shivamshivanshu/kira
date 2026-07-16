---
id: 01KXH341CTBV4DPEFQEEAS8XTA
number: CORE-8
aliases: []
type: ticket
subtype: bug
title: "Hook chainTail swallows user-hook failures and 127-aborts commits on missing shim"
state: DONE
priority: P1
labels: []
epic: null
blocked_by: []
created: 2026-07-15T01:24:33+05:30
updated: 2026-07-16T13:38:53+05:30
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01KXH341JH1KDXY19W8GNAK6V7 author=Shivam-Shivanshu ts=2026-07-15T01:24:33+05:30 -->
chainTail (internal/hooks/hooks.go:120-122) appends a bare '.kira/hooks/<name> "$@"' as the last line of a user hook: (a) sh continues past a failed user linter and the script exits with the shim's 0 — user pre-commit checks silently defeated; (b) on a checkout predating kira, sh exits 127 and git aborts chained commit-path hooks (prepare-commit-msg by default, pre-commit if installed; a 127 in post-merge is only noise since git ignores its status). Embedded scripts guard correctly with command -v (scripts/pre-commit:3). Third failure mode: a user script ending in 'exec' skips the chain block entirely. Status-masking bites user hooks relying on the last command's rc (no set -e).

Also: Invokes (:43-46) only matches 'kira hooks run'; StateOf (:142-157) classifies the pure '.kira/hooks/' shim (pinned pure by test/hooks_test.go:83) as StateForeign, so core/hooks.go:126-131 Chain()s a SECOND invocation; uninstall leaves the bare-shim form as 'left' (core/hooks.go:296 checks IsPureShim only under StateDrifted — same root cause). Unchain's no-newline branch trims a byte of user content when the block isn't terminal (:124-127). IsShellScript misparses '#!/usr/bin/env -S bash' (:92-94). No test executes a chained hook under sh.

Fix: rewrite chainTail to guard and preserve status — capture the user portion's rc first: 'rc=$?; if [ -x .kira/hooks/<n> ]; then .kira/hooks/<n> "$@" || exit; fi; exit $rc'. Recognize the shim form in Invokes/StateOf (or return StateDrifted when IsPureShim). Anchor Unchain's newline compensation to EOF; skip '-' fields after env in IsShellScript. Fold in API cleanups: single source for hook-name constants (cli/hooks.go:124-131), hoist delegationLineRe compile out of the IsPureShim loop, express Classify as a projection of StateOf, unexport the mutable Default slice, iterate hooks.Known() in the pinning test.

Verify: one behavioral test running Chain output under sh — failing user hook -> nonzero; missing shim -> 0; shim invoked exactly once.

Files: internal/hooks/hooks.go, internal/core/hooks.go, internal/cli/hooks.go, internal/hooks/test/hooks_test.go
<!-- /kira:comment -->
