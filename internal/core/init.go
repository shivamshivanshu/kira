package core

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/shivamshivanshu/kira/internal/config"
	"github.com/shivamshivanshu/kira/internal/item"
)

// InitResult is the --json shape of a successful init.
type InitResult struct {
	Initialized bool   `json:"initialized"`
	Path        string `json:"path"`
	ProjectKey  string `json:"project_key"`
}

// Init scaffolds .kira/ under startDir (cwd when empty): config.yaml, tickets/,
// templates/{ticket,epic}.md, and .gitignore (.cache/), then records the initial
// `kira: init` commit. init always commits, regardless of commit.mode, so the
// scaffold is never left uncommitted (docs/design/04-cli.md init). key empty
// derives a default from the directory name (prompted first when interactive).
func Init(startDir, key string, force bool) (*InitResult, error) {
	root := startDir
	if root == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, envErr("cannot determine working directory: %v", err)
		}
		root = cwd
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, envErr("resolving %q: %v", root, err)
	}
	s := &Store{root: abs}
	if err := s.requireRepo(); err != nil {
		return nil, err
	}

	if fi, err := os.Stat(s.kiraDir()); err == nil && fi.IsDir() && !force {
		return nil, userErr("%s already exists (use --force to reinitialize)", dirName)
	}

	name := filepath.Base(abs)
	if key == "" {
		def := deriveKey(name)
		if isInteractive() {
			key = promptKey(def)
		} else {
			key = def
		}
	}

	for _, dir := range []string{s.ticketsDir(), s.templateDir()} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, userErr("creating %s: %v", dir, err)
		}
	}

	files := map[string]string{
		s.configPath():                              initConfigYAML(key, name),
		filepath.Join(s.kiraDir(), ".gitignore"):    ".cache/\n",
		filepath.Join(s.templateDir(), "ticket.md"): defaultTemplate(item.TypeTicket),
		filepath.Join(s.templateDir(), "epic.md"):   defaultTemplate(item.TypeEpic),
	}
	for path, content := range files {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			return nil, userErr("writing %s: %v", s.relToRoot(path), err)
		}
	}

	// Verify the scaffolded config parses and validates before committing it.
	if _, err := config.Load(abs); err != nil {
		return nil, userErr("scaffolded config is invalid: %v", err)
	}

	// init always commits, regardless of config commit.mode, so the scaffold is
	// never left uncommitted; route it through the shared choke point with an
	// explicit auto mode.
	if err := s.finalize(config.CommitAuto, "", "kira: init", "", dirName); err != nil {
		return nil, err
	}

	return &InitResult{Initialized: true, Path: dirName, ProjectKey: key}, nil
}

// deriveKey turns a directory name into a default project key: uppercased,
// non-alphanumeric characters dropped. Falls back to KIRA when nothing usable
// remains.
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
