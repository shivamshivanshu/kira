package core

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/testutil"
)

func TestApplyFieldEditSystemFieldRejected(t *testing.T) {
	it := &datamodel.Item{Created: "2026-07-20T10:00:00Z"}
	if err := applyFieldEdit(it, "created", "2020-01-01T00:00:00Z"); err == nil {
		t.Fatal("a system field must not be settable via --field")
	}
}

func TestApplyFieldEditBodyPlacedFieldSetsAndClearsItsSection(t *testing.T) {
	it := &datamodel.Item{Body: "## Description\n\nsome prose\n\n## Acceptance criteria\n\nold criteria\n"}

	if err := applyFieldEdit(it, "acceptance", "new notes"); err != nil {
		t.Fatalf("set body field: %v", err)
	}
	if got := datamodel.BodySections(it.Body)["Acceptance criteria"]; got != "new notes" {
		t.Errorf("Acceptance criteria section = %q, want %q", got, "new notes")
	}
	if got := datamodel.BodySections(it.Body)["Description"]; got != "some prose" {
		t.Errorf("unrelated Description section changed: got %q", got)
	}

	if err := applyFieldEdit(it, "acceptance", ""); err != nil {
		t.Fatalf("clear body field: %v", err)
	}
	if _, ok := datamodel.BodySections(it.Body)["Acceptance criteria"]; ok {
		t.Error("--field key= must remove the body section")
	}
}

func TestApplyFieldEditBodyValueWithHeadingRejected(t *testing.T) {
	it := &datamodel.Item{Body: "## Acceptance criteria\n\nold\n"}
	if err := applyFieldEdit(it, "acceptance", "intro\n## Overview\nmore"); err == nil {
		t.Fatal("a body value containing a \"## \" heading line must be rejected")
	}
	if got := datamodel.BodySections(it.Body)["Acceptance criteria"]; got != "old" {
		t.Errorf("a rejected body edit must not mutate the section, got %q", got)
	}
}

func TestEditFieldSetsBodySectionAndRoundTrips(t *testing.T) {
	dir := testutil.InitGitRepo(t)
	s := newStore(dir)
	if err := os.MkdirAll(filepath.Join(dir, ".kira", "tickets"), 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := defaultConfigWithKIRABoard()
	created, err := s.Create(cfg, CreateOpts{Type: datamodel.TypeTicket, Title: "T", NoEdit: true})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	if _, err := s.Edit(cfg, created.ID, EditOpts{Fields: []FieldEdit{{Key: "description", Value: "edited description"}}}); err != nil {
		t.Fatalf("edit: %v", err)
	}

	it, _, _, err := s.resolveRef(cfg, created.ID)
	if err != nil {
		t.Fatalf("resolveRef: %v", err)
	}
	if got := datamodel.BodySections(it.Body)["Description"]; got != "edited description" {
		t.Errorf("Description section = %q, want %q", got, "edited description")
	}

	if _, err := s.Edit(cfg, created.ID, EditOpts{Fields: []FieldEdit{{Key: "description", Value: ""}}}); err != nil {
		t.Fatalf("clear: %v", err)
	}
	it, _, _, err = s.resolveRef(cfg, created.ID)
	if err != nil {
		t.Fatalf("resolveRef: %v", err)
	}
	if _, ok := datamodel.BodySections(it.Body)["Description"]; ok {
		t.Error("--field description= must remove the Description section")
	}
}

func TestApplyFieldEditUnknownKeySuggestsDeclaredBodyField(t *testing.T) {
	it := &datamodel.Item{}
	err := applyFieldEdit(it, "acceptanse", "x")
	if err == nil {
		t.Fatal("unknown key must be rejected")
	}
	var e *errx.Error
	if !errors.As(err, &e) {
		t.Fatalf("want *errx.Error, got %T", err)
	}
	if !strings.Contains(e.Hint, "acceptance") {
		t.Errorf("hint = %q, want a did-you-mean for the body field acceptance", e.Hint)
	}
}
