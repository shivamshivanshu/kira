package index

import (
	"fmt"

	"github.com/shivamshivanshu/kira/internal/errx"
)

const schemaVersion = 6

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
  updated  TEXT NOT NULL,
  activity TEXT NOT NULL
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
  kind    TEXT NOT NULL,
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
		return errx.Env("reading index schema version: %v", err)
	}
	if version == schemaVersion {
		return nil
	}
	if version != 0 {
		if err := i.dropAllTables(); err != nil {
			return err
		}
	}
	if _, err := i.db.Exec(ddl); err != nil {
		return errx.Env("creating index schema: %v", err)
	}
	if _, err := i.db.Exec(fmt.Sprintf("PRAGMA user_version=%d", schemaVersion)); err != nil {
		return errx.Env("setting index schema version: %v", err)
	}
	return nil
}

func (i *Index) dropAllTables() error {
	rows, err := i.db.Query("SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'")
	if err != nil {
		return errx.Env("listing index tables: %v", err)
	}
	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			_ = rows.Close()
			return errx.Env("scanning index tables: %v", err)
		}
		names = append(names, name)
	}
	_ = rows.Close()
	if err := rows.Err(); err != nil {
		return errx.Env("listing index tables: %v", err)
	}
	for _, name := range names {
		if _, err := i.db.Exec("DROP TABLE IF EXISTS " + name); err != nil {
			return errx.Env("dropping stale index table %s: %v", name, err)
		}
	}
	return nil
}
