package core_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/storage"
)

const futureFieldDecl = `{"name": "future_field", "type": "string", "placement": "frontmatter"}`

func declareTicketExtras(t *testing.T, s *core.Store, fieldsJSON string) *core.Store {
	t.Helper()
	return declareTicketSchema(t, s, `{"name": "ticket", "fields": [`+fieldsJSON+`]}`)
}

func declareTicketSchema(t *testing.T, s *core.Store, schemaJSON string) *core.Store {
	t.Helper()
	path := filepath.Join(storage.New(s.Root()).SchemaDir(), "ticket.json")
	if err := os.WriteFile(path, []byte(schemaJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	fresh, err := core.Discover(s.Root())
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	return fresh
}

func TestDeclaredExtraDoesNotBlockWrites(t *testing.T) {
	t.Parallel()
	s, cfg := newStore(t)
	res := mustCreate(t, s, cfg, "Declared extra")
	overwriteItem(t, s, res.ID, withUnknownKey(mustReadItem(t, s, res.ID)))
	s = declareTicketExtras(t, s, futureFieldDecl)

	if _, err := s.Assign(cfg, "KIRA-1", "bob", core.AssignOpts{}); err != nil {
		t.Fatalf("a schema-declared extra must not block mutation: %v", err)
	}
	if got := mustReadItem(t, s, res.ID); !strings.Contains(got, unknownKeyLine) {
		t.Fatalf("mutation dropped the declared extra:\n%s", got)
	}
}

func TestUndeclaredExtraStillBlocksWrites(t *testing.T) {
	t.Parallel()
	s, cfg := newStore(t)
	res := mustCreate(t, s, cfg, "Undeclared extra")
	content := strings.Replace(mustReadItem(t, s, res.ID), "---\n", "---\nother_field: x\n", 1)
	overwriteItem(t, s, res.ID, content)
	s = declareTicketExtras(t, s, futureFieldDecl)

	_, err := s.Assign(cfg, "KIRA-1", "bob", core.AssignOpts{})
	assertUpgradeRefusal(t, err)
	if !strings.Contains(err.Error(), "other_field") {
		t.Fatalf("refusal must name the undeclared key: %v", err)
	}
}

func TestExtrasBlockWhenTypeHasNoSchema(t *testing.T) {
	t.Parallel()
	s, cfg := newStore(t)
	res := mustCreate(t, s, cfg, "No schema for type")
	content := strings.Replace(withUnknownKey(mustReadItem(t, s, res.ID)), "type: ticket\n", "type: gadget\n", 1)
	overwriteItem(t, s, res.ID, content)
	s = declareTicketExtras(t, s, futureFieldDecl)

	_, err := s.Assign(cfg, "KIRA-1", "bob", core.AssignOpts{})
	assertUpgradeRefusal(t, err)
}

func TestDeclaredUncapturedExtraStillBlocks(t *testing.T) {
	t.Parallel()
	s, cfg := newStore(t)
	res := mustCreate(t, s, cfg, "Declared mapping")
	content := strings.Replace(mustReadItem(t, s, res.ID), "---\n", "---\nfuture_field:\n  nested: 1\n", 1)
	overwriteItem(t, s, res.ID, content)
	s = declareTicketExtras(t, s, futureFieldDecl)

	_, err := s.Assign(cfg, "KIRA-1", "bob", core.AssignOpts{})
	assertUpgradeRefusal(t, err)
}

func TestEditFieldSetsDeclaredExtras(t *testing.T) {
	t.Parallel()
	s, cfg := newStore(t)
	res := mustCreate(t, s, cfg, "Typed extras")
	s = declareTicketExtras(t, s,
		`{"name": "points", "type": "int", "placement": "frontmatter"}, `+
			`{"name": "note", "type": "string", "placement": "frontmatter"}`)

	fields := []core.FieldEdit{{Key: "points", Value: "42"}, {Key: "note", Value: "a: b"}}
	if _, err := s.Edit(cfg, "KIRA-1", core.EditOpts{Fields: fields}); err != nil {
		t.Fatalf("Edit: %v", err)
	}
	got := mustReadItem(t, s, res.ID)
	if !strings.Contains(got, "points: 42\n") {
		t.Fatalf("int extra must serialize unquoted:\n%s", got)
	}
	if !strings.Contains(got, "note: 'a: b'\n") {
		t.Fatalf("quote-needing extra must serialize quoted:\n%s", got)
	}

	if _, err := s.Edit(cfg, "KIRA-1", core.EditOpts{Fields: []core.FieldEdit{{Key: "points", Value: "43"}}}); err != nil {
		t.Fatalf("re-edit after reparse: %v", err)
	}
	got = mustReadItem(t, s, res.ID)
	if !strings.Contains(got, "points: 43\n") || !strings.Contains(got, "note: 'a: b'\n") {
		t.Fatalf("stored extras did not round-trip through parse and serialize:\n%s", got)
	}
}

func TestEditFieldRejectsUndeclaredKey(t *testing.T) {
	t.Parallel()
	s, cfg := newStore(t)
	mustCreate(t, s, cfg, "Reject unknown --field")
	s = declareTicketExtras(t, s, futureFieldDecl)

	_, err := s.Edit(cfg, "KIRA-1", core.EditOpts{Fields: []core.FieldEdit{{Key: "titl", Value: "x"}}})
	if err == nil {
		t.Fatal("expected an unknown-field error")
	}
	if !strings.Contains(err.Error(), `unknown or immutable field "titl"`) {
		t.Fatalf("unexpected error: %v", err)
	}
	var e *errx.Error
	if !errors.As(err, &e) || !strings.Contains(e.Hint, "did you mean `--field title=...`") {
		t.Fatalf("expected the did-you-mean hint, got %v", err)
	}
}

func TestEditFieldHonorsImmutableExtra(t *testing.T) {
	t.Parallel()
	s, cfg := newStore(t)
	mustCreate(t, s, cfg, "Immutable extra")
	s = declareTicketExtras(t, s, `{"name": "external_id", "type": "string", "immutable": true, "placement": "frontmatter"}`)

	_, err := s.Edit(cfg, "KIRA-1", core.EditOpts{Fields: []core.FieldEdit{{Key: "external_id", Value: "x"}}})
	if err == nil || !strings.Contains(err.Error(), `unknown or immutable field "external_id"`) {
		t.Fatalf("expected the immutable-field refusal, got %v", err)
	}
}

const typedTicketSchema = `{
  "name": "ticket",
  "enums": [{"name": "sev", "values": ["LOW", "HIGH"]}],
  "fields": [
    {"name": "severity", "type": "enum", "enum": "sev", "placement": "frontmatter"},
    {"name": "story_points", "type": "int", "placement": "frontmatter"}
  ]
}`

func TestEditFieldRejectsMistypedExtraValues(t *testing.T) {
	t.Parallel()
	s, cfg := newStore(t)
	res := mustCreate(t, s, cfg, "Typed rejection")
	s = declareTicketSchema(t, s, typedTicketSchema)
	before := mustReadItem(t, s, res.ID)

	for key, tt := range map[string]struct{ value, want string }{
		"story_points": {"abc", "expected type int"},
		"severity":     {"BOGUS", `not in the "sev" vocabulary`},
	} {
		_, err := s.Edit(cfg, "KIRA-1", core.EditOpts{Fields: []core.FieldEdit{{Key: key, Value: tt.value}}})
		if err == nil || !strings.Contains(err.Error(), tt.want) {
			t.Fatalf("%s=%s: expected a violation containing %q, got %v", key, tt.value, tt.want, err)
		}
	}
	if got := mustReadItem(t, s, res.ID); got != before {
		t.Fatalf("a refused edit mutated the file:\n%s", got)
	}

	fields := []core.FieldEdit{{Key: "story_points", Value: "5"}, {Key: "severity", Value: "LOW"}}
	if _, err := s.Edit(cfg, "KIRA-1", core.EditOpts{Fields: fields}); err != nil {
		t.Fatalf("conforming extras must pass: %v", err)
	}
	got := mustReadItem(t, s, res.ID)
	if !strings.Contains(got, "story_points: 5\n") || !strings.Contains(got, "severity: LOW\n") {
		t.Fatalf("conforming extras did not persist:\n%s", got)
	}
}

func TestMutationBlockedByInvalidOnDiskExtra(t *testing.T) {
	t.Parallel()
	s, cfg := newStore(t)
	res := mustCreate(t, s, cfg, "Invalid on disk")
	content := strings.Replace(mustReadItem(t, s, res.ID), "---\n", "---\nseverity: BOGUS\n", 1)
	overwriteItem(t, s, res.ID, content)
	s = declareTicketSchema(t, s, typedTicketSchema)

	_, err := s.Assign(cfg, "KIRA-1", "bob", core.AssignOpts{})
	if err == nil || !strings.Contains(err.Error(), `not in the "sev" vocabulary`) {
		t.Fatalf("expected the on-disk violation to block mutation, got %v", err)
	}
}
