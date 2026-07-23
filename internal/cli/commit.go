package cli

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/gitx"
)

func newCommitCmd(g *globalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "commit",
		Short: "Commit all pending kira changes as a single commit",
		Args:  cobra.NoArgs,
		RunE: storeActionRunE(g,
			func(s *core.Store, cfg *datamodel.Config, _ []string) (*datamodel.CommitResult, error) {
				return s.CommitKira(cfg)
			},
			printCommit),
	}
}

func printCommit(out io.Writer, res *datamodel.CommitResult) {
	_, _ = fmt.Fprintf(out, "committed %s: %s (%d files)\n", gitx.ShortSHA(res.SHA), res.Subject, res.Files)
}
