package storage

import (
	"path"
	"strings"
)

func IsItemPath(rel string) bool {
	return strings.HasPrefix(rel, dirName+"/tickets/") && isItemFilename(path.Base(rel))
}

func isItemFilename(base string) bool {
	return strings.HasSuffix(base, ".md") && !strings.HasPrefix(base, ".")
}
