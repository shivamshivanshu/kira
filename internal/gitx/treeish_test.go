package gitx

import "testing"

func TestNumstatNoIndexSurfacesRealErrors(t *testing.T) {
	repo := Repo{Dir: "/nonexistent-kira-numstat-dir"}
	_, _, err := repo.NumstatNoIndex("a", "b")
	if err == nil {
		t.Fatalf("want error when git cannot run, got nil")
	}
}
