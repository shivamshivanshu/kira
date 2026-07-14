package core

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shivamshivanshu/kira/internal/config"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/testutil"
	"github.com/shivamshivanshu/kira/internal/workon"
)

func TestTemplatePathSubtype(t *testing.T) {
	dir := testutil.InitGitRepo(t)
	s := newStore(dir)
	tdir := s.fs().TemplateDir()
	if err := os.MkdirAll(tdir, 0o755); err != nil {
		t.Fatal(err)
	}
	base := filepath.Join(tdir, "ticket.md")
	sub := filepath.Join(tdir, "ticket.bug.md")
	if err := os.WriteFile(base, []byte("title: base\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if got := s.templatePath("ticket", ""); got != base {
		t.Errorf("no subtype: templatePath = %q, want %q", got, base)
	}
	if got := s.templatePath("ticket", "bug"); got != base {
		t.Errorf("subtype without a matching file must fall back: templatePath = %q, want %q", got, base)
	}

	if err := os.WriteFile(sub, []byte("title: bug\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := s.templatePath("ticket", "bug"); got != sub {
		t.Errorf("subtype with a matching file: templatePath = %q, want %q", got, sub)
	}
	if got := s.templatePath("ticket", "story"); got != base {
		t.Errorf("unknown subtype falls back to base: templatePath = %q, want %q", got, base)
	}
}

func TestTemplateDraftRejectsUnsafeSubtype(t *testing.T) {
	dir := testutil.InitGitRepo(t)
	s := newStore(dir)
	for _, bad := range []string{"x/../../secret", "a b", "../x", "x/y", "x.y", "."} {
		if _, err := s.templateDraft("ticket", bad); err == nil {
			t.Errorf("templateDraft with subtype %q must be rejected", bad)
		}
	}
	if _, err := s.templateDraft("ticket", "bug"); err != nil {
		t.Errorf("valid subtype rejected: %v", err)
	}
}

func TestListWithMatchesActivity(t *testing.T) {
	dir := testutil.InitGitRepo(t)
	s := newStore(dir)
	if err := os.MkdirAll(filepath.Join(dir, ".kira", "tickets"), 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := config.Default()
	it := &datamodel.Item{
		ID: "01HZZ0ACT0000000000000000", Number: "KIRA-1", Type: datamodel.TypeTicket,
		Title: "Act", State: "TODO", Labels: []string{},
		Created: "2026-05-01T00:00:00Z", Updated: "2026-05-01T00:00:00Z",
	}
	if _, err := s.fs().WriteItem(it); err != nil {
		t.Fatal(err)
	}
	_, matched, err := s.ListWithMatches(cfg, "activity < 2026-05-02")
	if err != nil {
		t.Fatalf("ListWithMatches: %v", err)
	}
	if !matched[it.ID] {
		t.Error("activity filter must match through ListWithMatches; Activity has to be populated on the load path")
	}
	_, matched, err = s.ListWithMatches(cfg, "activity > 2026-05-02")
	if err != nil {
		t.Fatal(err)
	}
	if matched[it.ID] {
		t.Error("activity > a future day must not match")
	}
}

func TestCreateHereBlockingHalfFailure(t *testing.T) {
	dir := testutil.InitGitRepo(t)
	s := newStore(dir)
	if err := os.MkdirAll(filepath.Join(dir, ".kira", "tickets"), 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := config.Default()
	active, err := s.Create(cfg, CreateOpts{Type: datamodel.TypeTicket, Title: "Active", NoEdit: true})
	if err != nil {
		t.Fatalf("create active: %v", err)
	}
	if err := s.writeActive(workon.ActivePointer{Ticket: active.ID}); err != nil {
		t.Fatal(err)
	}

	p := s.itemPath(active.ID)
	raw, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, bytes.ReplaceAll(raw, []byte("\n"), []byte("\r\n")), 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := s.Create(cfg, CreateOpts{Type: datamodel.TypeTicket, Title: "Prereq", NoEdit: true, Here: true, Blocking: true})
	if err == nil {
		t.Fatal("expected the blocking link to fail")
	}
	if res == nil {
		t.Fatal("a half-failure must still return the created ticket")
	}
	if !strings.Contains(err.Error(), res.Number) {
		t.Errorf("error must name the created ticket %s: %v", res.Number, err)
	}
	if !s.RefExists(cfg, res.Number) {
		t.Errorf("created ticket %s must exist despite the link failure", res.Number)
	}
}
