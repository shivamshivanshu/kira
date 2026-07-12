package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/shivamshivanshu/kira/internal/core"
)

func newQueryCmd(g *globalFlags) *cobra.Command {
	var tree, flat bool
	cmd := &cobra.Command{
		Use:   "query <expr>",
		Short: "Filter tickets and epics with a query expression",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if tree && flat {
				return fmt.Errorf("cannot use --tree and --flat together")
			}
			s, cfg, err := openStore(g)
			if err != nil {
				return err
			}
			// Tree is the default render; --flat opts out (docs/design/04-cli.md query).
			res, err := s.List(cfg, core.ListOpts{Query: args[0], Tree: !flat})
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
	f.BoolVar(&tree, "tree", false, "group results by epic (default)")
	f.BoolVar(&flat, "flat", false, "linear list instead of the epic tree")
	return cmd
}
