---
id: 01KXH346VEKQ0Q1CGHZCPM6KAV
number: CLI-4
aliases: []
type: ticket
subtype: bug
title: "fzfx: real fzf failures misclassified as user cancel — kira exits 0"
state: IN_PROGRESS
priority: P1
labels: []
epic: null
blocked_by: []
created: 2026-07-15T01:24:38+05:30
updated: 2026-07-15T02:07:01+05:30
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01KXH347151FY6MZT7RNRX5KQ5 author=Shivam-Shivanshu ts=2026-07-15T01:24:39+05:30 -->
internal/fzfx/fzfx.go:35-39 returns aborted=true for ANY ExitError; fzf's contract: 0=selected, 1=no match, 2=error, 130=interrupted. pickFzf (cli/discover.go:97-99) maps aborted to nil error, so real fzf failures (exit 2) become exit status 0 (fzf's own stderr text does print — fzfx.go:33 wires cmd.Stderr = os.Stderr — but the failure status is swallowed). Sibling rgx.Search (rgx.go:45-55) handles the identical problem correctly. No _test.go in the package. Adjacent hygiene: Pick's (string, bool, error) polarity is only discoverable from the body; --with-nth 1.. is a no-op; --prompt passed unconditionally (blank prompt); error wrapped %v not %w; hand-rolled Builder duplicates strings.Join.

Fix: switch on ee.ExitCode(): 1 and 130 -> exported ErrCancelled sentinel (drop the bool return; callers errors.Is), other codes -> error including the code, mirroring rgx. Extract classification into a pure func. Sweep: drop --with-nth, guard --prompt, %w wrap, strings.Join stdin.

Verify: unit-test the classifier with synthetic ExitErrors (sh -c 'exit N').

Files: internal/fzfx/fzfx.go, internal/cli/discover.go
<!-- /kira:comment -->
