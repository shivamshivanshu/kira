package core_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/storage"
)

const severityDecl = `{"name": "severity", "type": "string", "placement": "frontmatter"}`

func TestCreateFromFileCarriesDeclaredExtras(t *testing.T) {
	t.Parallel()
	s, cfg := newStore(t)
	s = declareTicketExtras(t, s, severityDecl+`, {"name": "watchlist", "type": "string", "list": true, "placement": "frontmatter"}`)

	src := writeTempItem(t, "---\ntitle: Sev ticket\ntype: ticket\nseverity: high\nwatchlist: [alice, bob]\n---\n\nbody\n")
	res, err := s.Create(cfg, core.CreateOpts{Type: datamodel.TypeTicket, FromFile: src})
	if err != nil {
		t.Fatalf("Create --from-file: %v", err)
	}
	got := mustReadItem(t, s, res.ID)
	if !strings.Contains(got, "severity: high\n") {
		t.Fatalf("created item dropped the declared scalar extra:\n%s", got)
	}
	if !strings.Contains(got, "watchlist: [alice, bob]\n") {
		t.Fatalf("created item dropped the declared list extra:\n%s", got)
	}
}

func TestCreateFromFileRejectsUndeclaredKey(t *testing.T) {
	t.Parallel()
	s, cfg := newStore(t)
	s = declareTicketExtras(t, s, severityDecl)

	src := writeTempItem(t, "---\ntitle: Bad ticket\ntype: ticket\nwibble: x\n---\n\nbody\n")
	_, err := s.Create(cfg, core.CreateOpts{Type: datamodel.TypeTicket, FromFile: src})
	if err == nil {
		t.Fatal("an undeclared frontmatter key must be rejected, not dropped")
	}
	if !strings.Contains(err.Error(), "wibble") {
		t.Fatalf("rejection must name the offending key: %v", err)
	}
}

func TestCreateEditorRejectsUndeclaredThenCarriesDeclaredExtra(t *testing.T) {
	s, cfg := newStore(t)
	s = declareTicketExtras(t, s, severityDecl)

	dir := t.TempDir()
	undeclared := "---\ntitle: Edited\ntype: ticket\nwibble: x\n---\n\nbody\n"
	declared := "---\ntitle: Edited\ntype: ticket\nseverity: high\n---\n\nbody\n"
	if err := os.WriteFile(filepath.Join(dir, "first.md"), []byte(undeclared), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "second.md"), []byte(declared), 0o644); err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(dir, "editor.sh")
	src := "#!/bin/sh\n" +
		"if grep -q 'kira:error' \"$1\"; then cp \"$KIRA_SECOND\" \"$1\"; else cp \"$KIRA_FIRST\" \"$1\"; fi\n"
	if err := os.WriteFile(script, []byte(src), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("KIRA_FIRST", filepath.Join(dir, "first.md"))
	t.Setenv("KIRA_SECOND", filepath.Join(dir, "second.md"))
	t.Setenv("VISUAL", "")
	t.Setenv("EDITOR", "sh "+script)

	res, err := s.Create(cfg, core.CreateOpts{Type: datamodel.TypeTicket})
	if err != nil {
		t.Fatalf("Create via editor: %v", err)
	}
	got := mustReadItem(t, s, res.ID)
	if strings.Contains(got, "wibble") {
		t.Fatalf("undeclared key leaked into the created item:\n%s", got)
	}
	if !strings.Contains(got, "severity: high\n") {
		t.Fatalf("editor-typed declared extra was dropped (the undeclared first buffer was never rejected):\n%s", got)
	}
}

func TestCreateNoEditCarriesTemplateExtras(t *testing.T) {
	t.Parallel()
	s, cfg := newStore(t)
	s = declareTicketExtras(t, s, severityDecl)
	tdir := storage.New(s.Root()).TemplateDir()
	if err := os.MkdirAll(tdir, 0o755); err != nil {
		t.Fatal(err)
	}
	tmpl := "---\ntitle: ''\ntype: ticket\nseverity: low\n---\n\nbody\n"
	if err := os.WriteFile(filepath.Join(tdir, "ticket.md"), []byte(tmpl), 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := s.Create(cfg, core.CreateOpts{Type: datamodel.TypeTicket, Title: "From template", NoEdit: true})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if got := mustReadItem(t, s, res.ID); !strings.Contains(got, "severity: low\n") {
		t.Fatalf("no-edit create dropped the template's declared extra:\n%s", got)
	}
}
