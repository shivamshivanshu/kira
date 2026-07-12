package core

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/shivamshivanshu/kira/internal/config"
	"github.com/shivamshivanshu/kira/internal/item"
)

// newStore initializes a repo + store + config for a mutation test.
func newStore(t *testing.T) (*Store, *config.Config) {
	t.Helper()
	root := initGitRepo(t)
	if _, err := Init(root, "KIRA", false); err != nil {
		t.Fatalf("Init: %v", err)
	}
	s, err := Discover(root)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	cfg, err := s.Config()
	if err != nil {
		t.Fatalf("Config: %v", err)
	}
	return s, cfg
}

func mustCreate(t *testing.T, s *Store, cfg *config.Config, title string) *CreateResult {
	t.Helper()
	res, err := s.Create(cfg, CreateOpts{Type: item.TypeTicket, Title: title, NoEdit: true})
	if err != nil {
		t.Fatalf("Create %q: %v", title, err)
	}
	return res
}

func stateOf(t *testing.T, s *Store, cfg *config.Config, ref string) string {
	t.Helper()
	show, err := s.Show(cfg, ref)
	if err != nil {
		t.Fatalf("Show %s: %v", ref, err)
	}
	return show.State
}

// TestMoveTransitionMatrix exercises the full adjacency matrix of the ticket
// workflow: every allowed edge succeeds, every off-graph move is rejected under
// enforce_transitions, and --force overrides the rejection (WP-1.1 verify).
func TestMoveTransitionMatrix(t *testing.T) {
	s, cfg := newStore(t)
	wf := cfg.Workflows[item.TypeTicket]
	states := make([]string, len(wf.States))
	for i, st := range wf.States {
		states[i] = st.Key
	}
	res := mustCreate(t, s, cfg, "matrix")

	for _, from := range states {
		for _, to := range states {
			positionTo(t, s, cfg, res.Number, from)
			allowed := from == to || transitionAllowed(wf, from, to)
			// Satisfy any require: guard on done-category targets so the
			// matrix stays a pure adjacency exercise; require enforcement has
			// its own tests (TestMoveRequireGuard).
			opts := MoveOpts{}
			if cat, _ := categoryOf(cfg, item.TypeTicket, to); cat == config.CategoryDone {
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
			// Re-position (the rejected move left state at `from`) then force it.
			opts.Force = true
			if _, ferr := s.Move(cfg, res.Number, to, opts); ferr != nil {
				t.Errorf("forced move %s -> %s: %v", from, to, ferr)
			} else if got := stateOf(t, s, cfg, res.Number); got != to {
				t.Errorf("forced move %s -> %s: state = %s", from, to, got)
			}
		}
	}
}

// TestMoveInvalidStateAlwaysErrors proves an unknown state name is rejected even
// under --force: --force bypasses only the adjacency check, never state
// existence (docs/design/04-cli.md move).
func TestMoveInvalidStateAlwaysErrors(t *testing.T) {
	s, cfg := newStore(t)
	res := mustCreate(t, s, cfg, "bogus")
	for _, force := range []bool{false, true} {
		if _, err := s.Move(cfg, res.Number, "NOPE", MoveOpts{Force: force}); err == nil {
			t.Fatalf("move to invalid state (force=%v): expected error", force)
		}
	}
}

// TestMoveActivateWritesPointer checks --activate records the item's ULID at
// .cache/active, gitignored local state.
func TestMoveActivateWritesPointer(t *testing.T) {
	s, cfg := newStore(t)
	res := mustCreate(t, s, cfg, "activate")
	if _, err := s.Move(cfg, res.Number, "IN_PROGRESS", MoveOpts{Activate: true}); err != nil {
		t.Fatalf("Move --activate: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(s.cacheDir(), "active"))
	if err != nil {
		t.Fatalf("read active pointer: %v", err)
	}
	if strings.TrimSpace(string(data)) != res.ID {
		t.Fatalf("active = %q, want %s", strings.TrimSpace(string(data)), res.ID)
	}
}

// TestCommentPureSuffix is the WP-1.2 golden guarantee: appending a comment
// leaves the prior file content as an exact byte prefix, and never rewrites
// frontmatter (the `updated` timestamp is untouched), so concurrent comments
// merge cleanly (docs/design/02-data-model.md §4).
func TestCommentPureSuffix(t *testing.T) {
	s, cfg := newStore(t)
	res := mustCreate(t, s, cfg, "comment")
	path := filepath.Join(s.Root(), res.Path)

	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Comment(cfg, res.Number, CommentOpts{Message: "first note", HasMessage: true}); err != nil {
		t.Fatalf("Comment: %v", err)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(after), string(before)) {
		t.Fatalf("comment is not a pure suffix:\nbefore=%q\nafter=%q", before, after)
	}

	origItem, _ := item.Parse(string(before))
	newItem, _ := item.Parse(string(after))
	if origItem.Updated != newItem.Updated {
		t.Fatalf("comment bumped updated: %q -> %q", origItem.Updated, newItem.Updated)
	}
	comments := item.ParseComments(newItem.Body)
	if len(comments) != 1 || comments[0].Body != "first note" || comments[0].Author != "tester" {
		t.Fatalf("parsed comments = %+v", comments)
	}

	// A second comment is also a pure suffix of the one-comment file.
	mid, _ := os.ReadFile(path)
	if _, err := s.Comment(cfg, res.Number, CommentOpts{Message: "second", HasMessage: true}); err != nil {
		t.Fatalf("Comment 2: %v", err)
	}
	final, _ := os.ReadFile(path)
	if !strings.HasPrefix(string(final), string(mid)) {
		t.Fatal("second comment is not a pure suffix")
	}
}

// TestCommentEmptyRejected checks an empty -m body is refused.
func TestCommentEmptyRejected(t *testing.T) {
	s, cfg := newStore(t)
	res := mustCreate(t, s, cfg, "empty")
	if _, err := s.Comment(cfg, res.Number, CommentOpts{Message: "  ", HasMessage: true}); err == nil {
		t.Fatal("empty comment: expected error")
	}
}

// TestLinkTouchesOneFile is the WP-1.2 single-sided-storage golden: linking A
// blocked-by B writes A's file only; B's file is byte-identical after the link
// (blocks is a derived inverse, never stored — docs/design/02-data-model.md §3).
func TestLinkTouchesOneFile(t *testing.T) {
	s, cfg := newStore(t)
	a := mustCreate(t, s, cfg, "A")
	b := mustCreate(t, s, cfg, "B")
	bPath := filepath.Join(s.Root(), b.Path)

	bBefore, err := os.ReadFile(bPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Link(cfg, a.Number, LinkOpts{Target: LinkBlockedBy, Ref: b.Number}); err != nil {
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

	// --remove is idempotent and also writes only A.
	if _, err := s.Link(cfg, a.Number, LinkOpts{Target: LinkBlockedBy, Ref: b.Number, Remove: true}); err != nil {
		t.Fatalf("Link remove: %v", err)
	}
	show, _ = s.Show(cfg, a.Number)
	if len(show.BlockedBy) != 0 {
		t.Fatalf("A.blocked_by after remove = %v, want empty", show.BlockedBy)
	}
}

// TestLinkRejections covers self-link and epic-on-epic (WP-1.2).
func TestLinkRejections(t *testing.T) {
	s, cfg := newStore(t)
	a := mustCreate(t, s, cfg, "A")
	epicRes, err := s.Create(cfg, CreateOpts{Type: item.TypeEpic, Title: "E", NoEdit: true})
	if err != nil {
		t.Fatalf("Create epic: %v", err)
	}

	if _, err := s.Link(cfg, a.Number, LinkOpts{Target: LinkBlockedBy, Ref: a.Number}); err == nil {
		t.Fatal("self blocked-by: expected rejection")
	}
	if _, err := s.Link(cfg, a.Number, LinkOpts{Target: LinkEpic, Ref: a.Number}); err == nil {
		t.Fatal("self epic: expected rejection")
	}
	if _, err := s.Link(cfg, epicRes.Number, LinkOpts{Target: LinkEpic, Ref: a.Number}); err == nil {
		t.Fatal("epic-on-epic: expected rejection")
	}
	// Dangling reference is rejected too.
	if _, err := s.Link(cfg, a.Number, LinkOpts{Target: LinkEpic, Ref: "KIRA-999"}); err == nil {
		t.Fatal("dangling epic: expected rejection")
	}
}

// TestVocabStrictWarn is the WP-1.3 table: an unknown vocabulary value is a hard
// error only under strict without --force, and a warning otherwise
// (docs/design/02-data-model.md §5).
func TestVocabStrictWarn(t *testing.T) {
	base := item.Item{ID: "X", Number: "KIRA-1", Type: item.TypeTicket, Title: "t", State: "TODO"}
	cases := []struct {
		name       string
		strict     bool
		owner      string
		force      bool
		wantErr    bool
		wantWarned bool
	}{
		{"known", true, "shivam", false, false, false},
		{"unknown-strict", true, "mallory", false, true, false},
		{"unknown-strict-force", true, "mallory", true, false, true},
		{"unknown-lenient", false, "mallory", false, false, true},
		{"known-lenient", false, "alice", false, false, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := config.Default()
			cfg.People.Strict = tc.strict
			it := base
			it.Owner = &tc.owner
			errs, warns := validateItem(cfg, &it, tc.force)
			if gotErr := len(errs) > 0; gotErr != tc.wantErr {
				t.Errorf("errs = %v, wantErr = %v", errs, tc.wantErr)
			}
			if gotWarn := len(warns) > 0; gotWarn != tc.wantWarned {
				t.Errorf("warns = %v, wantWarned = %v", warns, tc.wantWarned)
			}
		})
	}
}

// strPtr is the shared optional-scalar literal helper for tests.
func strPtr(s string) *string { return &s }

// TestParityFieldValidation is the docs/design/02-data-model.md §10 table for
// the M1.5 fields: the enum-ish scalars follow the labels strict/warn
// convention against their own config list (free-form when the list is empty),
// while rank/sprint/due violations are hard rejects.
func TestParityFieldValidation(t *testing.T) {
	base := item.Item{ID: "X", Number: "KIRA-1", Type: item.TypeTicket, Title: "t", State: "TODO"}
	cases := []struct {
		name     string
		tweak    func(*config.Config)
		mutate   func(*item.Item)
		wantErr  bool
		wantWarn bool
	}{
		{"subtype-known", nil, func(it *item.Item) { it.Subtype = strPtr("bug") }, false, false},
		{"subtype-unknown-lenient", nil, func(it *item.Item) { it.Subtype = strPtr("saga") }, false, true},
		{"subtype-unknown-strict", func(c *config.Config) { c.Labels.Strict = true },
			func(it *item.Item) { it.Subtype = strPtr("saga") }, true, false},
		{"subtype-freeform-when-empty", func(c *config.Config) { c.Subtypes = nil },
			func(it *item.Item) { it.Subtype = strPtr("saga") }, false, false},
		{"priority-unknown-lenient", nil, func(it *item.Item) { it.Priority = strPtr("P9") }, false, true},
		{"priority-unknown-strict", func(c *config.Config) { c.Labels.Strict = true },
			func(it *item.Item) { it.Priority = strPtr("P9") }, true, false},
		{"resolution-known", nil, func(it *item.Item) { it.Resolution = strPtr("dropped") }, false, false},
		{"resolution-unknown-lenient", nil, func(it *item.Item) { it.Resolution = strPtr("meh") }, false, true},
		{"rank-empty", nil, func(it *item.Item) { it.Rank = strPtr("") }, true, false},
		{"rank-freeform", nil, func(it *item.Item) { it.Rank = strPtr("0|zzz:") }, false, false},
		{"sprint-known", func(c *config.Config) {
			c.Sprints = []config.Sprint{{Key: "2026-S14", Name: "Sprint 14", Start: "2026-07-13", End: "2026-07-26"}}
		}, func(it *item.Item) { it.Sprint = strPtr("2026-S14") }, false, false},
		{"sprint-unknown", nil, func(it *item.Item) { it.Sprint = strPtr("2099-S1") }, true, false},
		{"due-valid", nil, func(it *item.Item) { it.Due = strPtr("2026-07-20") }, false, false},
		{"due-invalid", nil, func(it *item.Item) { it.Due = strPtr("someday") }, true, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := config.Default()
			if tc.tweak != nil {
				tc.tweak(cfg)
			}
			it := base
			tc.mutate(&it)
			errs, warns := validateItem(cfg, &it, false)
			if gotErr := len(errs) > 0; gotErr != tc.wantErr {
				t.Errorf("errs = %v, wantErr = %v", errs, tc.wantErr)
			}
			if gotWarn := len(warns) > 0; gotWarn != tc.wantWarn {
				t.Errorf("warns = %v, wantWarn = %v", warns, tc.wantWarn)
			}
		})
	}
}

// TestLinkTyped covers the links.<type> edges end to end: add is single-sided
// and idempotent, remove restores the canonical no-links form (key omitted from
// the file), and self-links are rejected (docs/design/02-data-model.md §3).
func TestLinkTyped(t *testing.T) {
	s, cfg := newStore(t)
	a := mustCreate(t, s, cfg, "A")
	b := mustCreate(t, s, cfg, "B")

	if _, err := s.Link(cfg, a.Number, LinkOpts{Target: LinkTyped, Type: item.LinkRelates, Ref: b.Number}); err != nil {
		t.Fatalf("link relates: %v", err)
	}
	if _, err := s.Link(cfg, a.Number, LinkOpts{Target: LinkTyped, Type: item.LinkDuplicateOf, Ref: b.Number}); err != nil {
		t.Fatalf("link duplicate-of: %v", err)
	}
	show, _ := s.Show(cfg, a.Number)
	if !slices.Equal(show.Links[item.LinkRelates], []string{b.ID}) ||
		!slices.Equal(show.Links[item.LinkDuplicateOf], []string{b.ID}) {
		t.Fatalf("links = %v, want %s in both types", show.Links, b.ID)
	}

	// Idempotent add is a no-op mutation.
	res, err := s.Link(cfg, a.Number, LinkOpts{Target: LinkTyped, Type: item.LinkRelates, Ref: b.Number})
	if err != nil || len(res.Changed) != 0 {
		t.Fatalf("re-link: changed = %v, err = %v", res.Changed, err)
	}

	// Removing both restores the canonical form: no links key in the file.
	for _, typ := range item.LinkTypes {
		if _, err := s.Link(cfg, a.Number, LinkOpts{Target: LinkTyped, Type: typ, Ref: b.Number, Remove: true}); err != nil {
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

	if _, err := s.Link(cfg, a.Number, LinkOpts{Target: LinkTyped, Type: item.LinkRelates, Ref: a.Number}); err == nil {
		t.Fatal("self relates-link: expected rejection")
	}
	if _, err := s.Link(cfg, a.Number, LinkOpts{Target: LinkTyped, Type: item.LinkRelates, Ref: "KIRA-999"}); err == nil {
		t.Fatal("dangling relates-link: expected rejection")
	}
}

// TestEditPathSelfLinkRejected pins that self-reference rejection lives in
// normalizeRefs, so it guards the edit/from-file path too, not just kira link.
func TestEditPathSelfLinkRejected(t *testing.T) {
	s, cfg := newStore(t)
	a := mustCreate(t, s, cfg, "A")

	editWith := func(mutate func(*item.Item)) error {
		show, err := s.Show(cfg, a.Number)
		if err != nil {
			t.Fatal(err)
		}
		it, err := item.Parse(mustReadItem(t, s, show.ID))
		if err != nil {
			t.Fatal(err)
		}
		mutate(it)
		path := filepath.Join(t.TempDir(), "edited.md")
		if err := os.WriteFile(path, []byte(it.Serialize()), 0o644); err != nil {
			t.Fatal(err)
		}
		_, err = s.Edit(cfg, a.Number, EditOpts{FromFile: path})
		return err
	}

	if err := editWith(func(it *item.Item) { it.BlockedBy = []string{a.ID} }); err == nil {
		t.Fatal("edit-path self blocked_by: expected rejection")
	}
	if err := editWith(func(it *item.Item) { it.Epic = &a.ID }); err == nil {
		t.Fatal("edit-path self epic: expected rejection")
	}
}

// mustReadItem reads an item's file content by ULID.
func mustReadItem(t *testing.T, s *Store, ulid string) string {
	t.Helper()
	data, err := os.ReadFile(s.itemPath(ulid))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// TestMoveRecordsResolution checks move --resolution stores the field alongside
// the state write, outranking the target state's resolution: tag (WONT_DO would
// default to dropped, so an ignored flag would be indistinguishable there).
func TestMoveRecordsResolution(t *testing.T) {
	s, cfg := newStore(t)
	res := mustCreate(t, s, cfg, "resolved")
	if _, err := s.Move(cfg, res.Number, "WONT_DO", MoveOpts{Resolution: "duplicate"}); err != nil {
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

// positionTo places an item in a state directly via edit, bypassing move's
// workflow enforcement, so guard tests control their starting state.
func positionTo(t *testing.T, s *Store, cfg *config.Config, ref, state string) {
	t.Helper()
	if _, err := s.Edit(cfg, ref, EditOpts{Fields: []FieldEdit{{Key: "state", Value: state}}}); err != nil {
		t.Fatalf("position %s to %s: %v", ref, state, err)
	}
}

// withTicketTransitions replaces the ticket workflow's transitions for one
// from-state; each newStore config is freshly parsed, so mutating it in place
// leaks nowhere.
func withTicketTransitions(cfg *config.Config, from string, ts []config.Transition) {
	cfg.Workflows[item.TypeTicket].Transitions[from] = ts
}

// TestMoveRequireGuard proves the default config's REVIEW -> DONE guard: the
// move fails listing the missing field, leaves state untouched, and passes once
// --resolution supplies it — with the flag outranking the transition's set:
// (docs/design/02-data-model.md §6; docs/design/04-cli.md move).
func TestMoveRequireGuard(t *testing.T) {
	s, cfg := newStore(t)
	res := mustCreate(t, s, cfg, "guarded")
	positionTo(t, s, cfg, res.Number, "REVIEW")

	_, err := s.Move(cfg, res.Number, "DONE", MoveOpts{})
	if err == nil || !strings.Contains(err.Error(), "requires resolution") {
		t.Fatalf("REVIEW -> DONE without resolution: err = %v, want missing-field rejection", err)
	}
	if got := stateOf(t, s, cfg, res.Number); got != "REVIEW" {
		t.Fatalf("state after rejected move = %s, want REVIEW", got)
	}

	if _, err := s.Move(cfg, res.Number, "DONE", MoveOpts{Resolution: "duplicate"}); err != nil {
		t.Fatalf("REVIEW -> DONE --resolution: %v", err)
	}
	show, _ := s.Show(cfg, res.Number)
	if show.Resolution == nil || *show.Resolution != "duplicate" {
		t.Fatalf("resolution = %v, want duplicate (--resolution outranks set:)", show.Resolution)
	}
}

// TestMoveRequireForceBypass proves --force skips the require: guard with the
// move still applying the transition's set: assignments.
func TestMoveRequireForceBypass(t *testing.T) {
	s, cfg := newStore(t)
	res := mustCreate(t, s, cfg, "forced")
	positionTo(t, s, cfg, res.Number, "REVIEW")

	if _, err := s.Move(cfg, res.Number, "DONE", MoveOpts{Force: true}); err != nil {
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

// TestMoveRequireMultiMissing proves a rejected move lists every missing field,
// and shrinks as fields are supplied.
func TestMoveRequireMultiMissing(t *testing.T) {
	s, cfg := newStore(t)
	withTicketTransitions(cfg, "TODO", []config.Transition{
		{To: "IN_PROGRESS", Require: []string{"owner", "due"}},
	})
	res := mustCreate(t, s, cfg, "multi")

	_, err := s.Move(cfg, res.Number, "IN_PROGRESS", MoveOpts{})
	if err == nil || !strings.Contains(err.Error(), "requires owner, due") {
		t.Fatalf("err = %v, want both missing fields listed", err)
	}

	if _, err := s.Edit(cfg, res.Number, EditOpts{Fields: []FieldEdit{{Key: "owner", Value: "shivam"}}}); err != nil {
		t.Fatalf("set owner: %v", err)
	}
	_, err = s.Move(cfg, res.Number, "IN_PROGRESS", MoveOpts{})
	if err == nil || !strings.Contains(err.Error(), "requires due") || strings.Contains(err.Error(), "owner") {
		t.Fatalf("err = %v, want only due listed", err)
	}

	if _, err := s.Edit(cfg, res.Number, EditOpts{Fields: []FieldEdit{{Key: "due", Value: "2026-07-20"}}}); err != nil {
		t.Fatalf("set due: %v", err)
	}
	if _, err := s.Move(cfg, res.Number, "IN_PROGRESS", MoveOpts{}); err != nil {
		t.Fatalf("move with all required fields set: %v", err)
	}
}

// TestMoveSetApplied proves set: assignments of every value kind land on the
// item as part of the transition write.
func TestMoveSetApplied(t *testing.T) {
	s, cfg := newStore(t)
	withTicketTransitions(cfg, "TODO", []config.Transition{
		{To: "IN_PROGRESS", Set: map[string]string{"owner": "alice", "priority": "P1", "estimate": "5"}},
	})
	res := mustCreate(t, s, cfg, "stamped")

	if _, err := s.Move(cfg, res.Number, "IN_PROGRESS", MoveOpts{}); err != nil {
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

// TestMoveSetVocabViolation proves a set: value still runs the normal runtime
// validation path: a strict-vocabulary violation blocks the move even though
// the assignment came from config, standing in for a config edited after
// static validation.
func TestMoveSetVocabViolation(t *testing.T) {
	s, cfg := newStore(t)
	cfg.Labels.Strict = true // governs the priority enum check (02 §10)
	withTicketTransitions(cfg, "TODO", []config.Transition{
		{To: "IN_PROGRESS", Set: map[string]string{"priority": "P9"}},
	})
	res := mustCreate(t, s, cfg, "badset")

	_, err := s.Move(cfg, res.Number, "IN_PROGRESS", MoveOpts{})
	if err == nil || !strings.Contains(err.Error(), "priority") {
		t.Fatalf("err = %v, want priority vocabulary rejection", err)
	}
	if got := stateOf(t, s, cfg, res.Number); got != "TODO" {
		t.Fatalf("state after rejected set: = %s, want TODO", got)
	}
}

// TestMoveResolutionLifecycle walks resolution across done boundaries: tagged
// done entry defaults the field, leaving done clears it, a done -> done move
// retakes the new state's tag, and --resolution is rejected on a non-done
// target (docs/design/02-data-model.md §6).
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

	// Entering a tagged done state with no explicit source takes the tag.
	if _, err := s.Move(cfg, res.Number, "WONT_DO", MoveOpts{}); err != nil {
		t.Fatalf("TODO -> WONT_DO: %v", err)
	}
	wantResolution("tagged done entry", "dropped")

	// Leaving done-category clears the field (reopen; off-graph, forced).
	if _, err := s.Move(cfg, res.Number, "TODO", MoveOpts{Force: true}); err != nil {
		t.Fatalf("WONT_DO -> TODO: %v", err)
	}
	wantResolution("reopen", "")

	// done -> done: entering a tagged state with nothing explicit this move
	// retakes that state's tag over the carried value.
	positionTo(t, s, cfg, res.Number, "REVIEW")
	if _, err := s.Move(cfg, res.Number, "DONE", MoveOpts{Resolution: "done"}); err != nil {
		t.Fatalf("REVIEW -> DONE: %v", err)
	}
	if _, err := s.Move(cfg, res.Number, "WONT_DO", MoveOpts{Force: true}); err != nil {
		t.Fatalf("DONE -> WONT_DO: %v", err)
	}
	wantResolution("done -> done retag", "dropped")

	// --resolution is meaningless outside done-category targets.
	_, err := s.Move(cfg, res.Number, "TODO", MoveOpts{Force: true, Resolution: "done"})
	if err == nil || !strings.Contains(err.Error(), "done-category") {
		t.Fatalf("--resolution to non-done target: err = %v, want rejection", err)
	}
}

// TestMoveWipWarning proves the advisory limit: silent at the limit, a warning
// in MoveResult.Warnings over it, and the move never blocked
// (docs/design/02-data-model.md §6; docs/design/04-cli.md move).
func TestMoveWipWarning(t *testing.T) {
	s, cfg := newStore(t) // default config: IN_PROGRESS wip: 3
	var nums []string
	for _, title := range []string{"w1", "w2", "w3", "w4"} {
		nums = append(nums, mustCreate(t, s, cfg, title).Number)
	}
	for i, num := range nums[:3] {
		mres, err := s.Move(cfg, num, "IN_PROGRESS", MoveOpts{})
		if err != nil {
			t.Fatalf("move %s: %v", num, err)
		}
		if len(mres.Warnings) != 0 {
			t.Fatalf("move %d of 3 (at limit): warnings = %v, want none", i+1, mres.Warnings)
		}
	}
	mres, err := s.Move(cfg, nums[3], "IN_PROGRESS", MoveOpts{})
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

// TestMoveWipCountsPerType proves the WIP census is per workflow type: an epic
// sitting in a same-named state does not count against the ticket limit.
func TestMoveWipCountsPerType(t *testing.T) {
	s, cfg := newStore(t)
	states := cfg.Workflows[item.TypeTicket].States
	for i := range states {
		if states[i].Key == "DONE" {
			states[i].Wip = 1
		}
	}

	epic, err := s.Create(cfg, CreateOpts{Type: item.TypeEpic, Title: "epic", NoEdit: true})
	if err != nil {
		t.Fatalf("create epic: %v", err)
	}
	for _, state := range []string{"ACTIVE", "DONE"} {
		if _, err := s.Move(cfg, epic.Number, state, MoveOpts{}); err != nil {
			t.Fatalf("move epic to %s: %v", state, err)
		}
	}

	first := mustCreate(t, s, cfg, "t1")
	positionTo(t, s, cfg, first.Number, "REVIEW")
	mres, err := s.Move(cfg, first.Number, "DONE", MoveOpts{Resolution: "done"})
	if err != nil {
		t.Fatalf("move t1: %v", err)
	}
	if len(mres.Warnings) != 0 {
		t.Fatalf("first ticket at limit: warnings = %v (epic in DONE must not count)", mres.Warnings)
	}

	second := mustCreate(t, s, cfg, "t2")
	positionTo(t, s, cfg, second.Number, "REVIEW")
	mres, err = s.Move(cfg, second.Number, "DONE", MoveOpts{Resolution: "done"})
	if err != nil {
		t.Fatalf("move t2: %v", err)
	}
	want := "DONE is over its WIP limit (2 > 1)"
	if len(mres.Warnings) != 1 || mres.Warnings[0] != want {
		t.Fatalf("warnings = %v, want [%s]", mres.Warnings, want)
	}
}

// TestFieldPresentCoversMutableFields locks fieldPresent to the schema surface
// guards may name: every item.MutableFields entry reads present on a populated
// item and absent on a zero one, so a new field cannot silently fall out of
// require: reach.
func TestFieldPresentCoversMutableFields(t *testing.T) {
	estimate := 1.0
	full := &item.Item{
		Title: "t", Subtype: ptrOrNil("bug"), Resolution: ptrOrNil("done"),
		Priority: ptrOrNil("P1"), Rank: ptrOrNil("0|m:"), Owner: ptrOrNil("shivam"),
		Reporter: ptrOrNil("alice"), Labels: []string{"x"}, Epic: ptrOrNil("01X"),
		Sprint: ptrOrNil("2026-S14"), Due: ptrOrNil("2026-07-20"), Estimate: &estimate,
	}
	empty := &item.Item{}
	for _, f := range item.MutableFields {
		if !fieldPresent(full, f) {
			t.Errorf("fieldPresent(populated, %q) = false", f)
		}
		if fieldPresent(empty, f) {
			t.Errorf("fieldPresent(zero, %q) = true", f)
		}
	}
}

// TestAssignStrictBypass proves the strict-rejection path and --force bypass
// hold end-to-end through the assign command (WP-1.3).
func TestAssignStrictBypass(t *testing.T) {
	s, cfg := newStore(t)
	cfg.People.Known = []string{"shivam", "alice"}
	cfg.People.Strict = true
	res := mustCreate(t, s, cfg, "assign")

	if _, err := s.Assign(cfg, res.Number, "mallory", AssignOpts{}); err == nil {
		t.Fatal("strict assign of unknown user: expected rejection")
	}
	if _, err := s.Assign(cfg, res.Number, "mallory", AssignOpts{Force: true}); err != nil {
		t.Fatalf("forced assign: %v", err)
	}
	if show, _ := s.Show(cfg, res.Number); show.Owner == nil || *show.Owner != "mallory" {
		t.Fatalf("owner = %v, want mallory", show.Owner)
	}
	// --reporter targets the reporter field. Use a fresh item: the forced unknown
	// owner above would otherwise re-fail whole-item validation on any non-forced
	// write (force is per-write, not sticky).
	fresh := mustCreate(t, s, cfg, "reporter")
	if _, err := s.Assign(cfg, fresh.Number, "alice", AssignOpts{Reporter: true}); err != nil {
		t.Fatalf("assign reporter: %v", err)
	}
	if show, _ := s.Show(cfg, fresh.Number); show.Reporter == nil || *show.Reporter != "alice" {
		t.Fatalf("reporter = %v, want alice", show.Reporter)
	}
}
