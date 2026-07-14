package gitx

import "os"

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
	return containsLine(string(content), line)
}
