package integration

import (
	"testing"

	"github.com/shivamshivanshu/kira/internal/core"
)

// The apply rebase backend refuses `--continue` when the auto-resolved file is
// byte-identical to the rebase parent (the remote won every conflicting
// field); sync must skip the emptied patch instead of spinning to
// "did not converge".
func TestSyncSkipsPatchEmptiedByAutoResolve(t *testing.T) {
	t.Parallel()
	bare := bareRemote(t)

	seed := cloneWorld(t, bare)
	initStore(t, seed.root)
	seed.write(matrixItem(ulidX, "KIRA-1", "X"))
	seed.commit("base")
	if _, err := seed.repo.Output("push", "-u", "origin", "main"); err != nil {
		t.Fatalf("push seed: %v", err)
	}

	ours := cloneWorld(t, bare)
	them := cloneWorld(t, bare)

	theirsIt := matrixItem(ulidX, "KIRA-1", "X")
	theirsIt.State, theirsIt.Updated = "DONE", tsLate
	them.write(theirsIt)
	them.commit("theirs")
	if _, err := them.repo.Output("push", "origin", "main"); err != nil {
		t.Fatalf("push theirs: %v", err)
	}

	oursIt := matrixItem(ulidX, "KIRA-1", "X")
	oursIt.State, oursIt.Updated = "REVIEW", tsEarly
	ours.write(oursIt)
	ours.commit("ours")
	if err := ours.repo.SetConfig("rebase.backend", "apply"); err != nil {
		t.Fatalf("set rebase.backend: %v", err)
	}

	s, cfg := discoverStore(t, ours.root)
	if _, err := s.Sync(cfg, core.SyncOpts{}); err != nil {
		t.Fatalf("sync: %v", err)
	}
	if ours.repo.RebaseInProgress() {
		t.Fatal("rebase left in progress after sync")
	}
	if got := ours.item(ulidX).State; got != "DONE" {
		t.Fatalf("state = %s, want DONE (remote side, later updated)", got)
	}
}
