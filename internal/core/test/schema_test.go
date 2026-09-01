package core_test

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestMaterializeSchemasFillsGapsWithoutClobbering(t *testing.T) {
	t.Parallel()
	s, cfg := newStore(t)
	dir := filepath.Join(s.Root(), ".kira", "schema")

	// a repo predating schema files, with one hand-edited schema kept
	edited := filepath.Join(dir, "ticket.json")
	custom := []byte(`{"name":"ticket","fields":[]}`)
	if err := os.WriteFile(edited, custom, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(dir, "board.json")); err != nil {
		t.Fatal(err)
	}

	written, err := s.MaterializeSchemas(cfg)
	if err != nil {
		t.Fatalf("MaterializeSchemas: %v", err)
	}
	if !slices.ContainsFunc(written, func(p string) bool { return filepath.Base(p) == "board.json" }) {
		t.Fatalf("board.json not reported as written: %v", written)
	}
	for _, p := range written {
		if filepath.Base(p) == "ticket.json" {
			t.Fatalf("an existing schema must not be rewritten: %v", written)
		}
	}
	got, err := os.ReadFile(edited)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(custom) {
		t.Fatalf("hand-edited schema was clobbered: %s", got)
	}
}
