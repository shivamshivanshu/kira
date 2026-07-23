package entityschema

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadBuiltinsWithNoUserDir(t *testing.T) {

	schemas, err := Load("")

	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	for _, name := range []string{"ticket", "epic", "board"} {
		if _, ok := schemas[name]; !ok {
			t.Errorf("missing built-in schema %q", name)
		}
	}
}

func TestLoadMissingSchemaDirYieldsBuiltinsOnly(t *testing.T) {

	schemas, err := Load(filepath.Join(t.TempDir(), "does-not-exist"))

	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(schemas) != 3 {
		t.Fatalf("expected the 3 embedded built-ins, got %d", len(schemas))
	}
}

func TestLoadUserFileOverridesBuiltinByName(t *testing.T) {
	dir := t.TempDir()
	custom := `{"name": "ticket", "fields": [{"name": "title", "type": "string", "required": true}]}`
	if err := os.WriteFile(filepath.Join(dir, "ticket.json"), []byte(custom), 0o644); err != nil {
		t.Fatal(err)
	}

	schemas, err := Load(dir)

	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(schemas["ticket"].Fields) != 1 {
		t.Fatalf("user schema should replace the built-in ticket schema, got fields %v", schemas["ticket"].Fields)
	}
}

func TestLoadUserFileAddsNewType(t *testing.T) {
	dir := t.TempDir()
	custom := `{"name": "bug", "fields": [{"name": "title", "type": "string", "required": true}]}`
	if err := os.WriteFile(filepath.Join(dir, "bug.json"), []byte(custom), 0o644); err != nil {
		t.Fatal(err)
	}

	schemas, err := Load(dir)

	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if _, ok := schemas["bug"]; !ok {
		t.Fatal("expected the user-defined bug schema alongside the built-ins")
	}
	if _, ok := schemas["ticket"]; !ok {
		t.Fatal("built-in ticket schema should still be present")
	}
}

func TestLoadRejectsStateFieldWithoutStates(t *testing.T) {
	dir := t.TempDir()
	custom := `{"name": "bad", "fields": [{"name": "state", "type": "state"}]}`
	if err := os.WriteFile(filepath.Join(dir, "bad.json"), []byte(custom), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(dir)

	if err == nil {
		t.Fatal("expected an error for a state field with no declared states")
	}
}

func TestLoadRejectsUnknownFieldType(t *testing.T) {
	dir := t.TempDir()
	custom := `{"name": "bad", "fields": [{"name": "title", "type": "nonsense"}]}`
	if err := os.WriteFile(filepath.Join(dir, "bad.json"), []byte(custom), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(dir)

	if err == nil {
		t.Fatal("expected an error for an unknown field type")
	}
}

func TestValidateSchemaRejectsMalformedDefinitions(t *testing.T) {
	cases := map[string]Schema{
		"missing schema name": {Fields: []FieldDef{{Name: "x", Type: FieldString}}},
		"duplicate field":     {Name: "w", Fields: []FieldDef{{Name: "x", Type: FieldString}, {Name: "x", Type: FieldInt}}},
		"enum without enum":   {Name: "w", Fields: []FieldDef{{Name: "p", Type: FieldEnum}}},
		"ref without target":  {Name: "w", Fields: []FieldDef{{Name: "o", Type: FieldRef}}},
		"non-string list":     {Name: "w", Fields: []FieldDef{{Name: "n", Type: FieldInt, List: true}}},
	}

	for name, schema := range cases {
		t.Run(name, func(t *testing.T) {
			if err := checkSchemaShape(schema); err == nil {
				t.Fatalf("expected %s to be rejected", name)
			}
		})
	}
}

func TestWriteDefaultsIsIdempotentAndPreservesEdits(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "schema")

	if err := WriteDefaults(dir); err != nil {
		t.Fatalf("WriteDefaults: %v", err)
	}
	edited := []byte(`{"name": "ticket", "fields": []}`)
	if err := os.WriteFile(filepath.Join(dir, "ticket.json"), edited, 0o644); err != nil {
		t.Fatal(err)
	}

	if err := WriteDefaults(dir); err != nil {
		t.Fatalf("second WriteDefaults: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(dir, "ticket.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(edited) {
		t.Fatalf("re-running WriteDefaults clobbered a user edit: %s", got)
	}
}

func TestBuiltinNames(t *testing.T) {
	names, err := BuiltinNames()

	if err != nil {
		t.Fatalf("BuiltinNames: %v", err)
	}
	want := []string{"board", "epic", "ticket"}
	if len(names) != len(want) {
		t.Fatalf("got %v, want %v", names, want)
	}
	for i, n := range want {
		if names[i] != n {
			t.Fatalf("got %v, want %v", names, want)
		}
	}
}
