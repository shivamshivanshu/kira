package tui

import (
	"strings"

	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func loadFilteredTree(store *core.Store, cfg *datamodel.Config, expr string) (treeData, error) {
	tr, err := store.Tree(cfg, "", "")
	if err != nil {
		return treeData{}, err
	}
	nodes := tr.Nodes
	var rows []datamodel.ListItem
	if strings.TrimSpace(expr) == "" {
		lr, err := store.List(cfg, core.ListOpts{})
		if err != nil {
			return treeData{}, err
		}
		rows = lr.Items
	} else {
		var matched map[string]bool
		rows, matched, err = store.ListWithMatches(cfg, expr)
		if err != nil {
			return treeData{}, err
		}
		nodes = pruneNodes(tr.Nodes, matched)
	}
	pr, err := store.EpicProgress(cfg)
	if err != nil {
		return treeData{}, err
	}
	fields := make(map[string]datamodel.ListItem, len(rows))
	for _, it := range rows {
		fields[it.ID] = it
	}
	return treeData{nodes: nodes, fields: fields, progress: pr}, nil
}

func pruneNodes(nodes []datamodel.TreeNode, keep map[string]bool) []datamodel.TreeNode {
	out := make([]datamodel.TreeNode, 0, len(nodes))
	for _, n := range nodes {
		kids := pruneNodes(n.Children, keep)
		if keep[n.ID] || len(kids) > 0 {
			n.Children = kids
			out = append(out, n)
		}
	}
	return out
}
