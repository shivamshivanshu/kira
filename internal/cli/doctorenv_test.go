package cli

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestInstalledHooksDetectsPlainInvocation(t *testing.T) {
	t.Parallel()
	gitDir := t.TempDir()
	hooksDir := filepath.Join(gitDir, "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		t.Fatalf("mkdir hooks: %v", err)
	}
	body := "#!/bin/sh\nkira hooks post-merge \"$@\"\n"
	if err := os.WriteFile(filepath.Join(hooksDir, "post-merge"), []byte(body), 0o755); err != nil {
		t.Fatalf("write hook: %v", err)
	}

	got := installedHooks(gitDir, []string{"post-merge"})
	if !slices.Contains(got, "post-merge") {
		t.Fatalf("a plainly-installed hook invoking 'kira hooks post-merge' must be detected, got %v", got)
	}
}
