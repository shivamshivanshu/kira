package cli

import (
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/shivamshivanshu/kira/internal/core"
)

func newListCmd(g *globalFlags) *cobra.Command {
	var opts core.ListOpts
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List tickets and epics",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			s, cfg, err := openStore(g)
			if err != nil {
				return err
			}
			res, err := s.List(cfg, opts)
			if err != nil {
				return err
			}
			if g.json {
				return emitJSON(cmd.OutOrStdout(), res)
			}
			renderListResult(cmd.OutOrStdout(), res)
			return nil
		},
	}
	f := cmd.Flags()
	f.StringVar(&opts.Type, "type", "", "filter by type (ticket|epic)")
	f.StringVar(&opts.State, "state", "", "filter by state")
	f.StringVar(&opts.Category, "category", "", "filter by category (todo|doing|done)")
	f.StringVar(&opts.Owner, "owner", "", "filter by owner")
	f.StringVar(&opts.Label, "label", "", "filter by label")
	f.StringVar(&opts.Epic, "epic", "", "filter by parent epic")
	f.StringVar(&opts.Query, "query", "", "filter by a query expression (ANDed with the flags)")
	f.BoolVar(&opts.Tree, "tree", false, "group results by epic")
	return cmd
}

func renderListResult(w io.Writer, res *core.ListResult) {
	if res.Tree != nil {
		renderTreeGroups(w, res)
		return
	}
	renderList(w, res)
}

func renderList(w io.Writer, res *core.ListResult) {
	if res.Count == 0 {
		fmt.Fprintln(w, "no items")
		return
	}
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	for _, it := range res.Items {
		fmt.Fprintln(tw, formatItemRow(it))
	}
	tw.Flush()
}

// formatItemRow renders one item as tab-separated columns (number, state, type,
// title), shared by the flat list and the epic-grouped tree renders.
func formatItemRow(it core.ListItem) string {
	return fmt.Sprintf("%s\t%s\t%s\t%s", it.Number, it.State, it.Type, it.Title)
}
