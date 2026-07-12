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

	position := func(state string) {
		if _, err := s.Edit(cfg, res.Number, EditOpts{Fields: []FieldEdit{{Key: "state", Value: state}}}); err != nil {
			t.Fatalf("position to %s: %v", state, err)
		}
	}

	for _, from := range states {
		for _, to := range states {
			position(from)
			allowed := from == to || slices.Contains(wf.Transitions[from], to)
			_, err := s.Move(cfg, res.Number, to, MoveOpts{})
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
			if _, ferr := s.Move(cfg, res.Number, to, MoveOpts{Force: true}); ferr != nil {
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
