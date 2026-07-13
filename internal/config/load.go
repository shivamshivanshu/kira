// Package config loads, validates, and defaults kira's `.kira/config.yaml`.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/storage"
)

func Load(root string) (*datamodel.Config, error) {
	path := filepath.Join(root, storage.ConfigRelPath)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("config: reading %s: %w", path, err)
	}
	return Parse(data)
}

func Parse(data []byte) (*datamodel.Config, error) {
	cfg := Default()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}
	if err := Validate(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
