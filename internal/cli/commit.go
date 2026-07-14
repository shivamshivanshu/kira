package cli

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func newCommitCmd(g *globalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "commit",
		Short: "Commit all pending kira changes as a single commit",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			s, cfg, err := openStore(g)
			if err != nil {
				return err
			}
			res, err := s.CommitKira(cfg)
			if err != nil {
				return err
			}
			if g.json {
				return emitJSON(cmd.OutOrStdout(), res)
			}
			printCommit(cmd.OutOrStdout(), res)
			return nil
		},
	}
}

func printCommit(out io.Writer, res *datamodel.CommitResult) {
	sha := res.SHA
	if len(sha) > 7 {
		sha = sha[:7]
	}
	fmt.Fprintf(out, "committed %s: %s (%d files)\n", sha, res.Subject, res.Files)
}
