package core

import (
	"path/filepath"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/entityschema"
	"github.com/shivamshivanshu/kira/internal/errx"
)

// MaterializeSchemas writes the built-in schemas a repo is missing and returns
// their repo-relative paths. Existing files are left alone, so a repo that
// predates schema files can adopt them without losing local edits.
func (s *Store) MaterializeSchemas(cfg *datamodel.Config) ([]string, error) {
	fs := s.fs()
	release, err := fs.Lock()
	if err != nil {
		return nil, err
	}
	defer release()

	names, err := entityschema.WriteDefaults(fs.SchemaDir())
	if err != nil {
		return nil, errx.Env("writing %s: %v", fs.RelToRoot(fs.SchemaDir()), err)
	}
	if len(names) == 0 {
		return []string{}, nil
	}
	paths := make([]string, len(names))
	for i, n := range names {
		paths[i] = fs.RelToRoot(filepath.Join(fs.SchemaDir(), n))
	}
	// Restoring a file that was deleted from the worktree but is still in HEAD
	// writes it back byte-identical, leaving nothing to commit. Stage first and
	// check these paths specifically -- an unscoped index check would see a
	// user's unrelated staged work and commit anyway. finalize stages these
	// paths again, which is why staging here costs nothing.
	if err := s.repo().Stage(paths...); err != nil {
		return nil, errx.User("%s", err)
	}
	staged, err := s.repo().HasStagedChangesIn(paths...)
	if err != nil {
		return nil, errx.User("%s", err)
	}
	if !staged {
		return paths, nil
	}
	if _, err := s.finalize(cfg.Commit.Mode, commitSpec{
		trailerKey: cfg.Commit.Trailer,
		subject:    cfg.Commit.SubjectPrefix + "materialize built-in schemas",
	}, paths...); err != nil {
		return nil, err
	}
	return paths, nil
}
