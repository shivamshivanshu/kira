package cli

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/syncx"
)

func newSyncCmd(g *globalFlags) *cobra.Command {
	var opts core.SyncOpts
	var commit, stash bool
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Get up to date with the remote (pull --rebase, reconcile, reindex) and optionally publish",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if commit && stash {
				return errx.User("--commit and --stash are mutually exclusive")
			}
			switch {
			case commit:
				opts.Dirty = syncx.DirtyCommit
			case stash:
				opts.Dirty = syncx.DirtyStash
			}
			s, cfg, err := openStore(g)
			if err != nil {
				return err
			}
			report, err := s.Sync(cfg, opts, nil)
			if err != nil {
				if report != nil {
					printSyncReport(cmd.ErrOrStderr(), report)
				}
				return err
			}
			if g.json {
				return emitJSON(cmd.OutOrStdout(), report)
			}
			printSyncReport(cmd.OutOrStdout(), report)
			return nil
		},
	}
	f := cmd.Flags()
	f.BoolVar(&opts.Push, "push", false, "push to the remote after a clean sync")
	f.StringVar(&opts.Remote, "remote", "", "remote to sync with (default: the branch's upstream)")
	f.BoolVar(&commit, "commit", false, "commit dirty kira paths before pulling")
	f.BoolVar(&stash, "stash", false, "stash dirty changes before pulling and restore after")
	return cmd
}

func printSyncReport(out io.Writer, report *syncx.Report) {
	for _, step := range report.Steps {
		if step.Detail != "" {
			_, _ = fmt.Fprintf(out, "%-10s %s (%s)\n", step.Name+":", step.Status, step.Detail)
		} else {
			_, _ = fmt.Fprintf(out, "%-10s %s\n", step.Name+":", step.Status)
		}
	}
}
