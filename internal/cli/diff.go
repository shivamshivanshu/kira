package cli

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
)

func newDiffCmd(g *globalFlags) *cobra.Command {
	var incoming bool
	var since string
	cmd := &cobra.Command{
		Use:   "diff [ref]",
		Short: "Show your changes vs the merge-base with <ref>",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ref := ""
			if len(args) == 1 {
				ref = args[0]
			}
			if since != "" {
				if ref != "" {
					return errx.User("provide a ref positionally or --since, not both")
				}
				if incoming {
					return errx.User("--since cannot be combined with --incoming")
				}
			} else if ref == "" {
				return errx.User("diff requires a <ref> argument or --since")
			}
			s, _, err := openStore(g)
			if err != nil {
				return err
			}
			res, err := s.Diff(ref, since, incoming)
			if err != nil {
				return err
			}
			emitStderrNotes(cmd.ErrOrStderr(), res.StderrNotes)
			if g.json {
				return emitJSON(cmd.OutOrStdout(), res)
			}
			renderDiff(cmd.OutOrStdout(), res)
			return nil
		},
	}
	cmd.Flags().BoolVar(&incoming, "incoming", false, "show incoming changes on <ref> relative to the merge-base")
	cmd.Flags().StringVar(&since, "since", "", "show changes since a git ref or date (YYYY-MM-DD), relative to HEAD")
	return cmd
}

func renderDiffHeader(w io.Writer, status datamodel.DiffStatus, number, title string) {
	switch status {
	case datamodel.DiffCreated:
		_, _ = fmt.Fprintf(w, "created %s  %s\n", number, title)
	case datamodel.DiffDeleted:
		_, _ = fmt.Fprintf(w, "deleted %s  %s\n", number, title)
	default:
		_, _ = fmt.Fprintf(w, "%s  %s\n", number, title)
	}
}

func renderDiff(w io.Writer, res *datamodel.DiffResult) {
	if len(res.Items) == 0 {
		_, _ = fmt.Fprintln(w, "no backlog differences")
		return
	}
	for _, it := range res.Items {
		renderDiffHeader(w, it.Status, it.Number, it.Title)
		if it.Renumbered != nil {
			_, _ = fmt.Fprintf(w, "  renumbered %s -> %s\n", it.Renumbered.From, it.Renumbered.To)
		}
		for _, c := range it.Changes {
			if c.Field == datamodel.KeyBody {
				_, _ = fmt.Fprintf(w, "  body edited (%s)\n", c.To)
				continue
			}
			_, _ = fmt.Fprintf(w, "  %s: %s -> %s\n", c.Field, c.From, c.To)
		}
	}
}
