package webui

import (
	"os/exec"
	"testing"
)

func TestJavaScriptState(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node is unavailable; run node --test testdata/*_test.mjs to check browser state helpers")
	}
	cmd := exec.CommandContext(t.Context(), node, "--test", "testdata/ui_state_test.mjs", "testdata/preferences_test.mjs")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("JavaScript state tests: %v\n%s", err, output)
	}
}
