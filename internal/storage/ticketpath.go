package storage

import (
	"path"
	"strings"

	"github.com/shivamshivanshu/kira/internal/id"
)

const StrayMessage = "stray file in tickets dir; not a valid ULID-named item"

func IsItemPath(rel string) bool {
	return path.Dir(rel) == TicketsPrefix && isItemFilename(path.Base(rel))
}

// UnderTicketsDir reports whether rel names anything inside the tickets
// dir, including strays that IsItemPath rejects (nested paths, non-ULID
// filenames) — use it to catch stray files that IsItemPath alone would
// silently ignore.
func UnderTicketsDir(rel string) bool {
	return strings.HasPrefix(rel, TicketsPrefix+"/")
}

func isItemFilename(base string) bool {
	if strings.HasPrefix(base, ".") {
		return false
	}
	stem, ok := strings.CutSuffix(base, itemExt)
	if !ok {
		return false
	}
	_, err := id.ParseULIDLoose(stem)
	return err == nil
}
