package gitx

import "strings"

func (r Repo) StagedPaths() ([]string, error) {
	out, err := r.Output("diff", "--cached", "--name-only", "--diff-filter=ACM")
	if err != nil {
		return nil, err
	}
	if out == "" {
		return nil, nil
	}
	return strings.Split(out, "\n"), nil
}

func (r Repo) ShowStaged(path string) (string, error) {
	return r.OutputRaw("show", ":"+path)
}
func (r Repo) DirtyPaths(pathspecs ...string) ([]string, error) {
	out, err := r.Output(append([]string{"status", "--porcelain", "--"}, pathspecs...)...)
	if err != nil {
		return nil, err
	}
	var paths []string
	for _, line := range strings.Split(out, "\n") {
		if len(line) > 3 {
			paths = append(paths, strings.TrimSpace(line[3:]))
		}
	}
	return paths, nil
}
