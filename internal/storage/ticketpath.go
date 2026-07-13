package storage

import (
	"path"
	"strings"
)

func IsTicketPath(rel string) bool {
	return strings.HasPrefix(rel, dirName+"/tickets/") &&
		strings.HasSuffix(rel, ".md") &&
		!strings.HasPrefix(path.Base(rel), ".")
}
