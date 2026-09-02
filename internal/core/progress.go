package core

import (
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

// ContainerProgress reports the done/total counts under every container.
func (s *Store) ContainerProgress(cfg *datamodel.Config) (map[string]datamodel.ContainerProgress, error) {
	ld, err := s.read(cfg, loadOpts{useIndex: true})
	if err != nil {
		return nil, err
	}
	return allContainerProgress(cfg, ld.items, entityschema.ContainerTypes(ld.schemas))
}

func allContainerProgress(cfg *datamodel.Config, items []*datamodel.Item, containers map[string]bool) (map[string]datamodel.ContainerProgress, error) {
	_, children := indexByParent(items)
	out := make(map[string]datamodel.ContainerProgress)
	visiting := make(map[string]bool)
	var count func(*datamodel.Item) (datamodel.ContainerProgress, error)
	count = func(it *datamodel.Item) (datamodel.ContainerProgress, error) {
		if p, ok := out[it.ID]; ok {
			return p, nil
		}
		if visiting[it.ID] {
			return datamodel.ContainerProgress{}, errx.Conflict("parent cycle detected at %s", it.Number)
		}
		visiting[it.ID] = true
		var p datamodel.ContainerProgress
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
