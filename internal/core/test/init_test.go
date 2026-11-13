package core_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/shivamshivanshu/kira/internal/config"
	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/testutil"
)

func TestConfigTemplateParses(t *testing.T) {
	t.Parallel()
	root := testutil.InitGitRepo(t)
	if _, err := core.Init(root, "ACME", false); err != nil {
		t.Fatalf("Init: %v", err)
	}
	cfg, err := config.Load(root)
	if err != nil {
		t.Fatalf("scaffolded config must parse: %v", err)
	}
	if cfg.Project.Key != "ACME" {
		t.Fatalf("key = %q, want ACME", cfg.Project.Key)
	}
	if len(cfg.Labels.Known) != 0 || len(cfg.People.Known) != 0 {
		t.Fatalf("init must seed empty vocab, got labels=%v people=%v", cfg.Labels.Known, cfg.People.Known)
	}
	if cfg.Commit.Mode != datamodel.CommitAuto {
		t.Fatalf("commit mode = %q, want auto", cfg.Commit.Mode)
	}
}

type keyPrompter struct{ key string }

func (keyPrompter) Interactive() bool             { return true }
func (p keyPrompter) ReadLine(_, _ string) string { return p.key }

func initRepoNamed(t *testing.T, name string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	testutil.InitializeRepository(t, dir)
	return dir
}

func TestInitDerivedKeyTruncatesToValidKey(t *testing.T) {
	t.Parallel()
	root := initRepoNamed(t, "my-superlong-project")
	res, err := core.Init(root, "", false)
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	if res.ProjectKey != "MYSUPERLON" {
		t.Fatalf("derived key = %q, want MYSUPERLON", res.ProjectKey)
	}
	if !config.ValidBoardKey(res.ProjectKey) {
		t.Fatalf("derived key %q violates %s", res.ProjectKey, config.BoardKeyPattern)
	}
}

func TestInitDerivedKeyFallsBackWhenNameYieldsNoValidKey(t *testing.T) {
	t.Parallel()
	for _, name := range []string{"9-9", "x"} {
		root := initRepoNamed(t, name)
		res, err := core.Init(root, "", false)
		if err != nil {
			t.Fatalf("Init in %q: %v", name, err)
		}
		if res.ProjectKey != "KIRA" {
			t.Fatalf("derived key in %q = %q, want KIRA fallback", name, res.ProjectKey)
		}
	}
}

func TestInitRejectsInvalidPromptedKeyBeforeWriting(t *testing.T) {
	t.Parallel()
	root := testutil.InitGitRepo(t)
	_, err := core.Init(root, "", false, keyPrompter{key: "bad key!"})
	if err == nil {
		t.Fatal("Init accepted invalid prompted key")
	}
	if _, statErr := os.Stat(filepath.Join(root, ".kira")); !os.IsNotExist(statErr) {
		t.Fatalf(".kira must not be scaffolded on invalid key, stat err = %v", statErr)
	}
}

func TestInitAcceptsValidPromptedKey(t *testing.T) {
	t.Parallel()
	root := testutil.InitGitRepo(t)
	res, err := core.Init(root, "", false, keyPrompter{key: "ACME9"})
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	if res.ProjectKey != "ACME9" {
		t.Fatalf("key = %q, want ACME9", res.ProjectKey)
	}
}
