package hooks_test

import (
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
