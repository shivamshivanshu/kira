package core

import (
	"strings"
	"testing"

	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func TestMoveSubjectHandlesPercentInPrefix(t *testing.T) {
	s, cfg, repo := stagedFixture(t)
	cfg.Commit.SubjectPrefix = "100% "

	res, err := s.Create(cfg, CreateOpts{Type: "ticket", Title: "T", NoEdit: true})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := s.Move(cfg, res.Number, "IN_PROGRESS", MoveOpts{}); err != nil {
		t.Fatalf("move: %v", err)
	}

	msg, err := repo.Output("log", "-1", "--format=%B")
	if err != nil {
		t.Fatalf("git log: %v", err)
	}
	if strings.Contains(msg, "MISSING") {
		t.Fatalf("commit subject corrupted by a %%-containing prefix: %q", msg)
	}
	want := "100% " + res.Number + " state TODO -> IN_PROGRESS"
	if !strings.Contains(msg, want) {
		t.Errorf("commit message = %q, want it to contain %q", msg, want)
	}
}

// A batch must validate item N against the state left behind by item N-1's
// own commit, not the snapshot loaded before the batch started: otherwise a
// WIP limit sees stale occupancy and lets the batch overrun it.
func TestBatchMoveWipLimitSeesEarlierItemInSameBatch(t *testing.T) {
	s, cfg, _ := stagedFixture(t)
	wf := cfg.Workflows["ticket"]
	wf.WipPolicy = datamodel.WipBlock
	for i, st := range wf.States {
		if st.Key == "IN_PROGRESS" {
			wf.States[i].Wip = 1
		}
	}
	cfg.Workflows["ticket"] = wf

	a, err := s.Create(cfg, CreateOpts{Type: "ticket", Title: "a", NoEdit: true})
	if err != nil {
		t.Fatalf("create a: %v", err)
	}
	bTicket, err := s.Create(cfg, CreateOpts{Type: "ticket", Title: "b", NoEdit: true})
	if err != nil {
		t.Fatalf("create b: %v", err)
	}

	batch, err := s.BeginBatch(cfg)
	if err != nil {
		t.Fatalf("begin batch: %v", err)
	}
	defer batch.Close()

	if _, err := batch.Move(a.Number, "IN_PROGRESS", MoveOpts{}); err != nil {
		t.Fatalf("first move into the WIP-limited state: %v", err)
	}
	_, err = batch.Move(bTicket.Number, "IN_PROGRESS", MoveOpts{})
	if err == nil {
		t.Fatalf("second move should be blocked: WIP limit is 1 and the first move already occupies it")
	}
	if !strings.Contains(err.Error(), "over its WIP limit") {
		t.Fatalf("err = %v, want a WIP-limit error", err)
	}
}
