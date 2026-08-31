package gitx

func (r Repo) StagedPaths() ([]string, error) {
	lines, err := r.splitLines("diff", "--cached", "--name-only", "--diff-filter=ACM")
	if err != nil {
		return nil, err
	}
	return r.relToDir(lines)
}

// HasStagedChanges reports whether the index differs from HEAD at all,
// including deletions that StagedPaths' ACM filter would miss.
func (r Repo) HasStagedChanges() (bool, error) {
	lines, err := r.splitLines("diff", "--cached", "--name-only")
	if err != nil {
		return false, err
	}
	return len(lines) > 0, nil
}

func (r Repo) ShowStaged(path string) (string, error) {
	return r.OutputRaw("show", RevPath("", path))
}

func (r Repo) DirtyPaths(pathspecs ...string) ([]string, error) {
	out, err := r.OutputRaw(append([]string{"status", "--porcelain", "--untracked-files=all", "--"}, pathspecs...)...)
	if err != nil {
		return nil, err
	}
	return r.relToDir(parsePorcelainPaths(out))
}
