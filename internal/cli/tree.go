package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func newTreeCmd(g *globalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "tree [<id>]",
		Short: "Render the epic hierarchy",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, cfg, err := openStore(g)
			if err != nil {
				return err
			}
			ref := ""
			if len(args) == 1 {
				ref = args[0]
			}
			res, err := s.Tree(cfg, ref)
			if err != nil {
				return err
			}
			emitStderrNotes(cmd.ErrOrStderr(), res.StderrNotes)
			if g.json {
				return emitJSON(cmd.OutOrStdout(), res)
			}
			renderTree(cmd.OutOrStdout(), res)
			return nil
		},
	}
}

func renderTree(w io.Writer, res *datamodel.TreeResult) {
	if len(res.Nodes) == 0 {
		fmt.Fprintln(w, "no items")
		return
	}
	tw := newTabWriter(w)
	for _, n := range res.Nodes {
		renderTreeNode(tw, n, 0)
	}
	tw.Flush()
}

func renderTreeNode(w io.Writer, n datamodel.TreeNode, depth int) {
	indent := strings.Repeat("  ", depth)
	fmt.Fprintf(w, "%s%s\t%s\t%s\n", indent, n.Number, n.Type, n.Title)
	for _, c := range n.Children {
		renderTreeNode(w, c, depth+1)
	}
}

func renderTreeGroups(w io.Writer, res *datamodel.ListResult) {
	if res.Count == 0 {
		fmt.Fprintln(w, "no items")
		return
	}
	byID := make(map[string]datamodel.ListItem, len(res.Items))
	for _, it := range res.Items {
		byID[it.ID] = it
	}
	tw := newTabWriter(w)
	for _, grp := range res.Tree {
		switch label := epicLabel(grp); label {
		case "":
			fmt.Fprintln(tw, "(orphans)")
		default:
			fmt.Fprintf(tw, "%s\t\t\t(epic)\n", label)
		}
		for _, ulid := range grp.Items {
			fmt.Fprintln(tw, "  "+formatItemRow(byID[ulid]))
		}
	}
	tw.Flush()
}

func epicLabel(grp datamodel.TreeGroup) string {
	switch {
	case grp.EpicNumber != nil:
		return *grp.EpicNumber
	case grp.Epic != nil:
		return *grp.Epic
	default:
		return ""
	}
}
