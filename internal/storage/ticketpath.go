package storage

import (
	"path"
	"strings"
)

func IsTicketPath(rel string) bool {
	return strings.HasPrefix(rel, dirName+"/tickets/") && isTicketFilename(path.Base(rel))
}

func isTicketFilename(base string) bool {
	return strings.HasSuffix(base, ".md") && !strings.HasPrefix(base, ".")
}
