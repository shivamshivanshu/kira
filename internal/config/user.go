package config

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
)

const (
	userConfigDirName  = "kira"
	userConfigFileName = "config.yaml"
	userHooksYAMLName  = "hooks.yaml"

	userKeyUI         = "ui"
	userKeyWorkon     = "workon"
	userKeyCommit     = "commit"
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
	ui         *datamodel.UI
	workon     *datamodel.Workon
	commit     *userCommit
	hooks      []datamodel.AutomationHook
	uiWarnings []string
}

type userCommit struct {
	Subject string `yaml:"subject"`
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
	tier := readUserPrefs(filepath.Join(dir, userConfigFileName), warn)
	tier.hooks = readUserHooks(dir, warn)
	return tier
}

func readUserPrefs(path string, warn io.Writer) userTier {
	ignore := ignorer(warn, path)
	root, ok := readMapping(path, ignore)
	if !ok {
		return userTier{}
	}
	var uiNode, workonNode, commitNode *yaml.Node
	for i := 0; i+1 < len(root.Content); i += 2 {
		switch key := root.Content[i].Value; key {
		case userKeyUI:
			uiNode = root.Content[i+1]
		case userKeyWorkon:
			workonNode = root.Content[i+1]
		case userKeyCommit:
			commitNode = root.Content[i+1]
		case userKeyAutomation:
			ignore("automation: personal hooks belong in %s", userHooksYAMLName)
		default:
			ignore("key %q is repo-authoritative", key)
		}
	}
	def := Default()
	tier := userTier{
		ui:     decodeUserSection(uiNode, def.UI, userKeyUI, validateUISection, ignore),
		workon: decodeUserSection(workonNode, def.Workon, userKeyWorkon, validateWorkonSection, ignore),
		commit: decodeUserSection(commitNode, userCommit{}, userKeyCommit, validateUserCommit, ignore),
	}
	if tier.ui != nil {
		tier.uiWarnings = UIWarnings(*tier.ui)
		for _, w := range tier.uiWarnings {
			ignore("%s", w)
		}
	}
	return tier
}

func validateUserCommit(c userCommit) error {
	if strings.ContainsAny(c.Subject, "\n\r") {
		return errx.User("commit.subject: must be a single line")
	}
	return nil
}

func decodeUserSection[T any](node *yaml.Node, def T, label string, validate func(T) error, ignore ignoreFunc) *T {
	if node == nil {
		return nil
	}
	v := def
	if err := node.Decode(&v); err != nil {
		ignore("%s: %v", label, err)
		return nil
	}
	if err := validate(v); err != nil {
		ignore("%v", err)
		return nil
	}
	return &v
}

func readUserHooks(dir string, warn io.Writer) []datamodel.AutomationHook {
	path := filepath.Join(dir, userHooksYAMLName)
	ignore := ignorer(warn, path)
	data, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			ignore("%v", err)
		}
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
