package cli

import (
	"bytes"
	"strings"
	"testing"
)

// The TUI command bar bridges :discover through --non-interactive; without
// this guard it blocks on stdin (no fzf) or fights the alt-screen (fzf).
func TestDiscoverRejectedNonInteractive(t *testing.T) {
	dir := initFixture(t)
	root, _ := newRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetIn(strings.NewReader(""))
	root.SetArgs([]string{"-C", dir, "--non-interactive", "discover"})
	err := root.Execute()
	if err == nil {
		t.Fatalf("discover under --non-interactive succeeded, want error\n%s", out.String())
	}
	if !strings.Contains(err.Error(), "interactive picker") {
		t.Errorf("err = %v, want mention of the interactive picker", err)
	}
}

// --json must fail loudly rather than silently opening the alt-screen (`kira
// tui --json`) or printing human help with exit 0 (`kira --json`).
func TestTUIRejectsJSON(t *testing.T) {
	dir := initFixture(t)
	root, _ := newRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetIn(strings.NewReader(""))
	root.SetArgs([]string{"-C", dir, "--json", "tui"})
	err := root.Execute()
	if err == nil {
		t.Fatalf("kira --json tui succeeded, want a --json rejection\n%s", out.String())
	}
	if !strings.Contains(err.Error(), "--json is not supported") {
		t.Errorf("kira --json tui: err = %v, want a --json rejection", err)
	}
}

// Bare `kira --json` names no command for --json to apply to, so it prints
// help rather than blaming the tui it never tried to launch.
func TestBareJSONPrintsHelp(t *testing.T) {
	dir := initFixture(t)
	root, _ := newRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetIn(strings.NewReader(""))
	root.SetArgs([]string{"-C", dir, "--json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("kira --json: %v\n%s", err, out.String())
	}
	if !strings.Contains(out.String(), "Usage:") {
		t.Errorf("kira --json printed %q, want help", out.String())
	}
}
