// Package config loads, validates, and defaults kira's `.kira/config.yaml`.
package config

import (
	"io"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/storage"
)

// Load reads and parses the repo's .kira/config.yaml.
func Load(root string) (*datamodel.Config, error) {
	data, err := readRepoConfig(root)
	if err != nil {
		return nil, err
	}
	return Parse(data)
}

// LoadWithUser loads the repo config and merges in user preferences from the user config directory.
func LoadWithUser(root string, env func(string) string, warn io.Writer) (*datamodel.Config, error) {
	data, err := readRepoConfig(root)
	if err != nil {
		return nil, err
	}
	tier := readUserTier(env, warn)
	cfg := Default()
	if tier.ui != nil {
		cfg.UI = *tier.ui
	}
	if tier.workon != nil {
		cfg.Workon = *tier.workon
	}
	ignore := ignorer(warn, filepath.Join(root, storage.ConfigRelPath))
	if err := parseInto(cfg, data, ignore); err != nil {
		return nil, err
	}
	alreadyWarned := make(map[string]bool, len(tier.uiWarnings))
	for _, w := range tier.uiWarnings {
		alreadyWarned[w] = true
	}
	for _, w := range UIWarnings(cfg.UI) {
		if !alreadyWarned[w] {
			ignore("%s", w)
		}
	}
	cfg.UserAutomation = tier.hooks
	if tier.commit != nil {
		cfg.UserCommitSubject = tier.commit.Subject
	}
	return cfg, nil
}

func readRepoConfig(root string) ([]byte, error) {
	path := filepath.Join(root, storage.ConfigRelPath)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, errx.User("config: reading %s: %w", path, err)
	}
	return data, nil
}

// Parse parses raw YAML bytes into a Config structure with defaults applied.
func Parse(data []byte) (*datamodel.Config, error) {
	cfg := Default()
	if err := parseInto(cfg, data, nil); err != nil {
		return nil, err
	}
	return cfg, nil
}

func parseInto(cfg *datamodel.Config, data []byte, ignore ignoreFunc) error {
	restoreUserEditor := preserveUserEditor(cfg, data, ignore)
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return errx.User("config: %w", err)
	}
	restoreUserEditor()
	return Validate(cfg)
}

func preserveUserEditor(cfg *datamodel.Config, data []byte, ignore ignoreFunc) func() {
	userEditor := cfg.UI.Editor
	if ignore != nil && repoDocSetsUIEditor(data) {
		ignore("ui.editor is personal; set it in ~/.config/kira/config.yaml")
	}
	return func() { cfg.UI.Editor = userEditor }
}

func repoDocSetsUIEditor(data []byte) bool {
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil || len(doc.Content) == 0 {
		return false
	}
	_, ui := findTopLevel(&doc, userKeyUI)
	if ui == nil || ui.Kind != yaml.MappingNode {
		return false
	}
	return childValue(ui, "editor") != nil
}
