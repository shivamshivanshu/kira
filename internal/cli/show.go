package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/ptr"
	"github.com/shivamshivanshu/kira/internal/showfmt"
)

func newShowCmd(g *globalFlags) *cobra.Command {
	var at string
	var format string
	cmd := &cobra.Command{
		Use:   "show <id>",
		Short: "Show a ticket or epic",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if format != "" && g.json {
				return errx.User("--format cannot be combined with --json")
			}
			s, cfg, err := openStore(g)
			if err != nil {
				return err
			}
			res, err := s.Show(cfg, args[0], at)
			if err != nil {
				return err
			}
			emitStderrNotes(cmd.ErrOrStderr(), res.StderrNotes)
			if res.Skew != nil {
				_, _ = fmt.Fprintln(cmd.ErrOrStderr(), msgPrefix, renderSkew(res.Skew))
			}
			if format != "" {
				out, err := showfmt.Format(showfmt.Form(format), showfmt.Item{ID: res.ID, Number: res.Number, Title: res.Title})
				if err != nil {
					return err
				}
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), out)
				return nil
			}
			if g.json {
				return emitJSON(cmd.OutOrStdout(), res)
			}
			renderShow(cmd.OutOrStdout(), res)
			return nil
		},
	}
	cmd.Flags().StringVar(&at, "at", "", "read state at a git ref or date (YYYY-MM-DD), anchored on HEAD")
	cmd.Flags().StringVar(&format, "format", "", "print a single reference form: "+strings.Join(showfmt.Names(), "|"))
	return cmd
}

func renderSkew(sk *datamodel.Skew) string {
	return fmt.Sprintf("%s at %s is %s; currently it is a different item (%s)", sk.Ref, sk.At, sk.AtID, sk.NowID)
}

func renderShow(w io.Writer, r *datamodel.ShowResult) {
	_, _ = fmt.Fprintf(w, "%s  %s  [%s]\n", r.Number, r.Title, r.State)
	line := func(label, value string) {
		if value != "" {
			_, _ = fmt.Fprintf(w, "%-10s %s\n", label+":", value)
		}
	}
	line("id", r.ID)
	line("type", r.Type)
	line("subtype", ptr.Deref(r.Subtype))
	line("category", r.Category)
	line("priority", ptr.Deref(r.Priority))
	line("owner", ptr.Deref(r.Owner))
	line("reporter", ptr.Deref(r.Reporter))
	if len(r.Labels) > 0 {
		line("labels", strings.Join(r.Labels, ", "))
	}
	line("epic", ptr.Deref(r.Epic))
	if len(r.BlockedBy) > 0 {
		line("blocked_by", strings.Join(r.BlockedBy, ", "))
	}
	line("created", r.Created)
	line("updated", r.Updated)
	if strings.TrimSpace(r.Body) != "" {
		_, _ = fmt.Fprintf(w, "\n%s", r.Body)
	}
}
