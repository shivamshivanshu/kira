package index

import (
	"encoding/json"
	"os"

	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/storage"
)

type skipEntry struct {
	Note       string `json:"note"`
	ConflictID string `json:"conflict_id,omitempty"`
}

// meta is the previous run's watermark set, and the only reason a reindex can
// decide to do nothing. LastIndexedHeadSHA plus DirtyHash cover the item
// files; TrailerWatermarks covers the commit scans separately, keyed by the
// literal "HEAD" for the trailer scan and by ref name for each close scan,
// because those advance independently of the item files.
type meta struct {
	SchemaVersion      int                  `json:"schema_version"`
	ScanConfigHash     string               `json:"scan_config_hash,omitempty"`
	LastIndexedHeadSHA string               `json:"last_indexed_head_sha"`
	DirtyHash          string               `json:"dirty_hash"`
	DirtyPaths         []string             `json:"dirty_paths"`
	TrailerWatermarks  map[string]string    `json:"trailer_watermarks,omitempty"`
	Skipped            map[string]skipEntry `json:"skipped,omitempty"`
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
	if err := storage.WriteFileAtomic(metaPath(cacheDir), data); err != nil {
		return errx.Env("writing index meta: %v", err)
	}
	return nil
}
