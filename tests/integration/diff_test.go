package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/gitx"
)

func TestDiffDeletedAndBody(t *testing.T) {
	root := initGitRepo(t)
	initStore(t, root)
	repo := gitx.Repo{Dir: root}
	s, _ := core.Discover(root)
	cfg, _ := s.Config()

	keep, err := s.Create(cfg, core.CreateOpts{Type: datamodel.TypeTicket, Title: "Keep", NoEdit: true})
	if err != nil {
		t.Fatalf("create keep: %v", err)
	}
	gone, err := s.Create(cfg, core.CreateOpts{Type: datamodel.TypeTicket, Title: "Gone", NoEdit: true})
	if err != nil {
		t.Fatalf("create gone: %v", err)
	}

	mainBranch, _ := repo.Output("branch", "--show-current")
	if _, err := repo.Output("checkout", "-b", "later"); err != nil {
		t.Fatalf("checkout: %v", err)
	}

	keepPath := filepath.Join(root, filepath.FromSlash(keep.Path))
	body, _ := os.ReadFile(keepPath)
	os.WriteFile(keepPath, append(body, []byte("\nan added description line\n")...), 0o644)
	if _, err := repo.Output("rm", gone.Path); err != nil {
		t.Fatalf("git rm: %v", err)
	}
	if _, err := repo.Output("add", "-A"); err != nil {
		t.Fatalf("add: %v", err)
	}
	if _, err := repo.Output("commit", "-m", "delete gone, edit keep body"); err != nil {
		t.Fatalf("commit: %v", err)
	}
	if _, err := repo.Output("checkout", mainBranch); err != nil {
		t.Fatalf("checkout back: %v", err)
	}

	res, err := s.Diff("later")
	if err != nil {
		t.Fatalf("Diff: %v", err)
	}
	byNumber := map[string]datamodel.DiffItem{}
	for _, it := range res.Items {
		byNumber[it.Number] = it
	}
	if d := byNumber[gone.Number]; d.Status != datamodel.DiffDeleted {
		t.Fatalf("%s status = %q, want deleted", gone.Number, d.Status)
	}
	k := byNumber[keep.Number]
	if k.Status != datamodel.DiffChanged || len(k.Changes) != 1 || k.Changes[0].Field != datamodel.KeyBody {
		t.Fatalf("%s changes = %+v, want one body change", keep.Number, k.Changes)
	}
	to := k.Changes[0].To
	if !strings.HasSuffix(to, "lines") || strings.HasPrefix(to, "+0/") {
		t.Fatalf("body change summary = %q, want a +N/-M lines count with added lines", to)
	}
}

func TestDiffNonAliasNumberChangeVisible(t *testing.T) {
	root := initGitRepo(t)
	initStore(t, root)
	repo := gitx.Repo{Dir: root}
	s, _ := core.Discover(root)
	cfg, _ := s.Config()

	it, err := s.Create(cfg, core.CreateOpts{Type: datamodel.TypeTicket, Title: "Renum", NoEdit: true})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	mainBranch, _ := repo.Output("branch", "--show-current")
	if _, err := repo.Output("checkout", "-b", "later"); err != nil {
		t.Fatalf("checkout: %v", err)
	}

	path := filepath.Join(root, filepath.FromSlash(it.Path))
	body, _ := os.ReadFile(path)
	hacked := strings.Replace(string(body), "number: "+it.Number+"\n", "number: KIRA-99\n", 1)
	if hacked == string(body) {
		t.Fatalf("could not rewrite number line in %s", it.Path)
	}
	os.WriteFile(path, []byte(hacked), 0o644)
	repo.Output("add", "-A")
	repo.Output("commit", "-m", "hand-edited number, no alias")
	repo.Output("checkout", mainBranch)

	res, err := s.Diff("later")
	if err != nil {
		t.Fatalf("Diff: %v", err)
	}
	if len(res.Items) != 1 {
		t.Fatalf("items = %d, want 1", len(res.Items))
	}
	d := res.Items[0]
	if d.Renumbered != nil {
		t.Fatalf("non-alias-backed number change must not be a RenumberEvent: %+v", d.Renumbered)
	}
	if len(d.Changes) != 1 || d.Changes[0].Field != datamodel.KeyNumber || d.Changes[0].To != "KIRA-99" {
		t.Fatalf("changes = %+v, want a visible number field change to KIRA-99", d.Changes)
	}
}
