package contract

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBlockersContract(t *testing.T) {
	t.Parallel()
	dir := seededRepo(t)
	head := gitOutput(t, dir, "rev-parse", "HEAD")
	args := []string{"blockers", "KIRA-1"}
	jsonArgs := withJSON(args)
	out, stderr, code := kira(t, dir, jsonArgs...)
	if code != 0 || stderr != "" {
		t.Fatalf("blockers --json exited %d: %s", code, stderr)
	}
	checkSchemaConformance(t, out)
	got := scrub(out, dir)
	checkGolden(t, "blockers.json", got)
	checkStable(t, "blockers", dir, got, jsonArgs...)

	out, stderr, code = kira(t, dir, args...)
	if code != 0 || stderr != "" || !strings.Contains(out, "KIRA-2 -> KIRA-3") {
		t.Fatalf("blockers exited %d: stdout %q, stderr %q", code, out, stderr)
	}
	if got := gitOutput(t, dir, "rev-parse", "HEAD"); got != head {
		t.Errorf("HEAD moved from %s to %s", head, got)
	}
	if got := gitOutput(t, dir, "status", "--porcelain"); got != "" {
		t.Errorf("worktree changed: %s", got)
	}
	if _, err := os.Stat(filepath.Join(dir, ".kira", ".cache", "index.db")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("expected no index database, got %v", err)
	}
}

func TestBlockersRejectsAmbiguousNumber(t *testing.T) {
	t.Parallel()
	dir := seededRepo(t)
	renumberTicket(t, dir, "KIRA-3", "KIRA-2")
	_, stderr, code := kira(t, dir, "blockers", "KIRA-2")
	if code != 1 || !strings.Contains(stderr, "ambiguous") {
		t.Fatalf("blockers exited %d: %s", code, stderr)
	}
}
