package tui

import (
	"strings"

	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func loadFilteredTree(store *core.Store, cfg *datamodel.Config, expr string) (treeData, error) {
	if strings.TrimSpace(expr) == "" {
		return loadTreeData(store, cfg)
	}
	tr, err := store.Tree(cfg, "", "")
	if err != nil {
		return treeData{}, err
	}
	rows, keep, err := store.ListWithMatches(cfg, expr)
	if err != nil {
		return treeData{}, err
	}
	pr, err := store.EpicProgress(cfg)
	if err != nil {
		return treeData{}, err
	}
	fields := make(map[string]datamodel.ListItem, len(rows))
	for _, it := range rows {
		fields[it.ID] = it
	}
	return treeData{nodes: pruneNodes(tr.Nodes, keep), fields: fields, progress: pr}, nil
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
