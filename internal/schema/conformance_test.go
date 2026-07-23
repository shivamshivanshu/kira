package schema

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

const goldenDir = "../../tests/contract/testdata/golden"

// compileSchema compiles raw JSON Schema document bytes into a validator.
func compileSchema(t *testing.T, data []byte) *jsonschema.Schema {
	t.Helper()
	doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("unmarshal schema: %v", err)
	}
	c := jsonschema.NewCompiler()
	const resourceURL = "kira.json"
	if err := c.AddResource(resourceURL, doc); err != nil {
		t.Fatalf("add schema resource: %v", err)
	}
	sch, err := c.Compile(resourceURL)
	if err != nil {
		t.Fatalf("compile schema: %v", err)
	}
	return sch
}

// TestGoldenConformsToSchema validates the committed schema/kira.json against
// a real JSON Schema implementation, not just against Generate()'s own
// output — TestArtifactFresh only proves the artifact is up to date with the
// generator, not that either one is a well-formed, satisfiable schema.
func TestGoldenConformsToSchema(t *testing.T) {
	data, err := os.ReadFile(artifactPath)
	if err != nil {
		t.Fatalf("read artifact: %v", err)
	}
	sch := compileSchema(t, data)

	entries, err := os.ReadDir(goldenDir)
	if err != nil {
		t.Fatalf("read golden dir: %v", err)
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		t.Run(e.Name(), func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join(goldenDir, e.Name()))
			if err != nil {
				t.Fatalf("read golden: %v", err)
			}
			inst, err := jsonschema.UnmarshalJSON(bytes.NewReader(data))
			if err != nil {
				t.Fatalf("unmarshal golden: %v", err)
			}
			if err := sch.Validate(inst); err != nil {
				t.Errorf("golden does not conform to the root schema: %v", err)
			}
		})
	}
}
