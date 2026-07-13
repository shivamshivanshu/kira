package core

import "github.com/shivamshivanshu/kira/internal/datamodel"

func indexByEpic(items []*datamodel.Item) (map[string]*datamodel.Item, map[string][]*datamodel.Item) {
	byID := make(map[string]*datamodel.Item, len(items))
	for _, it := range items {
		byID[it.ID] = it
	}
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
	items, _, _, _, err := s.indexedLoad(cfg)
	if err != nil {
		return nil, err
	}
	_, children := indexByEpic(items)
	out := make(map[string]datamodel.EpicProgress)
	for _, it := range items {
		if it.Type != datamodel.TypeEpic {
			continue
		}
		var p datamodel.EpicProgress
		accumulateProgress(cfg, children, it.ID, map[string]bool{}, &p)
		out[it.ID] = p
	}
	return out, nil
}

func accumulateProgress(cfg *datamodel.Config, children map[string][]*datamodel.Item, epicID string, onPath map[string]bool, p *datamodel.EpicProgress) {
	if onPath[epicID] {
		return
	}
	onPath[epicID] = true
	defer delete(onPath, epicID)
	for _, c := range children[epicID] {
		if c.Type == datamodel.TypeEpic {
			accumulateProgress(cfg, children, c.ID, onPath, p)
			continue
		}
		p.Total++
		if cat, ok := categoryOf(cfg, c.Type, c.State); ok && cat == datamodel.CategoryDone && !isDropped(c) {
			p.Done++
		}
	}
}

func isDropped(it *datamodel.Item) bool {
	return it.Resolution != nil && *it.Resolution == datamodel.ResolutionDropped
}
