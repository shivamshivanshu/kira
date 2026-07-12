package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/shivamshivanshu/kira/internal/core"
)

func newLinkCmd(g *globalFlags) *cobra.Command {
	var (
		epic      string
		blockedBy string
		opts      core.LinkOpts
	)
	cmd := &cobra.Command{
		Use:   "link <id>",
		Short: "Set or remove an epic parent or blocked-by dependency",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			epicSet := cmd.Flags().Changed("epic")
			blockedSet := cmd.Flags().Changed("blocked-by")
			if epicSet == blockedSet {
				return fmt.Errorf("give exactly one of --epic or --blocked-by")
			}
			if epicSet {
				opts.Target, opts.Ref = core.LinkEpic, epic
			} else {
				opts.Target, opts.Ref = core.LinkBlockedBy, blockedBy
			}
			s, cfg, err := openStore(g)
			if err != nil {
				return err
			}
			res, err := s.Link(cfg, args[0], opts)
			if err != nil {
				return err
			}
			if g.json {
				return emitJSON(cmd.OutOrStdout(), res)
			}
			if len(res.Changed) == 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "%s: no changes\n", res.Number)
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "Linked %s\n", res.Number)
			}
			return nil
		},
	}
	f := cmd.Flags()
	f.StringVar(&epic, "epic", "", "epic parent to set (or clear with --remove)")
	f.StringVar(&blockedBy, "blocked-by", "", "blocking item to add (or remove with --remove)")
	f.BoolVar(&opts.Remove, "remove", false, "remove the given edge instead of adding it")
	f.BoolVar(&opts.Force, "force", false, "bypass strict-vocabulary rejection")
	return cmd
}
