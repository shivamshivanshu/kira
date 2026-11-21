package core

import (
	"os"
	"slices"
	"testing"

	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func reviewRepo(t *testing.T) *Store {
	t.Helper()
	s := eventRepo(t)
	config := []byte("version: 2\nproject: {key: KIRA, name: review}\nboards: [{key: KIRA, name: Review, default: true}]\ngit: {landed_ref: landed}\n")
	if err := os.WriteFile(s.fs().ConfigPath(), config, 0o644); err != nil {
		t.Fatal(err)
	}
	return s
}

func TestReviewUsesCommittedHistoryAndConditionalCloses(t *testing.T) {
	s := reviewRepo(t)
	one, two, done, newer := eventTicket(), eventTicket(), eventTicket(), eventTicket()
	one.Aliases = []string{"KIRA-8", "KIRA-9"}
	two.ID, two.Number, two.Aliases = "01hzz0test0000000000000001", "KIRA-2", []string{"KIRA-1", "KIRA-8"}
	done.ID, done.Number, done.State = "01HZZ0TEST0000000000000002", "KIRA-3", "DONE"
	newer.ID, newer.Number = "01HZZ0TEST0000000000000003", "KIRA-4"
	for _, it := range []*datamodel.Item{one, two, done, newer} {
		if _, err := s.fs().WriteItem(it); err != nil {
			t.Fatal(err)
		}
	}
	gitRun(t, s, "", "add", "-A")
	gitRun(t, s, "2026-01-05T10:00:00Z", "commit", "-m", "baseline")
	gitRun(t, s, "", "switch", "-c", "topic")
	commitState(t, s, one, "IN_PROGRESS", "2026-01-06")
	commitState(t, s, one, "TODO", "2026-01-07")
	gitRun(t, s, "2026-01-08T10:00:00Z", "commit", "--allow-empty", "-m", "KIRA-2 landed work",
		"-m", "Kira-Closes: KIRA-2\nKira-Closes: KIRA-4\nKira-Closes: KIRA-9")
	gitRun(t, s, "", "branch", "landed")
	gitRun(t, s, "2026-01-07T19:00:00Z", "commit", "--allow-empty", "-m", "[[KIRA-2]] follow-up",
		"-m", "Kira-Ticket: KIRA-99\nKira-Closes: KIRA-1\nKira-Closes: KIRA-3\nKira-Closes: KIRA-4\nKira-Closes: KIRA-8")
	commitState(t, s, newer, "IN_PROGRESS", "2026-01-10")
	gitRun(t, s, "", "switch", "main")
	gitRun(t, s, "2026-01-11T10:00:00Z", "commit", "--allow-empty", "-m", "KIRA-1 base-only work")
	gitRun(t, s, "", "switch", "topic")
	if err := os.WriteFile(s.fs().ConfigPath(), []byte("dirty, malformed: ["), 0o644); err != nil {
		t.Fatal(err)
	}
	repo := s.repo()
	beforeStatus, err := repo.Output("status", "--porcelain")
	if err != nil {
		t.Fatal(err)
	}
	res, err := s.Review("main")
	if err != nil {
		t.Fatal(err)
	}
	if res.Base == res.MergeBase || res.LandedRef != "landed" {
		t.Fatalf("review refs = %s / %s / %s", res.Base, res.MergeBase, res.LandedRef)
	}
	if len(res.StateChanges) != 1 || res.StateChanges[0].Ticket.Number != "KIRA-4" || res.StateChanges[0].To != "IN_PROGRESS" {
		t.Fatalf("net changes = %+v", res.StateChanges)
	}
	if len(res.StateEvents) != 2 || res.StateEvents[0].Ticket.Number != "KIRA-1" || len(res.StateEvents[0].Events) != 2 {
		t.Fatalf("patch history = %+v", res.StateEvents)
	}
	if len(res.Linked) != 4 || len(res.Linked[0].Commits) != 2 || len(res.Linked[1].Commits) != 2 {
		t.Fatalf("linked = %+v", res.Linked)
	}
	if len(res.Unresolved) != 2 || res.Unresolved[0].Ref != "KIRA-99" || res.Unresolved[0].Ambiguous ||
		res.Unresolved[1].Ref != "KIRA-8" || !res.Unresolved[1].Ambiguous {
		t.Fatalf("unresolved = %+v", res.Unresolved)
	}
	expected := []string{
		datamodel.ReviewCloseEligible, datamodel.ReviewCloseAlreadyLanded,
		datamodel.ReviewCloseAlreadyClosed, datamodel.ReviewCloseTicketNewer,
	}
	var outcomes []string
	for _, close := range res.Closes {
		outcomes = append(outcomes, close.Outcome)
	}
	if !slices.Equal(outcomes, expected) || res.Closes[0].Target != "DONE" {
		t.Fatalf("closes = %+v", res.Closes)
	}
	afterStatus, err := repo.Output("status", "--porcelain")
	if err != nil || afterStatus != beforeStatus {
		t.Fatalf("status changed from %q to %q (err %v)", beforeStatus, afterStatus, err)
	}
	head, err := repo.ResolveTreeish("HEAD")
	if err != nil || head != res.Head {
		t.Fatalf("HEAD changed to %q (err %v)", head, err)
	}
	if _, err := os.Stat(s.fs().CacheDir()); !os.IsNotExist(err) {
		t.Fatalf("review created cache directory (stat err %v)", err)
	}
}

func TestReviewIncludesStateIntroducedByMergeResolution(t *testing.T) {
	s := reviewRepo(t)
	it := eventTicket()
	commitState(t, s, it, "TODO", "2026-01-05")
	gitRun(t, s, "", "switch", "-c", "topic")
	commitState(t, s, it, "IN_PROGRESS", "2026-01-06")
	gitRun(t, s, "", "switch", "-c", "side", "main")
	commitState(t, s, it, "REVIEW", "2026-01-07")
	gitRun(t, s, "", "switch", "topic")
	if _, err := s.repo().Output("merge", "--no-commit", "side"); err == nil {
		t.Fatal("expected conflict on ticket state")
	}
	commitState(t, s, it, "DONE", "2026-01-08")

	res, err := s.Review("main")
	if err != nil {
		t.Fatal(err)
	}
	if len(res.StateChanges) != 1 || res.StateChanges[0].From != "TODO" || res.StateChanges[0].To != "DONE" {
		t.Fatalf("merge-only state absent from endpoint changes: %+v", res.StateChanges)
	}
	for _, history := range res.StateEvents {
		for _, event := range history.Events {
			if event.New == "DONE" {
				t.Fatalf("fixture unexpectedly reported merge-only state in patch history: %+v", event)
			}
		}
	}
}

func TestReviewPreviewsLastDoneStateFromCommittedWorkflow(t *testing.T) {
	s := reviewRepo(t)
	it := eventTicket()
	it.Subtype = new("bug")
	commitState(t, s, it, "TODO", "2026-01-05")
	gitRun(t, s, "", "switch", "-c", "topic")
	path := s.fs().ConfigPath()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(data, `ticket_states:
  bug:
    - {name: Ready, category: todo}
    - {name: Fixed, category: done}
    - {name: Verified, category: done}
`...), 0o644); err != nil {
		t.Fatal(err)
	}
	commitState(t, s, it, "Ready", "2026-01-06")
	gitRun(t, s, "2026-01-07T10:00:00Z", "commit", "--allow-empty", "-m", "fix", "-m", "Kira-Closes: KIRA-1")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}

	review, err := s.Review("main")
	if err != nil {
		t.Fatal(err)
	}
	if len(review.Closes) != 1 || review.Closes[0].Outcome != datamodel.ReviewCloseEligible || review.Closes[0].Target != "Verified" {
		t.Fatalf("close preview = %+v, want eligible for Verified", review.Closes)
	}
}
