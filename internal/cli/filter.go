package cli

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func newFilterCmd(g *globalFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "filter",
		Short: "Named saved queries from config filters:",
	}
	cmd.AddCommand(newFilterListCmd(g))
	return cmd
}

func newFilterListCmd(g *globalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List the named saved queries",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, cfg, err := openStore(g)
			if err != nil {
				return err
			}
			res := core.Filters(cfg)
			if g.json {
				return emitJSON(cmd.OutOrStdout(), res)
			}
			renderFilterList(cmd.OutOrStdout(), res)
			return nil
		},
	}
}

func renderFilterList(w io.Writer, res *datamodel.FilterListResult) {
	if len(res.Filters) == 0 {
		fmt.Fprintln(w, "no filters configured")
		return
	}
	tw := newTabWriter(w)
	for _, f := range res.Filters {
		fmt.Fprintf(tw, "%s\t%s\n", f.Name, f.Query)
	}
	tw.Flush()
}
