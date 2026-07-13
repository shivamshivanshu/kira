package core_test

import (
	"testing"

	"github.com/shivamshivanshu/kira/internal/config"
	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/testutil"
)

func TestConfigTemplateParses(t *testing.T) {
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
