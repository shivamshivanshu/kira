package treeish_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shivamshivanshu/kira/internal/codec"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/gitx"
	"github.com/shivamshivanshu/kira/internal/storage"
	"github.com/shivamshivanshu/kira/internal/testutil"
	"github.com/shivamshivanshu/kira/internal/treeish"
)

func writeCommit(t *testing.T, repo gitx.Repo, root, relPath, content, msg string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(relPath)), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Output("add", "--", relPath); err != nil {
		t.Fatalf("add: %v", err)
	}
	if _, err := repo.Output("commit", "-m", msg); err != nil {
		t.Fatalf("commit: %v", err)
	}
}

func initRepo(t *testing.T) (string, gitx.Repo) {
	t.Helper()
	root := testutil.InitGitRepo(t)
	repo := gitx.Repo{Dir: root}
	if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(storage.TicketsPrefix)), 0o755); err != nil {
		t.Fatal(err)
	}
	writeCommit(t, repo, root, storage.ConfigRelPath, "project:\n  key: KIRA\n  name: kira\n", "init")
	return root, repo
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
	writeCommit(t, repo, root, ".kira/tickets/"+it.ID+".md", codec.Serialize(it), msg)
}

func TestLoadSkipsMalformedItemWithWarning(t *testing.T) {
	t.Parallel()
	root, repo := initRepo(t)
	commitItem(t, repo, root, item("01J8X8Q7RZTN5Y3VXW2A9K4E7F", "KIRA-1", "TODO", nil), "good")

	bad := item("01J8X7B1Q2W3E4R5T6Y7U8I9O0", "KIRA-2", "TODO", nil)
	content := strings.Replace(codec.Serialize(bad), "state: TODO\n", "state: TODO\nstate: DONE\n", 1)
	writeCommit(t, repo, root, ".kira/tickets/"+bad.ID+".md", content, "bad")

	loaded, err := treeish.Load(repo, "HEAD")
	if err != nil {
		t.Fatalf("Load must not abort on one malformed item: %v", err)
	}
	if len(loaded.Items) != 1 || loaded.Items[0].Number != "KIRA-1" {
		t.Fatalf("items = %+v, want only KIRA-1", loaded.Items)
	}
	if len(loaded.Warnings) != 1 || !strings.Contains(loaded.Warnings[0], bad.ID) || !strings.Contains(loaded.Warnings[0], "duplicate key") {
		t.Fatalf("warnings = %q, want a skip note naming the file and the duplicate key", loaded.Warnings)
	}
}

func TestLoadSkipsMissingItemBlobWithWarning(t *testing.T) {
	t.Parallel()
	root, repo := initRepo(t)
	commitItem(t, repo, root, item("01J8X8Q7RZTN5Y3VXW2A9K4E7F", "KIRA-1", "TODO", nil), "good")
	commitItem(t, repo, root, item("01J8X7B1Q2W3E4R5T6Y7U8I9O0", "KIRA-2", "TODO", nil), "ghost")

	ghostSHA, err := repo.Output("rev-parse", "HEAD:.kira/tickets/01J8X7B1Q2W3E4R5T6Y7U8I9O0.md")
	if err != nil {
		t.Fatalf("rev-parse ghost blob: %v", err)
	}
	objPath := filepath.Join(root, ".git", "objects", ghostSHA[:2], ghostSHA[2:])
	if err := os.Remove(objPath); err != nil {
		t.Fatalf("remove loose object %s: %v", objPath, err)
	}

	loaded, err := treeish.Load(repo, "HEAD")
	if err != nil {
		t.Fatalf("Load must not abort on a missing blob: %v", err)
	}
	if len(loaded.Items) != 1 || loaded.Items[0].Number != "KIRA-1" {
		t.Fatalf("items = %+v, want only KIRA-1", loaded.Items)
	}
	if len(loaded.Warnings) != 1 || !strings.Contains(loaded.Warnings[0], "01J8X7B1Q2W3E4R5T6Y7U8I9O0") || !strings.Contains(loaded.Warnings[0], "corrupt tree") {
		t.Fatalf("warnings = %q, want a skip note naming the ghost item and corrupt tree", loaded.Warnings)
	}
}

func TestLoadBeforeInit(t *testing.T) {
	t.Parallel()
	root := testutil.InitGitRepo(t)
	repo := gitx.Repo{Dir: root}
	writeCommit(t, repo, root, "README.md", "hello\n", "before kira init")

	_, err := treeish.Load(repo, "HEAD")
	if err == nil {
		t.Fatal("Load must fail on a commit with no .kira/config.yaml")
	}
	if !strings.Contains(err.Error(), storage.ConfigRelPath) {
		t.Errorf("err = %q, want it to name %s", err, storage.ConfigRelPath)
	}
	if strings.Contains(err.Error(), "corrupt") {
		t.Errorf("err = %q, must not call a genuinely absent config corrupt", err)
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
