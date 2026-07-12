package core_test

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/shivamshivanshu/kira/internal/codec"
	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/storage"
)

func wfAllows(wf datamodel.Workflow, from, to string) bool {
	for _, tr := range wf.Transitions[from] {
		if tr.To == to {
			return true
		}
	}
	return false
}

func wfDoneState(wf datamodel.Workflow, state string) bool {
	for _, st := range wf.States {
		if st.Key == state && st.Category == datamodel.CategoryDone {
			return true
		}
	}
	return false
}

func TestMoveTransitionMatrix(t *testing.T) {
	s, cfg := newStore(t)
	wf := cfg.Workflows[datamodel.TypeTicket]
	states := make([]string, len(wf.States))
	for i, st := range wf.States {
		states[i] = st.Key
	}
	res := mustCreate(t, s, cfg, "matrix")

	for _, from := range states {
		for _, to := range states {
			positionTo(t, s, cfg, res.Number, from)
			allowed := from == to || wfAllows(wf, from, to)
			opts := core.MoveOpts{}
			if wfDoneState(wf, to) {
				opts.Resolution = "done"
			}
			_, err := s.Move(cfg, res.Number, to, opts)
			if allowed {
				if err != nil {
					t.Errorf("move %s -> %s: unexpected error %v", from, to, err)
				}
				continue
			}
			if err == nil {
				t.Errorf("move %s -> %s: expected rejection, got success", from, to)
			}
			opts.Force = true
			if _, ferr := s.Move(cfg, res.Number, to, opts); ferr != nil {
				t.Errorf("forced move %s -> %s: %v", from, to, ferr)
			} else if got := stateOf(t, s, cfg, res.Number); got != to {
				t.Errorf("forced move %s -> %s: state = %s", from, to, got)
			}
		}
	}
}

func TestMoveInvalidStateAlwaysErrors(t *testing.T) {
	s, cfg := newStore(t)
	res := mustCreate(t, s, cfg, "bogus")
	for _, force := range []bool{false, true} {
		if _, err := s.Move(cfg, res.Number, "NOPE", core.MoveOpts{Force: force}); err == nil {
			t.Fatalf("move to invalid state (force=%v): expected error", force)
		}
	}
}

