package storage

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
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
	return strings.TrimSuffix(name, itemExt), true
}

func ULIDFromPath(p string) string {
	ulid, _ := ULIDFromFilename(filepath.Base(p))
	return ulid
}

func (s *FS) ItemFilenames() ([]string, error) {
	return s.itemsDirEntries(isItemFilename)
}

// StrayFilenames lists tickets-dir entries that are not valid item files —
// e.g. a stray README.md — so callers like doctor can still surface them
// even though ItemFilenames/LoadAll now silently skip them.
func (s *FS) StrayFilenames() ([]string, error) {
	return s.itemsDirEntries(func(name string) bool { return !isItemFilename(name) })
}

func (s *FS) itemsDirEntries(keep func(name string) bool) ([]string, error) {
	entries, err := os.ReadDir(s.ItemsDir())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, errx.User("reading tickets: %v", err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() && keep(e.Name()) {
			names = append(names, e.Name())
		}
	}
	return names, nil
}

type RawItem struct {
	Name    string
	Content string
}

func (s *FS) RawItems() ([]RawItem, error) {
	names, err := s.ItemFilenames()
	if err != nil {
		return nil, err
	}
	items := make([]RawItem, len(names))
	for i, name := range names {
		data, err := os.ReadFile(filepath.Join(s.ItemsDir(), name))
		if err != nil {
			return nil, errx.User("reading %s: %v", name, err)
		}
		items[i] = RawItem{Name: name, Content: string(data)}
	}
	return items, nil
}

// Skipped describes the files LoadAll could not parse. Numbers holds the
// display numbers recovered from them textually: an unparseable file still owns
// its number, and an allocator that cannot see it will hand out a duplicate.
type Skipped struct {
	Warnings []string
	Numbers  []string
}

func (s *FS) LoadAll() ([]*datamodel.Item, Skipped, error) {
	names, err := s.ItemFilenames()
	if err != nil {
		return nil, Skipped{}, err
	}
	items := make([]*datamodel.Item, 0, len(names))
	var skipped Skipped
	for _, name := range names {
		path := filepath.Join(s.ItemsDir(), name)
		it, err := ReadItem(path)
		if err != nil {
			skipped.Warnings = append(skipped.Warnings, SkipNote(name, err))
			skipped.Numbers = append(skipped.Numbers, numbersInFile(path)...)
			continue
		}
		items = append(items, it)
	}
	return items, skipped, nil
}

var numberLine = regexp.MustCompile(`(?m)^number:[ \t]*["']?([A-Za-z0-9]+-[0-9]+)["']?[ \t]*$`)

// numbersInFile recovers every number a file declares without parsing it. A
// conflict-marked file legitimately carries two, and both are taken.
func numbersInFile(path string) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	matches := numberLine.FindAllSubmatch(data, -1)
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		out = append(out, string(m[1]))
	}
	return out
}

func SkipNote(name string, err error) string {
	return fmt.Sprintf("skipped %s/%s: %v; run kira doctor", TicketsPrefix, name, err)
}

func Snapshot(key string, items []*datamodel.Item, reserved ...string) id.Snapshot {
	snap := id.Snapshot{Key: key, Items: make([]id.Item, len(items)), Reserved: reserved}
	for i, it := range items {
		snap.Items[i] = id.Item{ULID: it.ID, Number: it.Number, Aliases: it.Aliases}
	}
	return snap
}

func SnapshotAndResolver(key string, items []*datamodel.Item, reserved ...string) (id.Snapshot, *id.Resolver) {
	snap := Snapshot(key, items, reserved...)
	return snap, id.NewResolver(snap)
}
