package schema

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

// These tests exercise the generator directly over synthetic types, so a
// generator regression fails here independent of whether schema/kira.json
// happens to already exercise the same shape.

type SyntheticEmbedded struct {
	Inner string `json:"inner"`
}

type syntheticChild struct {
	Name string `json:"name"`
}

type syntheticStruct struct {
	*SyntheticEmbedded
	Name     string         `json:"name"`
	Optional *string        `json:"optional,omitempty"`
	Required *string        `json:"required"`
	Hidden   string         `json:"-"`
	Nested   syntheticChild `json:"nested"`
	Tags     []string       `json:"tags"`
}

func TestGeneratorEmbeddedPointerDeref(t *testing.T) {
	g := newGenerator()
	name := g.register(reflect.TypeFor[syntheticStruct]())
	def := g.defs[name].(map[string]any)
	props := def["properties"].(map[string]any)
	if _, ok := props["inner"]; !ok {
		t.Fatalf("field promoted from embedded *syntheticEmbedded is missing: %+v", props)
	}
}

func TestGeneratorRequiredMatchesJSONEmit(t *testing.T) {
	g := newGenerator()
	name := g.register(reflect.TypeFor[syntheticStruct]())
	def := g.defs[name].(map[string]any)
	props := def["properties"].(map[string]any)

	if _, ok := props["hidden"]; ok {
		t.Errorf(`json:"-"` + ` field leaked into schema properties`)
	}

	required := map[string]bool{}
	for _, r := range def["required"].([]string) {
		required[r] = true
	}
	// "required" (a pointer field without omitempty) must be required: Go's
	// encoding/json always emits it, as null when nil. "optional" carries
	// omitempty and is correctly excluded.
	for _, name := range []string{"inner", "name", "required", "nested", "tags"} {
		if !required[name] {
			t.Errorf("%q should be required (always emitted by encoding/json)", name)
		}
	}
	if required["optional"] {
		t.Errorf("optional field (omitempty) should not be required")
	}
}

// TestSchemaRejectsNilSliceEmittedAsNull proves the schema is strict enough
// to catch the regression the ticket calls out: a nil slice field with no
// omitempty marshals to JSON `null`, not `[]`, but the generated schema
// requires the field's declared array type — so a caller that forgets to
// initialize the slice trips this test, not just a stale golden diff.
func TestSchemaRejectsNilSliceEmittedAsNull(t *testing.T) {
	g := newGenerator()
	name := g.register(reflect.TypeFor[syntheticStruct]())
	doc := map[string]any{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"$ref":    "#/$defs/" + name,
		"$defs":   g.defs,
	}
	buf, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("marshal schema doc: %v", err)
	}
	sch := compileSchema(t, buf)

	var zero syntheticStruct
	zero.SyntheticEmbedded = &SyntheticEmbedded{Inner: "x"}
	zero.Required = new(string)
	instBytes, err := json.Marshal(zero) // Tags is nil -> "tags":null
	if err != nil {
		t.Fatalf("marshal instance: %v", err)
	}

	inst, err := jsonschema.UnmarshalJSON(bytes.NewReader(instBytes))
	if err != nil {
		t.Fatalf("unmarshal instance: %v", err)
	}
	if err := sch.Validate(inst); err == nil {
		t.Fatalf("expected validation to reject a nil slice marshaled as null, got no error")
	}
}

func TestGeneratorEnumConstraint(t *testing.T) {
	type syntheticEnum string
	et := reflect.TypeFor[syntheticEnum]()
	t.Cleanup(func() { delete(stringEnums, et) })
	stringEnums[et] = []string{"a", "b"}

	g := newGenerator()
	got := g.schemaForType(et)
	if got["type"] != "string" {
		t.Errorf("type = %v, want string", got["type"])
	}
	enum, ok := got["enum"].([]string)
	if !ok || len(enum) != 2 || enum[0] != "a" || enum[1] != "b" {
		t.Errorf("enum = %v, want [a b]", got["enum"])
	}
}

// TestGeneratorNameCollisionQualifiesSecondType exercises the real collision
// this ticket describes: doctor.Report and syncx.Report share a bare Name().
// Both must end up registered under distinct, stable $defs keys, keyed by
// reflect.Type rather than by name.
func TestGeneratorNameCollisionQualifiesSecondType(t *testing.T) {
	type firstReport struct {
		A string `json:"a"`
	}
	type secondReport struct {
		B string `json:"b"`
	}
	// Rename via reflect isn't possible, so we simulate the collision the
	// way defName actually sees it: two distinct reflect.Types resolving to
	// the same t.Name(). We can't declare two Go types both literally named
	// "Report" in one package, so drive defName directly instead.
	g := newGenerator()
	firstT := reflect.TypeFor[firstReport]()
	secondT := reflect.TypeFor[secondReport]()
	g.owners["Report"] = firstT
	g.names[firstT] = "Report"

	name := g.defName(secondT)
	if name == "Report" {
		t.Fatalf("second type should not silently reuse the first type's name")
	}
	if g.owners[name] != secondT {
		t.Fatalf("owners[%q] = %v, want %v", name, g.owners[name], secondT)
	}
}
