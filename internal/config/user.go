package config

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/shivamshivanshu/kira/internal/datamodel"
)

const (
	userConfigDirName  = "kira"
	userConfigFileName = "config.yaml"
	userHooksYAMLName  = "hooks.yaml"

	userKeyUI         = "ui"
	userKeyAutomation = "automation"
)

func UserConfigDir(env func(string) string) (string, bool) {
	if xdg := env("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, userConfigDirName), true
	}
	if home := env("HOME"); home != "" {
		return filepath.Join(home, ".config", userConfigDirName), true
	}
	return "", false
}

type userTier struct {
	ui    *datamodel.UI
	hooks []datamodel.AutomationHook
}

type ignoreFunc func(format string, args ...any)

func ignorer(warn io.Writer, path string) ignoreFunc {
	return func(format string, args ...any) {
		fmt.Fprintf(warn, "%s: %s; ignored\n", path, fmt.Sprintf(format, args...))
	}
}

func readUserTier(env func(string) string, warn io.Writer) userTier {
	dir, ok := UserConfigDir(env)
	if !ok {
		return userTier{}
	}
	return userTier{
		ui:    readUserPrefs(filepath.Join(dir, userConfigFileName), warn),
		hooks: readUserHooks(dir, warn),
	}
}

func readUserPrefs(path string, warn io.Writer) *datamodel.UI {
	ignore := ignorer(warn, path)
	root, ok := readMapping(path, ignore)
	if !ok {
		return nil
	}
	var uiNode *yaml.Node
	for i := 0; i+1 < len(root.Content); i += 2 {
		switch key := root.Content[i].Value; key {
		case userKeyUI:
			uiNode = root.Content[i+1]
		case userKeyAutomation:
			ignore("automation: personal hooks belong in %s", userHooksYAMLName)
		default:
			ignore("key %q is repo-authoritative", key)
		}
	}
	if uiNode == nil {
		return nil
	}
	ui := Default().UI
	if err := uiNode.Decode(&ui); err != nil {
		ignore("ui: %v", err)
		return nil
	}
	if err := validateUISection(ui); err != nil {
		ignore("%v", err)
		return nil
	}
	return &ui
}

func readUserHooks(dir string, warn io.Writer) []datamodel.AutomationHook {
	path := filepath.Join(dir, userHooksYAMLName)
	if !fileExists(path) {
		return nil
	}
	ignore := ignorer(warn, path)
	data, err := os.ReadFile(path)
	if err != nil {
		ignore("%v", err)
		return nil
	}
	var hooks []datamodel.AutomationHook
	if err := yaml.Unmarshal(data, &hooks); err != nil {
		ignore("%v", err)
		return nil
	}
	if err := validateAutomationHooks("hooks", hooks); err != nil {
		ignore("%v", err)
		return nil
	}
	return hooks
}

func readMapping(path string, ignore ignoreFunc) (*yaml.Node, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			ignore("%v", err)
		}
		return nil, false
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		ignore("%v", err)
		return nil, false
	}
	if len(doc.Content) == 0 {
		return nil, false
	}
	if doc.Content[0].Kind != yaml.MappingNode {
		ignore("top level must be a mapping")
		return nil, false
	}
	return doc.Content[0], true
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
