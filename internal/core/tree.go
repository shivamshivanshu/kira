package core

import (
	"github.com/shivamshivanshu/kira/internal/config"
	"github.com/shivamshivanshu/kira/internal/id"
	"github.com/shivamshivanshu/kira/internal/item"
)

// Tree renders the epic hierarchy (docs/design/04-cli.md tree). With ref empty
// it returns the whole forest: every root item (one with no epic, or a dangling
// epic pointer) with its descendant subtree. With ref set it returns that
// item's subtree alone. Children are ordered by display number. If an epic
// cycle survives to read time (the doctor-flagged case went unrepaired),
// traversal stops and reports it (exit 2) rather than looping forever.
func (s *Store) Tree(cfg *config.Config, ref string) (*TreeResult, error) {
	items, _, resolver, err := s.load(cfg)
	if err != nil {
		return nil, err
	}

	byID := make(map[string]*item.Item, len(items))
	for _, it := range items {
		byID[it.ID] = it
	}
	// One pass indexes children by parent epic and collects the forest roots:
	// an item with no epic, or a dangling pointer, is a root.
	children := map[string][]*item.Item{}
	roots := make([]*item.Item, 0)
	for _, it := range items {
		p := ""
		if it.Epic != nil {
			if _, ok := byID[*it.Epic]; ok {
				p = *it.Epic
			}
		}
		if p != "" {
			children[p] = append(children[p], it)
		} else {
			roots = append(roots, it)
		}
	}
	for _, kids := range children {
		sortItems(kids)
	}

	build := newTreeBuilder(children)

	if ref != "" {
		ulid, err := resolver.Resolve(ref)
		if err != nil {
			return nil, userErr("%v", err)
		}
		it, ok := byID[ulid]
		if !ok {
			return nil, userErr("resolved %s to %s, which has no file", ref, ulid)
		}
		node, err := build.node(it)
		if err != nil {
			return nil, err
		}
		return &TreeResult{Root: &ulid, Nodes: []TreeNode{node}}, nil
	}

	sortItems(roots)
	nodes := make([]TreeNode, 0, len(roots))
	for _, it := range roots {
		node, err := build.node(it)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, node)
	}
	return &TreeResult{Root: nil, Nodes: nodes}, nil
}

// treeBuilder carries the child index and the on-path visited set for cycle
// detection during a depth-first build.
type treeBuilder struct {
	children map[string][]*item.Item
	onPath   map[string]bool
}

func newTreeBuilder(children map[string][]*item.Item) *treeBuilder {
	return &treeBuilder{children: children, onPath: map[string]bool{}}
}

func (b *treeBuilder) node(it *item.Item) (TreeNode, error) {
	if b.onPath[it.ID] {
		return TreeNode{}, conflictErr("epic cycle detected at %s", it.Number)
	}
	b.onPath[it.ID] = true
	defer delete(b.onPath, it.ID)

	kids := b.children[it.ID]
	childNodes := make([]TreeNode, 0, len(kids))
	for _, c := range kids {
		cn, err := b.node(c)
		if err != nil {
			return TreeNode{}, err
		}
		childNodes = append(childNodes, cn)
	}
	return TreeNode{
		ID:       it.ID,
		Number:   it.Number,
		Type:     it.Type,
		Title:    it.Title,
		Children: childNodes,
	}, nil
}

// sortItems orders items by the shared display-number key — the same
// deterministic order as list rows.
func sortItems(items []*item.Item) {
	sortByKey(items, func(it *item.Item) id.SortKey { return id.NewSortKey(it.Number, it.ID) })
}
