package cli

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/gitx"
)

func newRestoreCmd(g *globalFlags) *cobra.Command {
	var opts core.RestoreOpts
	cmd := &cobra.Command{
		Use:   "restore <id> --at <ref>",
		Short: "Restore ticket prose or selected fields from Git history",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, cfg, err := openStore(g)
			if err != nil {
				return err
			}
			res, err := store.Restore(cfg, args[0], opts)
			if err != nil {
				return err
			}
			emitStderrNotes(cmd.ErrOrStderr(), res.Warnings)
			if g.json {
				return emitJSON(cmd.OutOrStdout(), res)
			}
			renderRestore(cmd.OutOrStdout(), res)
			return nil
		},
	}
	cmd.Flags().StringVar(&opts.At, "at", "", "source Git ref or date (YYYY-MM-DD)")
	cmd.Flags().StringArrayVar(&opts.Fields, "field", nil, "field to restore (repeatable, default body)")
	cmd.Flags().BoolVar(&opts.DryRun, "dry-run", false, "preview the changes")
	_ = cmd.MarkFlagRequired("at")
	return cmd
}

func renderRestore(w io.Writer, res *datamodel.RestoreResult) {
	if len(res.Changes) == 0 {
		_, _ = fmt.Fprintln(w, "no changes")
		return
	}
	verb := "restored"
	if res.DryRun {
		verb = "would restore"
	}
	_, _ = fmt.Fprintf(w, "%s %s from %s\n", verb, res.Number, gitx.ShortSHA(res.Source))
	for _, change := range res.Changes {
		if change.Field == datamodel.KeyBody {
			_, _ = fmt.Fprintf(w, "  body (%s)\n", change.To)
		} else {
			_, _ = fmt.Fprintf(w, "  %s: %s -> %s\n", change.Field, change.From, change.To)
		}
	}
}
