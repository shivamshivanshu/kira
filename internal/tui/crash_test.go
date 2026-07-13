package tui

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHandleCrashRestoresAndReports(t *testing.T) {
	root := t.TempDir()
	var buf bytes.Buffer
	ce := handleCrash(root, crashInfo{value: "boom", stack: []byte("goroutine 1 [running]")}, &buf)

	out := buf.String()
	if !strings.HasPrefix(out, terminalRestore) {
		t.Errorf("output does not begin with terminal-restore sequence: %q", out)
	}
	if n := strings.Count(out, "\n"); n != 3 {
		t.Errorf("stderr has %d lines, want exactly 3", n)
	}
	if !strings.Contains(out, "kira crashed: boom") || !strings.Contains(out, "data is intact") {
		t.Errorf("missing required crash lines: %q", out)
	}
	if !strings.HasPrefix(ce.LogPath, filepath.Join(root, ".kira", ".local")) {
		t.Errorf("log path = %q, want under .kira/.local", ce.LogPath)
	}
	data, err := os.ReadFile(ce.LogPath)
	if err != nil || !strings.Contains(string(data), "boom") {
		t.Errorf("crash log unreadable or missing panic value: %v %q", err, data)
	}
}

func TestGuardRunCatchesSetupPanic(t *testing.T) {
	root := t.TempDir()
	var buf bytes.Buffer
	err := guardRun(root, &buf, func() error { panic("setup boom") })

	var ce *CrashError
	if !errors.As(err, &ce) {
		t.Fatalf("guardRun returned %v, want *CrashError from the recover funnel", err)
	}
	if n := strings.Count(buf.String(), "\n"); n != 3 {
		t.Errorf("stderr has %d lines, want exactly 3", n)
	}
	if !strings.Contains(buf.String(), "setup boom") {
		t.Errorf("crash report missing the setup panic value: %q", buf.String())
	}
}

func TestWriteCrashLogFallsBackToTempDir(t *testing.T) {
	blocker := filepath.Join(t.TempDir(), "afile")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	unmakeable := filepath.Join(blocker, "sub", "crash-x.log")
	path := writeCrashLog(unmakeable, crashInfo{value: "boom", stack: []byte("s")})
	t.Cleanup(func() { _ = os.Remove(path) })

	if !strings.HasPrefix(path, os.TempDir()) {
		t.Errorf("fallback path = %q, want under os.TempDir() %q", path, os.TempDir())
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("fallback log not written: %v", err)
	}
}
