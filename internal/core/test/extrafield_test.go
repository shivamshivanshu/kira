package core_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/entityschema"
)

func withDeclaredField(t *testing.T, s *core.Store, typ string, field entityschema.FieldDef, enums ...entityschema.EnumDef) {
	t.Helper()
	path := filepath.Join(s.Root(), ".kira", "schema", typ+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	var schema entityschema.Schema
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatalf("unmarshal %s: %v", path, err)
	}
	schema.Fields = append(schema.Fields, field)
	schema.Enums = append(schema.Enums, enums...)
	out, err := json.Marshal(schema)
	if err != nil {
		t.Fatalf("marshal %s: %v", path, err)
	}
	if err := os.WriteFile(path, out, 0o644); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
}

func TestCreateFieldSetsDeclaredExtra(t *testing.T) {
	s, cfg := newStore(t)
	withDeclaredField(t, s, datamodel.TypeTicket,
		entityschema.FieldDef{Name: "severity", Type: entityschema.FieldEnum, Enum: "severity"},
		entityschema.EnumDef{Name: "severity", Values: []string{"low", "high"}},
	)

	res, err := s.Create(cfg, core.CreateOpts{
		Type: datamodel.TypeTicket, Title: "t", NoEdit: true,
		Fields: []core.FieldEdit{{Key: "severity", Value: "high"}},
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if got := mustReadItem(t, s, res.ID); !strings.Contains(got, "severity: high") {
		t.Fatalf("created item must persist the declared extra:\n%s", got)
	}
}

func TestEditFieldSetsAndClearsDeclaredExtra(t *testing.T) {
	s, cfg := newStore(t)
	withDeclaredField(t, s, datamodel.TypeTicket,
		entityschema.FieldDef{Name: "severity", Type: entityschema.FieldEnum, Enum: "severity"},
		entityschema.EnumDef{Name: "severity", Values: []string{"low", "high"}},
	)
	res := mustCreate(t, s, cfg, "t")

	if _, err := s.Edit(cfg, res.Number, core.EditOpts{Fields: []core.FieldEdit{{Key: "severity", Value: "high"}}}); err != nil {
		t.Fatalf("edit: %v", err)
	}
	if got := mustReadItem(t, s, res.ID); !strings.Contains(got, "severity: high") {
		t.Fatalf("edit must persist the declared extra:\n%s", got)
	}

	cfg.Labels.Strict = true
	if _, err := s.Edit(cfg, res.Number, core.EditOpts{Fields: []core.FieldEdit{{Key: "severity", Value: "critical"}}}); err == nil {
		t.Fatal("an out-of-vocab enum extra must be rejected under a strict vocab")
	}

	if _, err := s.Edit(cfg, res.Number, core.EditOpts{Fields: []core.FieldEdit{{Key: "severity", Value: ""}}}); err != nil {
		t.Fatalf("clearing the extra: %v", err)
	}
	if got := mustReadItem(t, s, res.ID); strings.Contains(got, "severity") {
		t.Fatalf("--field severity= must remove the extra from the file:\n%s", got)
	}
}

func TestEditFieldNormalizesDeclaredRefExtraToULID(t *testing.T) {
	s, cfg := newStore(t)
	withDeclaredField(t, s, datamodel.TypeTicket,
		entityschema.FieldDef{Name: "reviewer_of", Type: entityschema.FieldRef, Target: datamodel.TypeTicket},
	)
	blocker := mustCreate(t, s, cfg, "blocker")
	res := mustCreate(t, s, cfg, "subject")

	if _, err := s.Edit(cfg, res.Number, core.EditOpts{Fields: []core.FieldEdit{{Key: "reviewer_of", Value: blocker.Number}}}); err != nil {
		t.Fatalf("edit: %v", err)
	}
	got := mustReadItem(t, s, res.ID)
	if strings.Contains(got, "reviewer_of: "+blocker.Number) {
		t.Fatalf("declared ref extra must be normalized to a ULID, not left as the display number %s:\n%s", blocker.Number, got)
	}
	if !strings.Contains(got, "reviewer_of: "+blocker.ID) {
		t.Fatalf("declared ref extra must resolve to blocker's ULID %s:\n%s", blocker.ID, got)
	}
}
