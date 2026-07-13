package core

import "github.com/shivamshivanshu/kira/internal/datamodel"

func indexByEpic(items []*datamodel.Item) (map[string]*datamodel.Item, map[string][]*datamodel.Item) {
	byID := byULID(items)
	children := map[string][]*datamodel.Item{}
	for _, it := range items {
		if it.Epic != nil {
			if _, ok := byID[*it.Epic]; ok {
				children[*it.Epic] = append(children[*it.Epic], it)
			}
		}
	}
	return byID, children
}

func (s *Store) EpicProgress(cfg *datamodel.Config) (map[string]datamodel.EpicProgress, error) {
	ld, err := s.read(cfg, loadOpts{useIndex: true})
	if err != nil {
		return nil, err
	}
	items := ld.items
	_, children := indexByEpic(items)
	out := make(map[string]datamodel.EpicProgress)
	for _, it := range items {
		if it.Type == datamodel.TypeEpic {
			out[it.ID] = epicProgress(cfg, children, it.ID)
		}
	}
	return out, nil
}

func epicProgress(cfg *datamodel.Config, children map[string][]*datamodel.Item, epicID string) datamodel.EpicProgress {
	var p datamodel.EpicProgress
	walkEpic(children, epicID, func(c *datamodel.Item) bool { return c.Type == datamodel.TypeEpic }, func(c *datamodel.Item) {
		if c.Type == datamodel.TypeEpic {
			return
		}
		p.Total++
		if cat, ok := categoryOf(cfg, c.Type, c.State); ok && cat == datamodel.CategoryDone && !isDropped(c) {
			p.Done++
		}
	})
	return p
}

func isDropped(it *datamodel.Item) bool {
	return it.Resolution != nil && *it.Resolution == datamodel.ResolutionDropped
}
