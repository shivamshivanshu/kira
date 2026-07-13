package cli

import (
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/gitx"
)

func newMergeFileCmd(g *globalFlags) *cobra.Command {
	return &cobra.Command{
		Use:    "merge-file <base> <ours> <theirs>",
		Short:  "Git merge driver entry point for kira items",
		Hidden: true,
		Args:   cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			repo := gitx.Repo{Dir: filepath.Dir(args[1])}
			res, err := core.MergeFile(repo, args[0], args[1], args[2])
			if err != nil {
				return err
			}
			if g.json {
				return emitJSON(cmd.OutOrStdout(), res)
			}
			return nil
		},
	}
}
