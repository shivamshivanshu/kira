package config

import (
	"embed"
	"os"
	"path/filepath"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
)

//go:embed templates/config.yaml templates/hooks.yaml
var userTemplateFS embed.FS

func mustTemplate(name string) string {
	data, err := userTemplateFS.ReadFile("templates/" + name)
	if err != nil {
		panic("config: missing embedded template " + name)
	}
	return string(data)
}

// InitUser initializes the user's kira config directory and returns the result.
func InitUser(env func(string) string) (*datamodel.ConfigInitResult, error) {
	dir, ok := UserConfigDir(env)
	if !ok {
		return nil, errx.Env("cannot resolve user config directory: set HOME or XDG_CONFIG_HOME")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, errx.Env("creating %s: %v", dir, err)
	}
	if present := PresentUserFiles(dir); len(present) > 0 {
		return &datamodel.ConfigInitResult{Path: dir, Created: false, Files: present}, nil
	}
	written := make([]string, 0, 2)
	for _, name := range []string{userConfigFileName, userHooksYAMLName} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(mustTemplate(name)), 0o644); err != nil {
			return nil, errx.Env("writing %s: %v", filepath.Join(dir, name), err)
		}
		written = append(written, name)
	}
	return &datamodel.ConfigInitResult{Path: dir, Created: true, Files: written}, nil
}

// PresentUserFiles returns which kira config files exist in the given directory.
func PresentUserFiles(dir string) []string {
	present := make([]string, 0, 2)
	for _, name := range []string{userConfigFileName, userHooksYAMLName} {
		if fileExists(filepath.Join(dir, name)) {
			present = append(present, name)
		}
	}
	return present
}
