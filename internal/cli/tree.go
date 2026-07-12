package cli

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/shivamshivanshu/kira/internal/core"
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
			if g.json {
				return emitJSON(cmd.OutOrStdout(), res)
			}
			renderTree(cmd.OutOrStdout(), res)
			return nil
		},
	}
}

// renderTree prints the hierarchy as an indented outline, two spaces per depth.
func renderTree(w io.Writer, res *core.TreeResult) {
	if len(res.Nodes) == 0 {
		fmt.Fprintln(w, "no items")
		return
	}
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	for _, n := range res.Nodes {
		renderTreeNode(tw, n, 0)
	}
	tw.Flush()
}

func renderTreeNode(w io.Writer, n core.TreeNode, depth int) {
	indent := strings.Repeat("  ", depth)
	fmt.Fprintf(w, "%s%s\t%s\t%s\n", indent, n.Number, n.Type, n.Title)
	for _, c := range n.Children {
		renderTreeNode(w, c, depth+1)
	}
}

// renderTreeGroups prints the epic-grouped list (list --tree / query default):
// each epic as a header, its items indented beneath, orphans last.
func renderTreeGroups(w io.Writer, res *core.ListResult) {
	if res.Count == 0 {
		fmt.Fprintln(w, "no items")
		return
	}
	byID := make(map[string]core.ListItem, len(res.Items))
	for _, it := range res.Items {
		byID[it.ID] = it
	}
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
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

// epicLabel is a tree group's header text: its epic display number, else the
// epic ULID, else "" for the orphan bucket.
func epicLabel(grp core.TreeGroup) string {
	switch {
	case grp.EpicNumber != nil:
		return *grp.EpicNumber
	case grp.Epic != nil:
		return *grp.Epic
	default:
		return ""
	}
}
