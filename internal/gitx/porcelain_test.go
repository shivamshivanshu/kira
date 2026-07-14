package gitx

import (
	"slices"
	"testing"
)

func TestParsePorcelainPaths(t *testing.T) {
	out := " M .kira/tickets/01ABC.md\n" +
		"R  old.md -> new.md\n" +
		"?? \"spa ced.md\"\n" +
		"R  \"old name.md\" -> \"new name.md\"\n"
	got := parsePorcelainPaths(out)
	want := []string{
		".kira/tickets/01ABC.md",
		"new.md",
		"spa ced.md",
		"new name.md",
	}
	if !slices.Equal(got, want) {
		t.Fatalf("parsePorcelainPaths:\n got  %q\n want %q", got, want)
	}
}

func TestRelToDirUsesCachedPrefix(t *testing.T) {
	dir := t.TempDir()
	showPrefixCache.Store(dir, "sub/")
	t.Cleanup(func() { showPrefixCache.Delete(dir) })

	repo := Repo{Dir: dir}
	got, err := repo.relToDir([]string{"sub/.kira/tickets/01ABC.md", "sub/other.md"})
	if err != nil {
		t.Fatalf("relToDir: %v", err)
	}
	want := []string{".kira/tickets/01ABC.md", "other.md"}
	if !slices.Equal(got, want) {
		t.Fatalf("relToDir:\n got  %q\n want %q", got, want)
	}
}

func TestRelToDirToplevelPassthrough(t *testing.T) {
	dir := t.TempDir()
	showPrefixCache.Store(dir, "")
	t.Cleanup(func() { showPrefixCache.Delete(dir) })

	in := []string{".kira/tickets/01ABC.md"}
	got, err := (Repo{Dir: dir}).relToDir(in)
	if err != nil {
		t.Fatalf("relToDir: %v", err)
	}
	if !slices.Equal(got, in) {
		t.Fatalf("toplevel passthrough altered paths:\n got  %q\n want %q", got, in)
	}
}
