package core

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/shivamshivanshu/kira/internal/datamodel"
)

// A batch must validate item N against the state left behind by item N-1's
// own commit, not the snapshot loaded before the batch started: otherwise a
// WIP limit sees stale occupancy and the batch overruns it silently.
func TestBatchMoveWipLimitSeesEarlierItemInSameBatch(t *testing.T) {
	s, cfg, _ := stagedFixture(t)
	ticketWorkflow, _ := datamodel.WorkflowOf(datamodel.TypeTicket)
	reviewState, _ := ticketWorkflow.StateByKey("REVIEW")
	numbers := make([]string, reviewState.Wip+1)
	for i := range numbers {
		res, err := s.Create(cfg, CreateOpts{Type: "ticket", Title: fmt.Sprintf("t%d", i), NoEdit: true})
		if err != nil {
			t.Fatalf("create %d: %v", i, err)
		}
		numbers[i] = res.Number
	}

	batch, err := s.BeginBatch(cfg)
	if err != nil {
		t.Fatalf("begin batch: %v", err)
	}
	defer batch.Close()

	for _, n := range numbers[:reviewState.Wip] {
		res, err := batch.Move(n, "REVIEW", MoveOpts{})
		if err != nil {
			t.Fatalf("move %s within the WIP limit: %v", n, err)
		}
		if len(res.Warnings) != 0 {
			t.Fatalf("move %s within the WIP limit warned: %v", n, res.Warnings)
		}
	}
	res, err := batch.Move(numbers[reviewState.Wip], "REVIEW", MoveOpts{})
	if err != nil {
		t.Fatalf("move past the WIP limit: %v", err)
	}
	if !slices.ContainsFunc(res.Warnings, func(w string) bool { return strings.Contains(w, "over its WIP limit") }) {
		t.Fatalf("warnings = %v, want a WIP-limit warning: the earlier moves already fill the state", res.Warnings)
	}
}
