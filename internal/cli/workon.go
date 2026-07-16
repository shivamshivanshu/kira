package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func newWorkonCmd(g *globalFlags) *cobra.Command {
	var opts core.WorkonOpts
	cmd := &cobra.Command{
		Use:   "workon <id>",
		Short: "Switch to (or create) a per-ticket branch and start work",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, cfg, err := openStore(g)
			if err != nil {
				return err
			}
			if !cmd.Flags().Changed("worktree") {
				opts.Worktree = cfg.Workon.Worktree
			}
			res, err := s.Workon(cfg, args[0], opts)
			if err != nil {
				return err
			}
			emitMutationWarnings(cmd.ErrOrStderr(), res.Warnings)
			if g.json {
				return emitJSON(cmd.OutOrStdout(), res)
			}
			printWorkon(cmd, res)
			return nil
		},
	}
	f := cmd.Flags()
	f.BoolVar(&opts.NoMove, "no-move", false, "do not transition the ticket into a doing state")
	f.BoolVar(&opts.Worktree, "worktree", false, "create a dedicated git worktree instead of switching in place")
	return cmd
}

func printWorkon(cmd *cobra.Command, res *datamodel.WorkonResult) {
	out := cmd.OutOrStdout()
	verb := "Switched to"
	if res.BranchCreated {
		verb = "Created"
	}
	if res.Worktree != "" {
		fmt.Fprintf(out, "%s worktree %s on branch %s for %s\n", verb, res.Worktree, res.Branch, res.Number)
	} else {
		fmt.Fprintf(out, "%s branch %s for %s\n", verb, res.Branch, res.Number)
	}
	if res.Moved {
		fmt.Fprintf(out, "Moved %s: %s -> %s\n", res.Number, res.From, res.To)
	}
}
