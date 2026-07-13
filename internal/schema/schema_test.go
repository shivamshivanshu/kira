package schema

import (
	"flag"
	"os"
	"path/filepath"
	"testing"
)

var update = flag.Bool("update", false, "rewrite the committed schema artifact")

const artifactPath = "../../docs/schema/kira.json"

func TestArtifactFresh(t *testing.T) {
	got, err := Generate()
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if *update {
		if err := os.MkdirAll(filepath.Dir(artifactPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(artifactPath, got, 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(artifactPath)
	if err != nil {
		t.Fatalf("read artifact: %v (run: go test ./internal/schema -update)", err)
	}
	if string(got) != string(want) {
		t.Errorf("schema artifact stale; run: go test ./internal/schema -update")
	}
}
