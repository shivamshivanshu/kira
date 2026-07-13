package storage

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/shivamshivanshu/kira/internal/codec"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/id"
)

func ReadItem(path string) (*datamodel.Item, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return codec.Parse(string(data))
}

func ULIDFromFilename(name string) (string, bool) {
	if !isItemFilename(name) {
		return "", false
	}
	return strings.TrimSuffix(name, ".md"), true
}

func ULIDFromPath(p string) string {
	ulid, _ := ULIDFromFilename(filepath.Base(p))
	return ulid
}

func (s *Store) LoadAll() ([]*datamodel.Item, []string, error) {
	entries, err := os.ReadDir(s.ItemsDir())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil, nil
		}
		return nil, nil, errx.User("reading tickets: %v", err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if _, ok := ULIDFromFilename(e.Name()); ok {
			names = append(names, e.Name())
		}
	}
	slices.Sort(names)

	items := make([]*datamodel.Item, 0, len(names))
	var warnings []string
	for _, name := range names {
		it, err := ReadItem(filepath.Join(s.ItemsDir(), name))
		if err != nil {
			warnings = append(warnings, SkipNote(name, err))
			continue
		}
		items = append(items, it)
	}
	return items, warnings, nil
}

func SkipNote(name string, err error) string {
	return fmt.Sprintf("skipped .kira/tickets/%s: %v; run kira doctor", name, err)
}

func Snapshot(key string, items []*datamodel.Item) id.Snapshot {
	snap := id.Snapshot{Key: key, Items: make([]id.Item, len(items))}
	for i, it := range items {
		snap.Items[i] = id.Item{ULID: it.ID, Number: it.Number, Aliases: it.Aliases}
	}
	return snap
}
