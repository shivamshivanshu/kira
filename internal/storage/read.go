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
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		names = append(names, e.Name())
	}
	sort.Strings(names)

	items := make([]*datamodel.Item, 0, len(names))
	for _, name := range names {
		path := filepath.Join(s.TicketsDir(), name)
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, errx.User("reading %s: %v", name, err)
		}
		it, err := codec.Parse(string(data))
		if err != nil {
			return nil, errx.User("parsing %s: %v", name, err)
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