func TestMoveActivateWritesPointer(t *testing.T) {
	s, cfg := newStore(t)
	res := mustCreate(t, s, cfg, "activate")
	if _, err := s.Move(cfg, res.Number, "IN_PROGRESS", core.MoveOpts{Activate: true}); err != nil {
		t.Fatalf("Move --activate: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(storage.New(s.Root()).CacheDir(), "active"))
	if err != nil {
		t.Fatalf("read active pointer: %v", err)
	}
	if strings.TrimSpace(string(data)) != res.ID {
		t.Fatalf("active = %q, want %s", strings.TrimSpace(string(data)), res.ID)
	}
}

func TestCommentPureSuffix(t *testing.T) {
	s, cfg := newStore(t)
	res := mustCreate(t, s, cfg, "comment")
	path := filepath.Join(s.Root(), res.Path)

	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Comment(cfg, res.Number, core.CommentOpts{Message: "first note", HasMessage: true}); err != nil {
		t.Fatalf("Comment: %v", err)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(after), string(before)) {
		t.Fatalf("comment is not a pure suffix:\nbefore=%q\nafter=%q", before, after)
	}

	origItem, _ := codec.Parse(string(before))
	newItem, _ := codec.Parse(string(after))
	if origItem.Updated != newItem.Updated {
		t.Fatalf("comment bumped updated: %q -> %q", origItem.Updated, newItem.Updated)
	}
	comments := codec.ParseComments(newItem.Body)
	if len(comments) != 1 || comments[0].Body != "first note" || comments[0].Author != "tester" {
		t.Fatalf("parsed comments = %+v", comments)
	}

	mid, _ := os.ReadFile(path)
	if _, err := s.Comment(cfg, res.Number, core.CommentOpts{Message: "second", HasMessage: true}); err != nil {
		t.Fatalf("Comment 2: %v", err)
	}
	final, _ := os.ReadFile(path)
	if !strings.HasPrefix(string(final), string(mid)) {
		t.Fatal("second comment is not a pure suffix")
	}
}

func TestCommentEmptyRejected(t *testing.T) {
	s, cfg := newStore(t)
	res := mustCreate(t, s, cfg, "empty")
	if _, err := s.Comment(cfg, res.Number, core.CommentOpts{Message: "  ", HasMessage: true}); err == nil {
		t.Fatal("empty comment: expected error")
	}
}

func TestLinkTouchesOneFile(t *testing.T) {
	s, cfg := newStore(t)
	a := mustCreate(t, s, cfg, "A")
	b := mustCreate(t, s, cfg, "B")
	bPath := filepath.Join(s.Root(), b.Path)

	bBefore, err := os.ReadFile(bPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Link(cfg, a.Number, core.LinkOpts{Target: core.LinkBlockedBy, Ref: b.Number}); err != nil {
		t.Fatalf("Link: %v", err)
	}
	bAfter, err := os.ReadFile(bPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(bBefore) != string(bAfter) {
		t.Fatal("blocker's file changed; storage is not single-sided")
	}
	show, _ := s.Show(cfg, a.Number)
	if !slices.Contains(show.BlockedBy, b.ID) {
		t.Fatalf("A.blocked_by = %v, want to contain %s", show.BlockedBy, b.ID)
	}

	if _, err := s.Link(cfg, a.Number, core.LinkOpts{Target: core.LinkBlockedBy, Ref: b.Number, Remove: true}); err != nil {
		t.Fatalf("Link remove: %v", err)
	}
	show, _ = s.Show(cfg, a.Number)
	if len(show.BlockedBy) != 0 {
		t.Fatalf("A.blocked_by after remove = %v, want empty", show.BlockedBy)
	}
}

func TestLinkRejections(t *testing.T) {
	s, cfg := newStore(t)
	a := mustCreate(t, s, cfg, "A")
	epicRes, err := s.Create(cfg, core.CreateOpts{Type: datamodel.TypeEpic, Title: "E", NoEdit: true})
	if err != nil {
		t.Fatalf("Create epic: %v", err)
	}

	if _, err := s.Link(cfg, a.Number, core.LinkOpts{Target: core.LinkBlockedBy, Ref: a.Number}); err == nil {
		t.Fatal("self blocked-by: expected rejection")
	}
	if _, err := s.Link(cfg, a.Number, core.LinkOpts{Target: core.LinkEpic, Ref: a.Number}); err == nil {
		t.Fatal("self epic: expected rejection")
	}
	if _, err := s.Link(cfg, epicRes.Number, core.LinkOpts{Target: core.LinkEpic, Ref: a.Number}); err == nil {
		t.Fatal("epic-on-epic: expected rejection")
	}
	if _, err := s.Link(cfg, a.Number, core.LinkOpts{Target: core.LinkEpic, Ref: "KIRA-999"}); err == nil {
		t.Fatal("dangling epic: expected rejection")
	}
}

func TestLinkTyped(t *testing.T) {
	s, cfg := newStore(t)
	a := mustCreate(t, s, cfg, "A")
	b := mustCreate(t, s, cfg, "B")

	if _, err := s.Link(cfg, a.Number, core.LinkOpts{Target: core.LinkTyped, Type: datamodel.LinkRelates, Ref: b.Number}); err != nil {
		t.Fatalf("link relates: %v", err)
	}
	if _, err := s.Link(cfg, a.Number, core.LinkOpts{Target: core.LinkTyped, Type: datamodel.LinkDuplicateOf, Ref: b.Number}); err != nil {
		t.Fatalf("link duplicate-of: %v", err)
	}
	show, _ := s.Show(cfg, a.Number)
	if !slices.Equal(show.Links[datamodel.LinkRelates], []string{b.ID}) ||
		!slices.Equal(show.Links[datamodel.LinkDuplicateOf], []string{b.ID}) {
		t.Fatalf("links = %v, want %s in both types", show.Links, b.ID)
	}

	res, err := s.Link(cfg, a.Number, core.LinkOpts{Target: core.LinkTyped, Type: datamodel.LinkRelates, Ref: b.Number})
	if err != nil || len(res.Changed) != 0 {
		t.Fatalf("re-link: changed = %v, err = %v", res.Changed, err)
	}

	for _, typ := range datamodel.LinkTypes {
		if _, err := s.Link(cfg, a.Number, core.LinkOpts{Target: core.LinkTyped, Type: typ, Ref: b.Number, Remove: true}); err != nil {
			t.Fatalf("unlink %s: %v", typ, err)
		}
	}
	data, err := os.ReadFile(filepath.Join(s.Root(), a.Path))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "links") {
		t.Fatalf("links key must be omitted once empty:\n%s", data)
	}

	if _, err := s.Link(cfg, a.Number, core.LinkOpts{Target: core.LinkTyped, Type: datamodel.LinkRelates, Ref: a.Number}); err == nil {
		t.Fatal("self relates-link: expected rejection")
	}
	if _, err := s.Link(cfg, a.Number, core.LinkOpts{Target: core.LinkTyped, Type: datamodel.LinkRelates, Ref: "KIRA-999"}); err == nil {
		t.Fatal("dangling relates-link: expected rejection")
	}
}

func TestEditPathSelfLinkRejected(t *testing.T) {
	s, cfg := newStore(t)
	a := mustCreate(t, s, cfg, "A")

	editWith := func(mutate func(*datamodel.Item)) error {
		show, err := s.Show(cfg, a.Number)
		if err != nil {
			t.Fatal(err)
		}
		it, err := codec.Parse(mustReadItem(t, s, show.ID))
		if err != nil {
			t.Fatal(err)
		}
		mutate(it)
		path := filepath.Join(t.TempDir(), "edited.md")
		if err := os.WriteFile(path, []byte(codec.Serialize(it)), 0o644); err != nil {
			t.Fatal(err)
		}
		_, err = s.Edit(cfg, a.Number, core.EditOpts{FromFile: path})
		return err
	}

	if err := editWith(func(it *datamodel.Item) { it.BlockedBy = []string{a.ID} }); err == nil {
		t.Fatal("edit-path self blocked_by: expected rejection")
	}
	if err := editWith(func(it *datamodel.Item) { it.Epic = &a.ID }); err == nil {
		t.Fatal("edit-path self epic: expected rejection")
	}
}

func TestMoveRecordsResolution(t *testing.T) {
	s, cfg := newStore(t)
	res := mustCreate(t, s, cfg, "resolved")
	if _, err := s.Move(cfg, res.Number, "WONT_DO", core.MoveOpts{Resolution: "duplicate"}); err != nil {
		t.Fatalf("Move --resolution: %v", err)
	}
	show, _ := s.Show(cfg, res.Number)
	if show.Resolution == nil || *show.Resolution != "duplicate" {
		t.Fatalf("resolution = %v, want duplicate (--resolution outranks the state tag)", show.Resolution)
	}
	if show.State != "WONT_DO" {
		t.Fatalf("state = %s, want WONT_DO", show.State)
	}
}

func TestMoveRequireGuard(t *testing.T) {
	s, cfg := newStore(t)
	res := mustCreate(t, s, cfg, "guarded")
	positionTo(t, s, cfg, res.Number, "REVIEW")

	_, err := s.Move(cfg, res.Number, "DONE", core.MoveOpts{})
	if err == nil || !strings.Contains(err.Error(), "requires resolution") {
		t.Fatalf("REVIEW -> DONE without resolution: err = %v, want missing-field rejection", err)
	}
	if got := stateOf(t, s, cfg, res.Number); got != "REVIEW" {
		t.Fatalf("state after rejected move = %s, want REVIEW", got)
	}

	if _, err := s.Move(cfg, res.Number, "DONE", core.MoveOpts{Resolution: "duplicate"}); err != nil {
		t.Fatalf("REVIEW -> DONE --resolution: %v", err)
	}
	show, _ := s.Show(cfg, res.Number)
	if show.Resolution == nil || *show.Resolution != "duplicate" {
		t.Fatalf("resolution = %v, want duplicate (--resolution outranks set:)", show.Resolution)
	}
}

func TestMoveRequireForceBypass(t *testing.T) {
	s, cfg := newStore(t)
	res := mustCreate(t, s, cfg, "forced")
	positionTo(t, s, cfg, res.Number, "REVIEW")

	if _, err := s.Move(cfg, res.Number, "DONE", core.MoveOpts{Force: true}); err != nil {
		t.Fatalf("forced REVIEW -> DONE: %v", err)
	}
	show, _ := s.Show(cfg, res.Number)
	if show.State != "DONE" {
		t.Fatalf("state = %s, want DONE", show.State)
	}
	if show.Resolution == nil || *show.Resolution != "done" {
		t.Fatalf("resolution = %v, want done (set: applies under --force)", show.Resolution)
	}
}

func TestMoveRequireMultiMissing(t *testing.T) {
	s, cfg := newStore(t)
	withTicketTransitions(cfg, "TODO", []datamodel.Transition{
		{To: "IN_PROGRESS", Require: []string{"owner", "due"}},
	})
	res := mustCreate(t, s, cfg, "multi")

	_, err := s.Move(cfg, res.Number, "IN_PROGRESS", core.MoveOpts{})
	if err == nil || !strings.Contains(err.Error(), "requires owner, due") {
		t.Fatalf("err = %v, want both missing fields listed", err)
	}

	if _, err := s.Edit(cfg, res.Number, core.EditOpts{Fields: []core.FieldEdit{{Key: "owner", Value: "shivam"}}}); err != nil {
		t.Fatalf("set owner: %v", err)
	}
	_, err = s.Move(cfg, res.Number, "IN_PROGRESS", core.MoveOpts{})
	if err == nil || !strings.Contains(err.Error(), "requires due") || strings.Contains(err.Error(), "owner") {
		t.Fatalf("err = %v, want only due listed", err)
	}

	if _, err := s.Edit(cfg, res.Number, core.EditOpts{Fields: []core.FieldEdit{{Key: "due", Value: "2026-07-20"}}}); err != nil {
		t.Fatalf("set due: %v", err)
	}
	if _, err := s.Move(cfg, res.Number, "IN_PROGRESS", core.MoveOpts{}); err != nil {
		t.Fatalf("move with all required fields set: %v", err)
	}
}

func TestMoveSetApplied(t *testing.T) {
	s, cfg := newStore(t)
	withTicketTransitions(cfg, "TODO", []datamodel.Transition{
		{To: "IN_PROGRESS", Set: map[string]string{"owner": "alice", "priority": "P1", "estimate": "5"}},
	})
	res := mustCreate(t, s, cfg, "stamped")

	if _, err := s.Move(cfg, res.Number, "IN_PROGRESS", core.MoveOpts{}); err != nil {
		t.Fatalf("move with set:: %v", err)
	}
	show, _ := s.Show(cfg, res.Number)
	if show.Owner == nil || *show.Owner != "alice" {
		t.Errorf("owner = %v, want alice", show.Owner)
	}
	if show.Priority == nil || *show.Priority != "P1" {
		t.Errorf("priority = %v, want P1", show.Priority)
	}
	if show.Estimate == nil || *show.Estimate != 5 {
		t.Errorf("estimate = %v, want 5", show.Estimate)
	}
}

func TestMoveSetVocabViolation(t *testing.T) {
	s, cfg := newStore(t)
	cfg.Labels.Strict = true
	withTicketTransitions(cfg, "TODO", []datamodel.Transition{
		{To: "IN_PROGRESS", Set: map[string]string{"priority": "P9"}},
	})
	res := mustCreate(t, s, cfg, "badset")

	_, err := s.Move(cfg, res.Number, "IN_PROGRESS", core.MoveOpts{})
	if err == nil || !strings.Contains(err.Error(), "priority") {
		t.Fatalf("err = %v, want priority vocabulary rejection", err)
	}
	if got := stateOf(t, s, cfg, res.Number); got != "TODO" {
		t.Fatalf("state after rejected set: = %s, want TODO", got)
	}
}

func TestMoveResolutionLifecycle(t *testing.T) {
	s, cfg := newStore(t)
	res := mustCreate(t, s, cfg, "lifecycle")

	wantResolution := func(step string, want string) {
		t.Helper()
		show, err := s.Show(cfg, res.Number)
		if err != nil {
			t.Fatalf("%s: Show: %v", step, err)
		}
		got := show.Resolution
		if want == "" {
			if got != nil {
				t.Fatalf("%s: resolution = %q, want cleared", step, *got)
			}
			return
		}
		if got == nil || *got != want {
			t.Fatalf("%s: resolution = %v, want %s", step, got, want)
		}
	}

	if _, err := s.Move(cfg, res.Number, "WONT_DO", core.MoveOpts{}); err != nil {
		t.Fatalf("TODO -> WONT_DO: %v", err)
	}
	wantResolution("tagged done entry", "dropped")

	if _, err := s.Move(cfg, res.Number, "TODO", core.MoveOpts{Force: true}); err != nil {
		t.Fatalf("WONT_DO -> TODO: %v", err)
	}
	wantResolution("reopen", "")

	positionTo(t, s, cfg, res.Number, "REVIEW")
	if _, err := s.Move(cfg, res.Number, "DONE", core.MoveOpts{Resolution: "done"}); err != nil {
		t.Fatalf("REVIEW -> DONE: %v", err)
	}
	if _, err := s.Move(cfg, res.Number, "WONT_DO", core.MoveOpts{Force: true}); err != nil {
		t.Fatalf("DONE -> WONT_DO: %v", err)
	}
	wantResolution("done -> done retag", "dropped")

	_, err := s.Move(cfg, res.Number, "TODO", core.MoveOpts{Force: true, Resolution: "done"})
	if err == nil || !strings.Contains(err.Error(), "done-category") {
		t.Fatalf("--resolution to non-done target: err = %v, want rejection", err)
	}
}

func TestMoveWipWarning(t *testing.T) {
	s, cfg := newStore(t)
	var nums []string
	for _, title := range []string{"w1", "w2", "w3", "w4"} {
		nums = append(nums, mustCreate(t, s, cfg, title).Number)
	}
	for i, num := range nums[:3] {
		mres, err := s.Move(cfg, num, "IN_PROGRESS", core.MoveOpts{})
		if err != nil {
			t.Fatalf("move %s: %v", num, err)
		}
		if len(mres.Warnings) != 0 {
			t.Fatalf("move %d of 3 (at limit): warnings = %v, want none", i+1, mres.Warnings)
		}
	}
	mres, err := s.Move(cfg, nums[3], "IN_PROGRESS", core.MoveOpts{})
	if err != nil {
		t.Fatalf("move over limit must not block: %v", err)
	}
	want := "IN_PROGRESS is over its WIP limit (4 > 3)"
	if len(mres.Warnings) != 1 || mres.Warnings[0] != want {
		t.Fatalf("warnings = %v, want [%s]", mres.Warnings, want)
	}
	if got := stateOf(t, s, cfg, nums[3]); got != "IN_PROGRESS" {
		t.Fatalf("state = %s, want IN_PROGRESS (advisory, never blocks)", got)
	}
}

func TestMoveWipCountsPerType(t *testing.T) {
	s, cfg := newStore(t)
	states := cfg.Workflows[datamodel.TypeTicket].States
	for i := range states {
		if states[i].Key == "DONE" {
			states[i].Wip = 1
		}
	}

	epic, err := s.Create(cfg, core.CreateOpts{Type: datamodel.TypeEpic, Title: "epic", NoEdit: true})
	if err != nil {
		t.Fatalf("create epic: %v", err)
	}
	for _, state := range []string{"ACTIVE", "DONE"} {
		if _, err := s.Move(cfg, epic.Number, state, core.MoveOpts{}); err != nil {
			t.Fatalf("move epic to %s: %v", state, err)
		}
	}

	first := mustCreate(t, s, cfg, "t1")
	positionTo(t, s, cfg, first.Number, "REVIEW")
	mres, err := s.Move(cfg, first.Number, "DONE", core.MoveOpts{Resolution: "done"})
	if err != nil {
		t.Fatalf("move t1: %v", err)
	}
	if len(mres.Warnings) != 0 {
		t.Fatalf("first ticket at limit: warnings = %v (epic in DONE must not count)", mres.Warnings)
	}

	second := mustCreate(t, s, cfg, "t2")
	positionTo(t, s, cfg, second.Number, "REVIEW")
	mres, err = s.Move(cfg, second.Number, "DONE", core.MoveOpts{Resolution: "done"})
	if err != nil {
		t.Fatalf("move t2: %v", err)
	}
	want := "DONE is over its WIP limit (2 > 1)"
	if len(mres.Warnings) != 1 || mres.Warnings[0] != want {
		t.Fatalf("warnings = %v, want [%s]", mres.Warnings, want)
	}
}

func TestAssignStrictBypass(t *testing.T) {
	s, cfg := newStore(t)
	cfg.People.Known = []string{"shivam", "alice"}
	cfg.People.Strict = true
	res := mustCreate(t, s, cfg, "assign")

	if _, err := s.Assign(cfg, res.Number, "mallory", core.AssignOpts{}); err == nil {
		t.Fatal("strict assign of unknown user: expected rejection")
	}
	if _, err := s.Assign(cfg, res.Number, "mallory", core.AssignOpts{Force: true}); err != nil {
		t.Fatalf("forced assign: %v", err)
	}
	if show, _ := s.Show(cfg, res.Number); show.Owner == nil || *show.Owner != "mallory" {
		t.Fatalf("owner = %v, want mallory", show.Owner)
	}
	fresh := mustCreate(t, s, cfg, "reporter")
	if _, err := s.Assign(cfg, fresh.Number, "alice", core.AssignOpts{Reporter: true}); err != nil {
		t.Fatalf("assign reporter: %v", err)
	}
	if show, _ := s.Show(cfg, fresh.Number); show.Reporter == nil || *show.Reporter != "alice" {
		t.Fatalf("reporter = %v, want alice", show.Reporter)
	}
}
