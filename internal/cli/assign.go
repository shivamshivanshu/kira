package cli

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
)

func newAssignCmd(g *globalFlags) *cobra.Command {
	var opts core.AssignOpts
	var owner string
	cmd := &cobra.Command{
		Use:   "assign <id>... [<user>]",
		Short: "Assign the owner (or reporter) of one or more tickets",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var ids []string
			user := owner
			positionalOwner := false
			switch {
			case cmd.Flags().Changed("owner"):
				ids = args
			case len(args) == 2:
				ids, user, positionalOwner = args[:1], args[1], true
			default:
				return errx.User("provide an owner: `assign <id> <user>` or `assign <id>... --owner <user>`")
			}
			s, cfg, err := openStore(g)
			if err != nil {
				return err
			}
			b, err := s.BeginBatch(cfg)
			if err != nil {
				return err
			}
			defer b.Close()
			if positionalOwner && b.RefExists(user) {
				return errx.User("%q resolves to an existing item, not an owner", user).WithHint("did you mean `--owner` to assign multiple ids?")
			}
			apply := func(id string) (*datamodel.MutationResult, error) { return b.Assign(id, user, opts) }
			line := func(res *datamodel.MutationResult) string { return assignLine(res, user) }
			warn := func(w io.Writer, res *datamodel.MutationResult) { emitMutationWarnings(w, res.Warnings) }
			out := cmd.OutOrStdout()
			return runSingleOrBulk(out, cmd.ErrOrStderr(), g.json, ids, apply, line, warn)
		},
	}
	f := cmd.Flags()
	f.StringVar(&owner, "owner", "", "owner to assign (required to assign multiple ids)")
	f.BoolVar(&opts.Reporter, "reporter", false, "assign the reporter field instead of owner")
	f.BoolVar(&opts.Force, "force", false, "accept field values outside the configured vocabulary")
	return cmd
}

func assignLine(res *datamodel.MutationResult, user string) string {
	if len(res.Changed) == 0 {
		return fmt.Sprintf("%s: no changes", res.Number)
	}
	return fmt.Sprintf("Assigned %s: %s", res.Number, user)
}
