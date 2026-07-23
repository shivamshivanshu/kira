package entityschema_test

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shivamshivanshu/kira/internal/config"
	"github.com/shivamshivanshu/kira/internal/entityschema"
	"github.com/shivamshivanshu/kira/internal/storage"
)

// This suite is the Phase 1 acceptance criteria: the embedded built-in
// schemas must accept, with zero violations, every item this repo already
// dogfoods on itself under .kira/tickets/, and its configured boards.

func repoRoot(t *testing.T) string {
	t.Helper()
	out, err := exec.Command("go", "env", "GOMOD").Output()
	if err != nil {
		t.Fatalf("go env GOMOD: %v", err)
	}
	return filepath.Dir(strings.TrimSpace(string(out)))
}

func TestConformanceAgainstRepoTickets(t *testing.T) {
	root := repoRoot(t)
	fs := storage.New(root)

	items, warnings, err := fs.LoadAll()
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("unexpected parse warnings: %v", warnings)
	}
	if len(items) == 0 {
		t.Fatal("expected a non-empty corpus of real tickets")
	}

	cfg, err := config.Load(root)
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	schemas, err := entityschema.Load(fs.SchemaDir())
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	enums := entityschema.ConfigVocab(cfg)

	violations := 0
	for _, it := range items {
		schema, ok := schemas[it.Type]
		if !ok {
			t.Errorf("%s: no schema for type %q", it.ID, it.Type)
			continue
		}
		for _, v := range entityschema.Validate(schema, entityschema.ProjectItem(schema, it), enums, nil) {
			t.Errorf("%s (%s): %v", it.Number, it.ID, v)
			violations++
		}
	}
	if violations != 0 {
		t.Fatalf("%d violation(s) across %d items", violations, len(items))
	}
	t.Logf("validated %d items with zero violations", len(items))
}

func TestConformanceAgainstConfigBoards(t *testing.T) {
	root := repoRoot(t)
	fs := storage.New(root)

	cfg, err := config.Load(root)
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	schemas, err := entityschema.Load(fs.SchemaDir())
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	schema := schemas["board"]

	boards := cfg.EffectiveBoards()
	if len(boards) == 0 {
		t.Fatal("expected at least one configured board")
	}
	for _, b := range boards {
		values := map[string]any{
			"key":         b.Key,
			"name":        b.Name,
			"description": b.Description,
			"default":     b.Default,
			"archived":    b.Archived,
		}
		for _, v := range entityschema.Validate(schema, values, nil, nil) {
			t.Errorf("board %s: %v", b.Key, v)
		}
	}
}
