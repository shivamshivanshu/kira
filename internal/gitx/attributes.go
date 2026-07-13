package gitx

import (
	"os"
	"strings"
)

func (r Repo) InfoAttributeHasLine(line string) bool {
	path, err := r.GitPath("info/attributes")
	if err != nil {
		return false
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	for _, l := range strings.Split(string(content), "\n") {
		if strings.TrimSpace(l) == line {
			return true
		}
	}
	return false
}
