package core

import (
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/id"
)

func (s *Store) Tree(cfg *datamodel.Config, ref string) (*datamodel.TreeResult, error) {
	items, _, resolver, idxNotes, err := s.indexedLoad(cfg)
	if err != nil {
		return nil, err
	}

	byID, children := indexByEpic(items)
	roots := make([]*datamodel.Item, 0)
	for _, it := range items {
		if it.Epic == nil || byID[*it.Epic] == nil {
			roots = append(roots, it)
		}
	}
	for _, kids := range children {
		sortItems(kids)
	}

	build := newTreeBuilder(children)

	if ref != "" {
		ulid, err := resolveID(resolver, ref)
		if err != nil {
			return nil, err
		}
		it, ok := byID[ulid]
		if !ok {
			return nil, errx.User("resolved %s to %s, which has no file", ref, ulid)
		}
		node, err := build.node(it)
		if err != nil {
			return nil, err
		}
		return &datamodel.TreeResult{Root: &ulid, Nodes: []datamodel.TreeNode{node}, StderrNotes: idxNotes}, nil
	}

	sortItems(roots)
	nodes := make([]datamodel.TreeNode, 0, len(roots))
	for _, it := range roots {
		node, err := build.node(it)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, node)
	}
	return &datamodel.TreeResult{Root: nil, Nodes: nodes, StderrNotes: idxNotes}, nil
}

func epicChildren(items []*datamodel.Item) map[string][]*datamodel.Item {
	_, children := indexByEpic(items)
	return children
}

type treeBuilder struct {
	children map[string][]*datamodel.Item
	onPath   map[string]bool
}

func newTreeBuilder(children map[string][]*datamodel.Item) *treeBuilder {
	return &treeBuilder{children: children, onPath: map[string]bool{}}
}

func (b *treeBuilder) node(it *datamodel.Item) (datamodel.TreeNode, error) {
	if b.onPath[it.ID] {
		return datamodel.TreeNode{}, errx.Conflict("epic cycle detected at %s", it.Number)
	}
	b.onPath[it.ID] = true
	defer delete(b.onPath, it.ID)

	kids := b.children[it.ID]
	childNodes := make([]datamodel.TreeNode, 0, len(kids))
	for _, c := range kids {
		cn, err := b.node(c)
		if err != nil {
			return datamodel.TreeNode{}, err
		}
		childNodes = append(childNodes, cn)
	}
	return datamodel.TreeNode{
		ID:       it.ID,
		Number:   it.Number,
		Type:     it.Type,
		Title:    it.Title,
		Children: childNodes,
	}, nil
}

func sortItems(items []*datamodel.Item) {
	sortByKey(items, func(it *datamodel.Item) id.SortKey { return id.NewSortKey(it.Number, it.ID) })
}
