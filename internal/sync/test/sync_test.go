package sync_test

import (
	"testing"

	syncpkg "github.com/shivamshivanshu/kira/internal/sync"
)

func TestReportAddOrdered(t *testing.T) {
	r := &syncpkg.Report{}
	r.Add("pull", syncpkg.StepDone, "rebased")
	r.Add("reconcile", syncpkg.StepDone, "")
	if len(r.Steps) != 2 {
		t.Fatalf("want 2 steps, got %d", len(r.Steps))
	}
	if r.Steps[0].Name != "pull" || r.Steps[0].Detail != "rebased" {
		t.Fatalf("first step = %+v", r.Steps[0])
	}
	if r.Steps[1].Status != syncpkg.StepDone {
		t.Fatalf("second step status = %q", r.Steps[1].Status)
	}
}

func TestNoopReindexerReportsSkipped(t *testing.T) {
	step := syncpkg.NoopReindexer{}.Reindex()
	if step.Name != "reindex" || step.Status != syncpkg.StepSkipped {
		t.Fatalf("noop reindex step = %+v, want skipped", step)
	}
}
