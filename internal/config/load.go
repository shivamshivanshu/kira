// Package config loads, validates, and defaults kira's `.kira/config.yaml`.
package config

import (
	"io"
	"os"
	"path/filepath"
	"reflect"

	"gopkg.in/yaml.v3"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/setx"
	"github.com/shivamshivanshu/kira/internal/storage"
)

// Load reads and parses the repo's .kira/config.yaml.
func Load(root string) (*datamodel.Config, error) {
	data, err := readRepoConfig(root)
	if err != nil {
		return nil, err
	}
	cfg, err := Parse(data)
	if err != nil {
		return nil, err
	}
	return cfg, requireCurrentSchema(cfg)
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
	uiBeforeRepo, workonBeforeRepo := cfg.UI, cfg.Workon
	ignore := ignorer(warn, filepath.Join(root, storage.ConfigRelPath))
	if err := parseInto(cfg, data, ignore); err != nil {
		return nil, err
	}
	if err := requireCurrentSchema(cfg); err != nil {
		return nil, err
	}
	cfg.UIProvenance = tierFor(!reflect.DeepEqual(cfg.UI, uiBeforeRepo), tier.ui != nil)
	cfg.WorkonProvenance = tierFor(!reflect.DeepEqual(cfg.Workon, workonBeforeRepo), tier.workon != nil)
	alreadyWarned := setx.ToSet(tier.uiWarnings)
	for _, w := range UIWarnings(cfg.UI) {
		if !alreadyWarned[w] {
			ignore("%s", w)
		}
	}
	if tier.commit != nil {
		cfg.UserCommitSubject = tier.commit.Subject
	}
	return cfg, nil
}

// requireCurrentSchema refuses a pre-boards config in the working tree. Parse
// still accepts one, so configs read from historical git trees keep loading.
func requireCurrentSchema(cfg *datamodel.Config) error {
	if cfg.Version == datamodel.SchemaVersion && len(cfg.Boards) > 0 {
		return nil
	}
	addBoards := "add boards: [" + defaultBoardEntry(cfg) + "]"
	if cfg.Version != datamodel.SchemaVersion {
		return errx.User("config: version %d is no longer supported", cfg.Version).
			WithHint("set version: %d and %s", datamodel.SchemaVersion, addBoards)
	}
	return errx.User("config: boards: at least one board is required").WithHint("%s", addBoards)
}

func defaultBoardEntry(cfg *datamodel.Config) string {
	entry, err := InlineBoardEntry(datamodel.Board{Key: cfg.Project.Key, Name: cfg.Project.Name, Default: true})
	if err != nil {
		return "{ key: <KEY>, name: <name>, default: true }"
	}
	return entry
}

// tierFor reports which tier supplied a section's effective value: repo when
// the repo overlay changed it, else user when the user tier set it.
func tierFor(repoChanged, userSet bool) datamodel.Provenance {
	switch {
	case repoChanged:
		return datamodel.ProvenanceRepo
	case userSet:
		return datamodel.ProvenanceUser
	default:
		return datamodel.ProvenanceDefault
	}
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
