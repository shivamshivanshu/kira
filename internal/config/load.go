// Package config loads, validates, and defaults kira's `.kira/config.yaml`.
package config

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/storage"
)

func Load(root string) (*datamodel.Config, error) {
	data, err := readRepoConfig(root)
	if err != nil {
		return nil, err
	}
	return Parse(data)
}

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
	if err := parseInto(cfg, data); err != nil {
		return nil, err
	}
	ignore := ignorer(warn, filepath.Join(root, storage.ConfigRelPath))
	for _, w := range UIWarnings(cfg.UI) {
		ignore("%s", w)
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
		return nil, fmt.Errorf("config: reading %s: %w", path, err)
	}
	return data, nil
}

func Parse(data []byte) (*datamodel.Config, error) {
	cfg := Default()
	if err := parseInto(cfg, data); err != nil {
		return nil, err
	}
	return cfg, nil
}

func parseInto(cfg *datamodel.Config, data []byte) error {
	userEditor := cfg.UI.Editor
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return fmt.Errorf("config: %w", err)
	}
	cfg.UI.Editor = userEditor
	return Validate(cfg)
}
