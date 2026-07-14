package gitx

func (r Repo) StagedPaths() ([]string, error) {
	lines, err := r.splitLines("diff", "--cached", "--name-only", "--diff-filter=ACM")
	if err != nil {
		return nil, err
	}
	return r.relToDir(lines)
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
