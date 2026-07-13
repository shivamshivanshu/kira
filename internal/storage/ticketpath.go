package storage

import (
	"path"
	"strings"
)

func IsItemPath(rel string) bool {
	return strings.HasPrefix(rel, TicketsPrefix+"/") && isItemFilename(path.Base(rel))
}

func isItemFilename(base string) bool {
	return strings.HasSuffix(base, itemExt) && !strings.HasPrefix(base, ".")
}
