package index

import (
	"encoding/json"
	"os"

	"github.com/shivamshivanshu/kira/internal/errx"
)

type meta struct {
	SchemaVersion      int               `json:"schema_version"`
	LastIndexedHeadSHA string            `json:"last_indexed_head_sha"`
	DirtyHash          string            `json:"dirty_hash"`
	DirtyPaths         []string          `json:"dirty_paths"`
	TrailerWatermarks  map[string]string `json:"trailer_watermarks,omitempty"`
}

func (i *Index) loadMeta() (meta, bool) { return loadMetaAt(i.cacheDir) }

func (i *Index) saveMeta(m meta) error { return saveMetaAt(i.cacheDir, m) }

func loadMetaAt(cacheDir string) (meta, bool) {
	data, err := os.ReadFile(metaPath(cacheDir))
	if err != nil {
		return meta{}, false
	}
	var m meta
	if err := json.Unmarshal(data, &m); err != nil {
		return meta{}, false
	}
	if m.SchemaVersion != schemaVersion {
		return meta{}, false
	}
	return m, true
}

func saveMetaAt(cacheDir string, m meta) error {
	m.SchemaVersion = schemaVersion
	if m.DirtyPaths == nil {
		m.DirtyPaths = []string{}
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return errx.User("encoding index meta: %v", err)
	}
	tmp := metaPath(cacheDir) + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return errx.User("writing index meta: %v", err)
	}
	if err := os.Rename(tmp, metaPath(cacheDir)); err != nil {
		os.Remove(tmp)
		return errx.User("renaming index meta: %v", err)
	}
	return nil
}
