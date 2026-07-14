package treeish_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/shivamshivanshu/kira/internal/codec"
	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/gitx"
	"github.com/shivamshivanshu/kira/internal/testutil"
	"github.com/shivamshivanshu/kira/internal/treeish"
)

func initRepo(t *testing.T) (string, gitx.Repo) {
	t.Helper()
	root := testutil.InitGitRepo(t)
	if _, err := core.Init(root, "KIRA", false); err != nil {
		t.Fatalf("Init: %v", err)
	}
	return root, gitx.Repo{Dir: root}
}

func item(ulid, number, state string, aliases []string) *datamodel.Item {
	return &datamodel.Item{
		ID: ulid, Number: number, Aliases: aliases, Type: datamodel.TypeTicket,
		Title: "T " + number, State: state, Labels: []string{}, BlockedBy: []string{},
		Created: "2026-01-01T00:00:00Z", Updated: "2026-01-01T00:00:00Z",
		Body: "## Description\n\nbody\n",
	}
}

func commitItem(t *testing.T, repo gitx.Repo, root string, it *datamodel.Item, msg string) {
	t.Helper()
	rel := ".kira/tickets/" + it.ID + ".md"
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(rel)), []byte(codec.Serialize(it)), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Output("add", "--", rel); err != nil {
		t.Fatalf("add: %v", err)
	}
	if _, err := repo.Output("commit", "-m", msg); err != nil {
		t.Fatalf("commit: %v", err)
	}
}

func TestLoadAtTreeish(t *testing.T) {
	t.Parallel()
	root, repo := initRepo(t)
	commitItem(t, repo, root, item("01J8X8Q7RZTN5Y3VXW2A9K4E7F", "KIRA-1", "TODO", nil), "one")
	first, err := repo.ResolveTreeish("HEAD")
	if err != nil {
		t.Fatalf("resolve HEAD: %v", err)
	}
	commitItem(t, repo, root, item("01J8X7B1Q2W3E4R5T6Y7U8I9O0", "KIRA-2", "TODO", nil), "two")

	head, err := treeish.Load(repo, "HEAD")
	if err != nil {
		t.Fatalf("Load HEAD: %v", err)
	}
	if len(head.Items) != 2 {
		t.Fatalf("HEAD items = %d, want 2", len(head.Items))
	}
	if head.Config.Project.Key != "KIRA" {
		t.Fatalf("config key = %q, want KIRA", head.Config.Project.Key)
	}

	old, err := treeish.Load(repo, first)
	if err != nil {
		t.Fatalf("Load first: %v", err)
	}
	if len(old.Items) != 1 {
		t.Fatalf("first-commit items = %d, want 1 (time-travel to before KIRA-2)", len(old.Items))
	}
}

func TestLoadResolvesAlias(t *testing.T) {
	t.Parallel()
	root, repo := initRepo(t)
	commitItem(t, repo, root, item("01J8X8Q7RZTN5Y3VXW2A9K4E7F", "KIRA-5", "TODO", []string{"KIRA-1"}), "renumbered")

	loaded, err := treeish.Load(repo, "HEAD")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	for _, ref := range []string{"KIRA-5", "KIRA-1"} {
		ulid, err := loaded.Resolver.Resolve(ref)
		if err != nil {
			t.Fatalf("resolve %s: %v", ref, err)
		}
		if ulid != "01J8X8Q7RZTN5Y3VXW2A9K4E7F" {
			t.Fatalf("resolve %s = %s, want the item ULID", ref, ulid)
		}
	}
}
