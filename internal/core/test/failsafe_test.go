package core_test

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/storage"
)

func overwriteItem(t *testing.T, s *core.Store, ulid, content string) {
	t.Helper()
	if err := os.WriteFile(storage.New(s.Root()).ItemPath(ulid), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestEditDedupsDuplicateLabels(t *testing.T) {
	t.Parallel()
	s, cfg := newStore(t)
	res := mustCreate(t, s, cfg, "Dup labels")

	_, err := s.Edit(cfg, "KIRA-1", core.EditOpts{Fields: []core.FieldEdit{{Key: "labels", Value: "bug,bug"}}})
	if err != nil {
		t.Fatalf("edit with a duplicate label must succeed, got %v", err)
	}
	if got := mustReadItem(t, s, res.ID); !strings.Contains(got, "labels: [bug]") {
		t.Fatalf("duplicate label was not deduped:\n%s", got)
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
