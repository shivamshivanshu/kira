package core

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/shivamshivanshu/kira/internal/config"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/termx"
)

const gitattributesLine = ".kira/** text eol=lf"

func Init(startDir, key string, force bool) (*datamodel.InitResult, error) {
	root := startDir
	if root == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, errx.Env("cannot determine working directory: %v", err)
		}
		root = cwd
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, errx.Env("resolving %q: %v", root, err)
	}
	s := newStore(abs)
	if err := s.requireRepo(); err != nil {
		return nil, err
	}

	fs := s.fs()
	dirName := fs.RelToRoot(fs.KiraDir())
	if fi, err := os.Stat(fs.KiraDir()); err == nil && fi.IsDir() && !force {
		return nil, errx.User("%s already exists", dirName).WithHint("use `--force` to reinitialize over it")
	}

	name := filepath.Base(abs)
	if key == "" {
		def := deriveKey(name)
		if termx.IsInteractive() {
			key = termx.ReadLineDefault(fmt.Sprintf("project key [%s]: ", def), def)
		} else {
			key = def
		}
	}

	for _, dir := range []string{fs.ItemsDir(), fs.TemplateDir()} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, errx.User("creating %s: %v", dir, err)
		}
	}

	files := map[string]string{
		fs.ConfigPath(): initConfigYAML(key, name),
		filepath.Join(fs.KiraDir(), ".gitignore"):    ".cache/\n",
		filepath.Join(fs.TemplateDir(), "ticket.md"): defaultTemplate(datamodel.TypeTicket),
		filepath.Join(fs.TemplateDir(), "epic.md"):   defaultTemplate(datamodel.TypeEpic),
	}
	for path, content := range files {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			return nil, errx.User("writing %s: %v", fs.RelToRoot(path), err)
		}
	}

	if _, err := config.Load(abs); err != nil {
		return nil, errx.User("scaffolded config is invalid: %v", err)
	}

	if err := ensureGitattributes(filepath.Join(abs, ".gitattributes")); err != nil {
		return nil, err
	}

	if err := s.finalize(datamodel.CommitAuto, "", "kira: init", "", dirName, ".gitattributes"); err != nil {
		return nil, err
	}

	return &datamodel.InitResult{Initialized: true, Path: dirName, ProjectKey: key}, nil
}

func ensureGitattributes(path string) error {
	existing, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return errx.User("reading .gitattributes: %v", err)
	}
	content := string(existing)
	for _, line := range strings.Split(content, "\n") {
		if strings.TrimSpace(line) == gitattributesLine {
			return nil
		}
	}
	if content != "" && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	content += gitattributesLine + "\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return errx.User("writing .gitattributes: %v", err)
	}
	return nil
}

func deriveKey(name string) string {
	var b strings.Builder
	for _, r := range strings.ToUpper(name) {
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	if b.Len() == 0 {
		return "KIRA"
	}
	return b.String()
}
