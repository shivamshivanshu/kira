package storage

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
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
	if !strings.HasSuffix(name, ".md") || strings.HasPrefix(name, ".") {
		return "", false
	}
	return strings.TrimSuffix(name, ".md"), true
}

func (s *Store) LoadAll() ([]*datamodel.Item, error) {
	entries, err := os.ReadDir(s.TicketsDir())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, errx.User("reading tickets: %v", err)
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
	sort.Strings(names)

	items := make([]*datamodel.Item, 0, len(names))
	for _, name := range names {
		it, err := ReadItem(filepath.Join(s.TicketsDir(), name))
		if err != nil {
			return nil, errx.User("reading %s: %v", name, err)
		}
		items = append(items, it)
	}
	return items, nil
}

func Snapshot(key string, items []*datamodel.Item) id.Snapshot {
	snap := id.Snapshot{Key: key, Items: make([]id.Item, len(items))}
	for i, it := range items {
		snap.Items[i] = id.Item{ULID: it.ID, Number: it.Number, Aliases: it.Aliases}
	}
	return snap
}
