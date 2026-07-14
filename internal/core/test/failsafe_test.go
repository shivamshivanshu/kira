package core_test

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/gitx"
	"github.com/shivamshivanshu/kira/internal/storage"
)

const unknownKeyLine = "future_field: fromNewerKira\n"

func withUnknownKey(content string) string {
	return strings.Replace(content, "---\n", "---\n"+unknownKeyLine, 1)
}

func withUnknownLink(content string) string {
	return strings.Replace(content, "blocked_by: []\n",
		"blocked_by: []\nlinks:\n  future_rel: [01J8XB3K9P0Q2R4S6T8V0W1X2Y]\n", 1)
}

func overwriteItem(t *testing.T, s *core.Store, ulid, content string) {
	t.Helper()
	if err := os.WriteFile(storage.New(s.Root()).ItemPath(ulid), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func assertUpgradeRefusal(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected refusal, got nil error")
	}
	var e *errx.Error
	if !errors.As(err, &e) {
		t.Fatalf("want *errx.Error, got %T: %v", err, err)
	}
	if e.Code != errx.ExitEnv {
		t.Fatalf("exit code = %d, want %d (ExitEnv)", e.Code, errx.ExitEnv)
	}
	if !strings.Contains(e.Error(), "newer kira") {
		t.Fatalf("error missing forward-compat message: %q", e.Error())
	}
	if !strings.Contains(e.Hint, "upgrade") {
		t.Fatalf("error missing upgrade hint: %q", e.Hint)
	}
}

func TestReadsSucceedWithUnknowns(t *testing.T) {
	t.Parallel()
	s, cfg := newStore(t)
	res := mustCreate(t, s, cfg, "Has future fields")
	overwriteItem(t, s, res.ID, withUnknownLink(withUnknownKey(mustReadItem(t, s, res.ID))))

	if _, err := s.Show(cfg, "KIRA-1", ""); err != nil {
		t.Fatalf("show must succeed on unknown-carrying item: %v", err)
	}
	list, err := s.List(cfg, core.ListOpts{})
	if err != nil {
		t.Fatalf("list must succeed on unknown-carrying item: %v", err)
	}
	if len(list.Items) != 1 || list.Items[0].Number != "KIRA-1" {
		t.Fatalf("list did not return the unknown-carrying item: %+v", list.Items)
	}
}

func TestWriteVerbsRefuseOnUnknown(t *testing.T) {
	t.Parallel()
	s, cfg := newStore(t)
	res := mustCreate(t, s, cfg, "Has future fields")
	overwriteItem(t, s, res.ID, withUnknownKey(mustReadItem(t, s, res.ID)))

	verbs := map[string]func() error{
		"edit-field": func() error {
			_, err := s.Edit(cfg, "KIRA-1", core.EditOpts{Fields: []core.FieldEdit{{Key: "state", Value: "IN_PROGRESS"}}})
			return err
		},
		"assign": func() error {
			_, err := s.Assign(cfg, "KIRA-1", "bob", core.AssignOpts{})
			return err
		},
		"move": func() error {
			_, err := s.Move(cfg, "KIRA-1", "IN_PROGRESS", core.MoveOpts{})
			return err
		},
		"link": func() error {
			_, err := s.Link(cfg, "KIRA-1", core.LinkOpts{Target: core.LinkBlockedBy, Ref: "KIRA-1"})
			return err
		},
		"comment": func() error {
			_, err := s.Comment(cfg, "KIRA-1", core.CommentOpts{Message: "hi", HasMessage: true})
			return err
		},
	}
	for name, run := range verbs {
		t.Run(name, func(t *testing.T) {
			assertUpgradeRefusal(t, run())
		})
	}

	if got := mustReadItem(t, s, res.ID); !strings.Contains(got, unknownKeyLine) {
		t.Fatalf("a refused write mutated the file:\n%s", got)
	}
}

func TestWriteVerbsRefuseOnCRLF(t *testing.T) {
	t.Parallel()
	s, cfg := newStore(t)
	res := mustCreate(t, s, cfg, "LF native")
	crlf := strings.ReplaceAll(mustReadItem(t, s, res.ID), "\n", "\r\n")
	overwriteItem(t, s, res.ID, crlf)

	if _, err := s.Show(cfg, "KIRA-1", ""); err != nil {
		t.Fatalf("show must tolerate CRLF for reads: %v", err)
	}

	_, err := s.Edit(cfg, "KIRA-1", core.EditOpts{Fields: []core.FieldEdit{{Key: "state", Value: "IN_PROGRESS"}}})
	if err == nil {
		t.Fatal("expected refusal, got nil error")
	}
	var e *errx.Error
	if !errors.As(err, &e) {
		t.Fatalf("want *errx.Error, got %T: %v", err, err)
	}
	if e.Code != errx.ExitEnv {
		t.Fatalf("exit code = %d, want %d (ExitEnv)", e.Code, errx.ExitEnv)
	}
	if !strings.Contains(e.Error(), "CRLF") || !strings.Contains(e.Hint, "renormalize") {
		t.Fatalf("error/hint missing CRLF renormalize guidance: %q / %q", e.Error(), e.Hint)
	}
	if got := mustReadItem(t, s, res.ID); !strings.Contains(got, "\r\n") {
		t.Fatalf("a refused write renormalized the file")
	}
}

func TestEditorModeRefusesBeforeReserializingOriginal(t *testing.T) {
	t.Parallel()
	s, cfg := newStore(t)
	res := mustCreate(t, s, cfg, "Has future fields")
	overwriteItem(t, s, res.ID, withUnknownKey(mustReadItem(t, s, res.ID)))

	_, err := s.Edit(cfg, "KIRA-1", core.EditOpts{})
	assertUpgradeRefusal(t, err)

	if got := mustReadItem(t, s, res.ID); !strings.Contains(got, unknownKeyLine) {
		t.Fatalf("editor-mode edit reserialized and dropped the unknown key:\n%s", got)
	}
}

func TestEditFullEditorRefusesBeforeDroppingUnknowns(t *testing.T) {
	t.Parallel()
	s, cfg := newStore(t)
	res := mustCreate(t, s, cfg, "Has future fields")
	overwriteItem(t, s, res.ID, withUnknownKey(mustReadItem(t, s, res.ID)))

	clean := strings.Replace(mustReadItem(t, s, res.ID), unknownKeyLine, "", 1)
	src := writeTempItem(t, strings.Replace(clean, "state: TODO", "state: IN_PROGRESS", 1))
	_, err := s.Edit(cfg, "KIRA-1", core.EditOpts{FromFile: src})
	assertUpgradeRefusal(t, err)
}

func TestReconcileWritesOldNumberAsAlias(t *testing.T) {
	s, cfg := newStore(t)
	one := mustCreate(t, s, cfg, "one")
	two := mustCreate(t, s, cfg, "two")
	overwriteItem(t, s, two.ID, strings.Replace(mustReadItem(t, s, two.ID), "number: "+two.Number, "number: "+one.Number, 1))

	res, err := s.Reconcile(cfg)
	if err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	if len(res.Renumbered) != 1 {
		t.Fatalf("want exactly one renumber, got %+v", res.Renumbered)
	}
	r := res.Renumbered[0]
	got := mustReadItem(t, s, r.ID)
	if !strings.Contains(got, "number: "+r.To) {
		t.Fatalf("renumbered file number is not %s:\n%s", r.To, got)
	}
	if !strings.Contains(got, r.From) {
		t.Fatalf("old number %s not kept as an alias:\n%s", r.From, got)
	}
}

func TestReconcileRefusesOnUnknown(t *testing.T) {
	t.Parallel()
	s, cfg := newStore(t)
	one := mustCreate(t, s, cfg, "one")
	two := mustCreate(t, s, cfg, "two")

	collide := strings.Replace(mustReadItem(t, s, two.ID), "number: KIRA-2", "number: KIRA-1", 1)
	overwriteItem(t, s, two.ID, withUnknownKey(collide))
	overwriteItem(t, s, one.ID, withUnknownKey(mustReadItem(t, s, one.ID)))

	_, err := s.Reconcile(cfg)
	assertUpgradeRefusal(t, err)
}

func TestMergeFileRefusesOnUnknown(t *testing.T) {
	t.Parallel()
	s, cfg := newStore(t)
	res := mustCreate(t, s, cfg, "mergeable")
	clean := mustReadItem(t, s, res.ID)

	base := writeTempItem(t, clean)
	ours := writeTempItem(t, withUnknownKey(clean))
	theirs := writeTempItem(t, clean)

	_, err := core.MergeFile(gitx.Repo{Dir: s.Root()}, base, ours, theirs)
	assertUpgradeRefusal(t, err)

	if got, _ := os.ReadFile(ours); !strings.Contains(string(got), "future_field") {
		t.Fatalf("refused merge overwrote ours, dropping unknown:\n%s", got)
	}
}

func TestResolveRefusesOnUnknown(t *testing.T) {
	t.Parallel()
	s, cfg := newStore(t)
	res := mustCreate(t, s, cfg, "conflicted")
	repo := gitx.Repo{Dir: s.Root()}
	git := func(args ...string) {
		t.Helper()
		if _, err := repo.Output(args...); err != nil {
			t.Fatalf("git %v: %v", args, err)
		}
	}
	base := mustReadItem(t, s, res.ID)
	variant := func(priority string) string {
		return withUnknownKey(strings.Replace(base, "blocked_by: []\n", "blocked_by: []\npriority: "+priority+"\n", 1))
	}

	git("checkout", "-b", "other")
	overwriteItem(t, s, res.ID, variant("P2"))
	git("add", "-A")
	git("commit", "-m", "other variant")
	git("checkout", "-")
	overwriteItem(t, s, res.ID, variant("P1"))
	git("add", "-A")
	git("commit", "-m", "main variant")

	if _, err := repo.Output("merge", "other"); err == nil {
		t.Fatal("expected a merge conflict on the ticket file")
	}

	_, err := s.Resolve(nil, false)
	assertUpgradeRefusal(t, err)
}
