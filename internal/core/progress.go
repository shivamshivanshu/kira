package core

import (
	"slices"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/entityschema"
	"github.com/shivamshivanshu/kira/internal/errx"
)

func indexByParent(items []*datamodel.Item) (map[string]*datamodel.Item, map[string][]*datamodel.Item) {
	byID := datamodel.IndexByID(items)
	children := map[string][]*datamodel.Item{}
	for _, it := range items {
		if it.Parent != nil {
			if _, ok := byID[*it.Parent]; ok {
				children[*it.Parent] = append(children[*it.Parent], it)
			}
		}
	}
	return byID, children
}

// EpicProgress reports the done/total ticket counts under every epic.
func (s *Store) EpicProgress(cfg *datamodel.Config) (map[string]datamodel.EpicProgress, error) {
	ld, err := s.read(cfg, loadOpts{useIndex: true})
	if err != nil {
		return nil, err
	}
	return allEpicProgress(cfg, ld.items, entityschema.ContainerTypes(ld.schemas))
}

func allEpicProgress(cfg *datamodel.Config, items []*datamodel.Item, containers map[string]bool) (map[string]datamodel.EpicProgress, error) {
	_, children := indexByParent(items)
	out := make(map[string]datamodel.EpicProgress)
	visiting := make(map[string]bool)
	var count func(*datamodel.Item) (datamodel.EpicProgress, error)
	count = func(it *datamodel.Item) (datamodel.EpicProgress, error) {
		if p, ok := out[it.ID]; ok {
			return p, nil
		}
		if visiting[it.ID] {
			return datamodel.EpicProgress{}, errx.Conflict("epic cycle detected at %s", it.Number)
		}
		visiting[it.ID] = true
		var p datamodel.EpicProgress
		for _, child := range children[it.ID] {
			if containers[child.Type] {
				sub, err := count(child)
				if err != nil {
					return p, err
				}
				p.Total += sub.Total
				p.Done += sub.Done
			} else {
				p.Total++
				if isDoneState(cfg, child.Type, child.State) {
					p.Done++
				}
			}
		}
		delete(visiting, it.ID)
		out[it.ID] = p
		return p, nil
	}
	for _, it := range items {
		if containers[it.Type] {
			_, err := count(it)
			if err != nil {
				return nil, err
			}
		}
	}
	return out, nil
}

func epicProgress(cfg *datamodel.Config, children map[string][]*datamodel.Item, containerID string, containers map[string]bool) (datamodel.EpicProgress, error) {
	var p datamodel.EpicProgress
	err := walkEpic(children, containerID, func(c *datamodel.Item) bool { return containers[c.Type] }, func(c *datamodel.Item) {
		if containers[c.Type] {
			return
		}
		p.Total++
		if cat, ok := cfg.CategoryOf(c.Type, c.State); ok && cat == datamodel.CategoryDone {
			p.Done++
		}
	})
	return p, err
}

func isDropped(cfg *datamodel.Config, it *datamodel.Item) bool {
	return it.Resolution != nil && slices.Contains(cfg.ResolutionsDropped, *it.Resolution)
}
