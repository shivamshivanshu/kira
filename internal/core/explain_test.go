package core

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/shivamshivanshu/kira/internal/config"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/storage"
)

func loadExplainConfig(t *testing.T, repoBody, userBody string) *datamodel.Config {
	t.Helper()
	root := t.TempDir()
	repoPath := filepath.Join(root, storage.ConfigRelPath)
	if err := os.MkdirAll(filepath.Dir(repoPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(repoPath, []byte(repoBody), 0o644); err != nil {
		t.Fatal(err)
	}
	home := t.TempDir()
	if userBody != "" {
		userDir := filepath.Join(home, ".config", "kira")
		if err := os.MkdirAll(userDir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(userDir, "config.yaml"), []byte(userBody), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	env := func(k string) string {
		if k == "HOME" {
			return home
		}
		return ""
	}
	var warn bytes.Buffer
	cfg, err := config.LoadWithUser(root, env, &warn)
	if err != nil {
		t.Fatalf("LoadWithUser: %v", err)
	}
	return cfg
}

func sectionProvenance(t *testing.T, res *datamodel.ExplainResult, name string) datamodel.Provenance {
	t.Helper()
	for _, sec := range res.Sections {
		if sec.Name == name {
			return sec.Provenance
		}
	}
	t.Fatalf("no %q section in %+v", name, res.Sections)
	return ""
}

func TestExplainProvenanceFollowsSupplyingTier(t *testing.T) {
	t.Parallel()
	const repoBase = "version: 1\nproject:\n  key: KIRA\n"
	cases := []struct {
		name       string
		repo, user string
		section    string
		want       datamodel.Provenance
	}{
		{"repo-set-ui", repoBase + "ui:\n  icons: emoji\n", "", "ui", datamodel.ProvenanceRepo},
		{"user-set-ui", repoBase, "ui:\n  icons: emoji\n", "ui", datamodel.ProvenanceUser},
		{"default-ui", repoBase, "", "ui", datamodel.ProvenanceDefault},
		{"repo-set-workon", repoBase + "workon:\n  branch_pattern: repo/{number}\n", "workon:\n  branch_pattern: user/{number}\n", "workon", datamodel.ProvenanceRepo},
		{"user-set-workon", repoBase, "workon:\n  branch_pattern: user/{number}\n", "workon", datamodel.ProvenanceUser},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			cfg := loadExplainConfig(t, c.repo, c.user)
			if got := sectionProvenance(t, Explain(cfg), c.section); got != c.want {
				t.Errorf("%s provenance = %q, want %q", c.section, got, c.want)
			}
		})
	}
}
