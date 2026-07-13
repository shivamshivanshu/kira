package gitx

import (
	"os"
	"strings"
)

const infoAttributesPath = "info/attributes"

func (r Repo) InfoAttributeHasLine(line string) bool {
	path, err := r.GitPath(infoAttributesPath)
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
