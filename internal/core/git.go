package core

import (
	"bytes"
	"os/exec"
	"strings"

	"github.com/shivamshivanshu/kira/internal/config"
)

// git runs a git subcommand in the store's root and returns trimmed stdout. It
// is the sole path to git: kira shells out to the user's real git binary for
// behavior parity (docs/design/01-architecture.md §2), never a reimplementation.
func (s *Store) git(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = s.root
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", userErr("git %s: %s", strings.Join(args, " "), msg)
	}
	return strings.TrimSpace(stdout.String()), nil
}

// requireGit fails with an environment error (exit 3) when the git binary is
// not on PATH, matching the exit-code policy for a missing hard dependency.
func requireGit() error {
	if _, err := exec.LookPath("git"); err != nil {
		return envErr("git binary not found on PATH")
	}
	return nil
}

// requireRepo fails with an environment error when root is not inside a git
// work tree, since every mutating command stages (and usually commits) its write.
func (s *Store) requireRepo() error {
	if err := requireGit(); err != nil {
		return err
	}
	if _, err := s.git("rev-parse", "--is-inside-work-tree"); err != nil {
		return envErr("not a git repository: %s", s.root)
	}
	return nil
}

// stage adds exactly the given repo-relative paths to the git index. It never
// runs `git add -A` and never touches unrelated staged files
// (docs/design/03-storage-and-git.md §6).
func (s *Store) stage(paths ...string) error {
	_, err := s.git(append([]string{"add", "--"}, paths...)...)
	return err
}

// commit records a single commit with the given subject and, when trailerVal is
// non-empty, a `<trailerKey>: <trailerVal>` trailer paragraph. Passing subject
// and trailer as separate -m arguments makes git emit them as distinct
// paragraphs, so the trailer parses cleanly (docs/design/03-storage-and-git.md §6).
func (s *Store) commit(subject, trailerKey, trailerVal string) error {
	args := []string{"commit", "-m", subject}
	if trailerVal != "" {
		args = append(args, "-m", trailerKey+": "+trailerVal)
	}
	_, err := s.git(args...)
	return err
}

// finalize stages the written paths and, per the given commit mode, commits
// them. It is the single choke point every mutation routes through — including
// init, which passes an explicit auto mode so the scaffold always commits
// regardless of config — so auto/manual/prompt behavior cannot drift between
// commands (docs/design/03-storage-and-git.md §6).
//
//   - auto: stage, then commit with subject + trailer.
//   - manual: stage only; `kira commit` records it later.
//   - prompt: interactively ask before committing; when not attached to a
//     terminal, degrade to manual (stage only) rather than commit unprompted.
//
// trailerKey/trailerNumber form the commit trailer; an empty trailerNumber
// omits the trailer (e.g. init, which references no single item).
func (s *Store) finalize(mode config.CommitMode, trailerKey, subject, trailerNumber string, paths ...string) error {
	if err := s.stage(paths...); err != nil {
		return err
	}
	switch mode {
	case config.CommitManual:
		return nil
	case config.CommitPrompt:
		if !isInteractive() {
			return nil // cannot ask: stage only, like manual
		}
		if !confirm(subject) {
			return nil
		}
		fallthrough
	default: // auto
		return s.commit(subject, trailerKey, trailerNumber)
	}
}
