package index

import (
	"fmt"

	"github.com/shivamshivanshu/kira/internal/errx"
)

const schemaVersion = 4

const ddl = `
CREATE TABLE IF NOT EXISTS items (
  id       TEXT PRIMARY KEY,
  number   TEXT NOT NULL,
  type     TEXT NOT NULL,
  subtype  TEXT,
  title    TEXT NOT NULL,
  state    TEXT NOT NULL,
  resolution TEXT,
  priority TEXT,
  rank     TEXT,
  owner    TEXT,
  reporter TEXT,
  epic     TEXT,
  sprint   TEXT,
  due      TEXT,
  estimate REAL,
  created  TEXT NOT NULL,
  updated  TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS aliases (
  item_id TEXT NOT NULL REFERENCES items(id) ON DELETE CASCADE,
  ord     INTEGER NOT NULL,
  number  TEXT NOT NULL,
  PRIMARY KEY (item_id, ord)
);
CREATE TABLE IF NOT EXISTS labels (
  item_id TEXT NOT NULL REFERENCES items(id) ON DELETE CASCADE,
  ord     INTEGER NOT NULL,
  label   TEXT NOT NULL,
  PRIMARY KEY (item_id, ord)
);
CREATE INDEX IF NOT EXISTS idx_labels_label ON labels(label);
CREATE TABLE IF NOT EXISTS links (
  item_id   TEXT NOT NULL REFERENCES items(id) ON DELETE CASCADE,
  ord       INTEGER NOT NULL,
  kind      TEXT NOT NULL,
  target_id TEXT NOT NULL,
  PRIMARY KEY (item_id, ord)
);
CREATE TABLE IF NOT EXISTS commit_links (
  item_id TEXT NOT NULL,
  sha     TEXT NOT NULL,
  subject TEXT NOT NULL,
  author  TEXT NOT NULL,
  ts      TEXT NOT NULL,
  source  TEXT NOT NULL,
  PRIMARY KEY (item_id, sha)
);
CREATE INDEX IF NOT EXISTS idx_commit_links_sha ON commit_links(sha);
CREATE TABLE IF NOT EXISTS event_heads (
  item_id  TEXT PRIMARY KEY,
  head_sha TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS events (
  item_id    TEXT NOT NULL,
  seq        INTEGER NOT NULL,
  ts         TEXT NOT NULL,
  field      TEXT NOT NULL,
  old_value  TEXT NOT NULL,
  new_value  TEXT NOT NULL,
  commit_sha TEXT NOT NULL,
  PRIMARY KEY (item_id, seq)
);
`

func (i *Index) ensureSchema() error {
	var version int
	if err := i.db.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		return errx.User("reading index schema version: %v", err)
	}
	if version == schemaVersion {
		return nil
	}
	if version != 0 {
		return errx.User("index schema version %d != %d", version, schemaVersion)
	}
	if _, err := i.db.Exec(ddl); err != nil {
		return errx.User("creating index schema: %v", err)
	}
	if _, err := i.db.Exec(fmt.Sprintf("PRAGMA user_version=%d", schemaVersion)); err != nil {
		return errx.User("setting index schema version: %v", err)
	}
	return nil
}
