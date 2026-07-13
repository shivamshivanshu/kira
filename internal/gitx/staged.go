package gitx

func (r Repo) StagedPaths() ([]string, error) {
	return r.splitLines("diff", "--cached", "--name-only", "--diff-filter=ACM")
}

func (r Repo) ShowStaged(path string) (string, error) {
	return r.OutputRaw("show", ":"+path)
}

func (r Repo) DirtyPaths(pathspecs ...string) ([]string, error) {
	out, err := r.OutputRaw(append([]string{"status", "--porcelain", "--"}, pathspecs...)...)
	if err != nil {
		return nil, err
	}
	return parsePorcelainPaths(out), nil
}
