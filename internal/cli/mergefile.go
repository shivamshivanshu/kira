package cli

import (
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/gitx"
)

func newMergeFileCmd(g *globalFlags) *cobra.Command {
	var sides core.MergeSides
	cmd := &cobra.Command{
		Use:    "merge-file <base> <ours> <theirs>",
		Short:  "Git merge driver entry point for kira items",
		Hidden: true,
		Args:   cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			if sides.Ours == "" || sides.Theirs == "" {
				return errx.Conflict("the Kira merge driver is missing the Git merge sides").
					WithHint("run kira hooks install to update the driver, then kira resolve")
			}
			repo := gitx.Repo{Dir: filepath.Dir(args[1])}
			res, err := core.MergeFile(repo, args[0], args[1], args[2], sides)
			if err != nil {
				return err
			}
			if g.json {
				return emitJSON(cmd.OutOrStdout(), res)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&sides.Ours, "ours", "", "Git's local-side conflict label")
	cmd.Flags().StringVar(&sides.Theirs, "incoming", "", "Git's incoming-side conflict label")
	return cmd
}
