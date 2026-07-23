package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/shivamshivanshu/kira/internal/config"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/entityschema"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/gitx"
	"github.com/shivamshivanshu/kira/internal/storage"
)

var gitattributesLine = storage.DirName + "/** text eol=lf"

func Init(startDir, key string, force bool, prompter ...Prompter) (*datamodel.InitResult, error) {
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
	s.prompter = firstPrompter(prompter)
	if err := s.requireRepo(); err != nil {
		return nil, err
	}

	fs := s.fs()
	dirName := fs.RelToRoot(fs.KiraDir())
	if fi, err := os.Stat(fs.KiraDir()); err == nil && fi.IsDir() && !force {
		return nil, errx.User("%s already exists", dirName).WithHint("use `--force` to reinitialize over it")
	}

	name := filepath.Base(abs)
	if key != "" && !config.ValidBoardKey(key) {
		return nil, errx.User("project key %q must match %s", key, config.BoardKeyPattern).
			WithHint("keys are 2-10 uppercase letters/digits starting with a letter, e.g. ABC")
	}
	if key == "" {
		def := deriveKey(name)
		if s.prompter.Interactive() {
			key = s.prompter.ReadLine(fmt.Sprintf("project key [%s]: ", def), def)
		} else {
			key = def
		}
	}

	for _, dir := range []string{fs.ItemsDir(), fs.TemplateDir()} {
		if err := os.MkdirAll(dir, dirPerm); err != nil {
			return nil, errx.Env("creating %s: %v", dir, err)
		}
	}

	files := map[string]string{
		fs.ConfigPath(): initConfigYAML(key, name),
		filepath.Join(fs.KiraDir(), ".gitignore"):    storage.CacheDirName + "/\n",
		filepath.Join(fs.TemplateDir(), "ticket.md"): defaultTemplate(datamodel.TypeTicket),
		filepath.Join(fs.TemplateDir(), "epic.md"):   defaultTemplate(datamodel.TypeEpic),
	}
	for path, content := range files {
		if err := os.WriteFile(path, []byte(content), filePerm); err != nil {
			return nil, errx.Env("writing %s: %v", fs.RelToRoot(path), err)
		}
	}

	if _, err := config.Load(abs); err != nil {
		return nil, errx.User("scaffolded config is invalid: %v", err)
	}

	// Unlike files above, WriteDefaults never overwrites an existing schema
	// file: schemas are meant to be user-edited, so even a --force
	// reinitialize must not clobber someone's customization.
	if err := entityschema.WriteDefaults(fs.SchemaDir()); err != nil {
		return nil, errx.Env("writing %s: %v", fs.RelToRoot(fs.SchemaDir()), err)
	}

	if err := ensureGitattributes(filepath.Join(abs, ".gitattributes")); err != nil {
		return nil, err
	}

	if _, err := s.finalize(datamodel.CommitAuto, commitSpec{subject: "kira: init"}, dirName, ".gitattributes"); err != nil {
		return nil, err
	}

	return &datamodel.InitResult{Initialized: true, Path: dirName, ProjectKey: key}, nil
}

func ensureGitattributes(path string) error {
	if err := gitx.AppendLineIfMissing(path, gitattributesLine); err != nil {
		return errx.User("updating .gitattributes: %v", err)
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
