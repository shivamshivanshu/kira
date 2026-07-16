package core

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/shivamshivanshu/kira/internal/codec"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/gitx"
	"github.com/shivamshivanshu/kira/internal/testutil"
)

func TestGitTextMergeCleanMerge(t *testing.T) {
	base := "alpha\nbeta\ngamma\n"
	ours := "ALPHA\nbeta\ngamma\n"
	theirs := "alpha\nbeta\nGAMMA\n"

	merged, conflict := gitTextMerge(base, ours, theirs)
	if conflict {
		t.Fatalf("disjoint edits reported conflict:\n%s", merged)
	}
	if !strings.Contains(merged, "ALPHA") || !strings.Contains(merged, "GAMMA") {
		t.Fatalf("merge dropped a side:\n%s", merged)
	}
	if strings.Contains(merged, "<<<<<<<") {
		t.Fatalf("clean merge left conflict markers:\n%s", merged)
	}
}

func TestGitTextMergeConflict(t *testing.T) {
	merged, conflict := gitTextMerge("origin\n", "ours\n", "theirs\n")
	if !conflict {
		t.Fatalf("divergent edits to the same line must conflict:\n%s", merged)
	}
	if !strings.Contains(merged, "<<<<<<<") || !strings.Contains(merged, ">>>>>>>") {
		t.Fatalf("conflict output missing markers:\n%s", merged)
	}
}

func mergeFixture(t *testing.T) (gitx.Repo, string) {
	t.Helper()
	dir := t.TempDir()
	if err := testutil.GitInit(dir); err != nil {
		t.Fatalf("git init: %v", err)
	}
	if _, err := Init(dir, "KIRA", false); err != nil {
		t.Fatalf("init store: %v", err)
	}
	return gitx.Repo{Dir: dir}, dir
}

func writeItemFile(t *testing.T, path string, it *datamodel.Item) {
	t.Helper()
	if err := os.WriteFile(path, []byte(codec.Serialize(it)), itemFileMode); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func readItemFile(t *testing.T, path string) *datamodel.Item {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	it, err := codec.Parse(string(data))
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	return it
}

func TestMergeFileParseableCleanMerge(t *testing.T) {
	repo, dir := mergeFixture(t)
	base := eventTicket()
	ours := *base
	ours.Title = "Mine"
	theirs := *base
	theirs.State = "IN_PROGRESS"

	basePath := filepath.Join(dir, "base.md")
	oursPath := filepath.Join(dir, "ours.md")
	theirsPath := filepath.Join(dir, "theirs.md")
	writeItemFile(t, basePath, base)
	writeItemFile(t, oursPath, &ours)
	writeItemFile(t, theirsPath, &theirs)

	res, err := MergeFile(repo, basePath, oursPath, theirsPath)
	if err != nil {
		t.Fatalf("MergeFile: %v", err)
	}
	if len(res.Arbitrated) != 0 {
		t.Fatalf("disjoint field edits arbitrated: %v", res.Arbitrated)
	}
	merged := readItemFile(t, oursPath)
	if merged.Title != "Mine" || merged.State != "IN_PROGRESS" {
		t.Fatalf("merged item = %q/%q, want Mine/IN_PROGRESS", merged.Title, merged.State)
	}
}

func TestMergeFileHonoursGitPrefixForNestedRoot(t *testing.T) {
	toplevel := t.TempDir()
	if err := testutil.GitInit(toplevel); err != nil {
		t.Fatalf("git init: %v", err)
	}
	nested := filepath.Join(toplevel, "sub")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Init(nested, "KIRA", false); err != nil {
		t.Fatalf("init store: %v", err)
	}
	t.Setenv("GIT_PREFIX", "sub/")

	repo := gitx.Repo{Dir: toplevel}
	base := eventTicket()
	ours := *base
	ours.Title = "Mine"
	theirs := *base
	theirs.State = "IN_PROGRESS"

	basePath := filepath.Join(toplevel, "base.md")
	oursPath := filepath.Join(toplevel, "ours.md")
	theirsPath := filepath.Join(toplevel, "theirs.md")
	writeItemFile(t, basePath, base)
	writeItemFile(t, oursPath, &ours)
	writeItemFile(t, theirsPath, &theirs)

	res, err := MergeFile(repo, basePath, oursPath, theirsPath)
	if err != nil {
		t.Fatalf("MergeFile: %v (want it to find the nested store via GIT_PREFIX, matching a real merge driver invocation)", err)
	}
	if len(res.Arbitrated) != 0 {
		t.Fatalf("disjoint field edits arbitrated: %v", res.Arbitrated)
	}
}

func TestMergeFileParseablePerFieldConflict(t *testing.T) {
	repo, dir := mergeFixture(t)
	base := eventTicket()
	ours := *base
	ours.Title = "Mine"
	theirs := *base
	theirs.Title = "Theirs"

	basePath := filepath.Join(dir, "base.md")
	oursPath := filepath.Join(dir, "ours.md")
	theirsPath := filepath.Join(dir, "theirs.md")
	writeItemFile(t, basePath, base)
	writeItemFile(t, oursPath, &ours)
	writeItemFile(t, theirsPath, &theirs)

	res, err := MergeFile(repo, basePath, oursPath, theirsPath)
	if err != nil {
		t.Fatalf("MergeFile: %v", err)
	}
	if !slices.Contains(res.Arbitrated, datamodel.KeyTitle) {
		t.Fatalf("arbitrated = %v, want title reported as conflicting", res.Arbitrated)
	}
	if readItemFile(t, oursPath).Title == base.Title {
		t.Fatalf("conflicting field was not arbitrated to a side")
	}
}
