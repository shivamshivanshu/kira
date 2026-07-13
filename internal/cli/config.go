package cli

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func newConfigCmd(g *globalFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Inspect and edit .kira/config.yaml",
		Args:  cobra.NoArgs,
	}
	cmd.AddCommand(newConfigSetCmd(g), newConfigFiltersCmd(g))
	return cmd
}

func newConfigFiltersCmd(g *globalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "filters",
		Short: "List the named saved queries from config filters:",
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

func newConfigSetCmd(g *globalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a scalar config key, preserving comments and formatting",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, cfg, err := openStore(g)
			if err != nil {
				return err
			}
			res, err := s.ConfigSet(cfg, args[0], args[1])
			if err != nil {
				return err
			}
			if g.json {
				return emitJSON(cmd.OutOrStdout(), res)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Set %s = %s\n", res.Key, res.Value)
			return nil
		},
	}
}
