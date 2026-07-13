package syncx_test

import (
	"testing"

	"github.com/shivamshivanshu/kira/internal/syncx"
)

func TestReportAddOrdered(t *testing.T) {
	r := &syncx.Report{}
	r.Add("pull", syncx.StepDone, "rebased")
	r.Add("reconcile", syncx.StepDone, "")
	if len(r.Steps) != 2 {
		t.Fatalf("want 2 steps, got %d", len(r.Steps))
	}
	if r.Steps[0].Name != "pull" || r.Steps[0].Detail != "rebased" {
		t.Fatalf("first step = %+v", r.Steps[0])
	}
	if r.Steps[1].Status != syncx.StepDone {
		t.Fatalf("second step status = %q", r.Steps[1].Status)
	}
}
