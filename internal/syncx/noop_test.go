package syncx

import "testing"

func TestNoopReindexerReportsSkipped(t *testing.T) {
	step := noopReindexer{}.Reindex()
	if step.Name != "reindex" || step.Status != StepSkipped {
		t.Fatalf("noop reindex step = %+v, want skipped", step)
	}
}
