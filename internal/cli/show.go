package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func newShowCmd(g *globalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "show <id>",
		Short: "Show a ticket or epic",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, cfg, err := openStore(g)
			if err != nil {
				return err
			}
			res, err := s.Show(cfg, args[0])
			if err != nil {
				return err
			}
			emitStderrNotes(cmd.ErrOrStderr(), res.StderrNotes)
			if g.json {
				return emitJSON(cmd.OutOrStdout(), res)
			}
			renderShow(cmd.OutOrStdout(), res)
			return nil
		},
	}
}

func renderShow(w io.Writer, r *datamodel.ShowResult) {
	fmt.Fprintf(w, "%s  %s  [%s]\n", r.Number, r.Title, r.State)
	line := func(label, value string) {
		if value != "" {
			fmt.Fprintf(w, "%-10s %s\n", label+":", value)
		}
	}
	line("id", r.ID)
	line("type", r.Type)
	line("category", r.Category)
	line("priority", deref(r.Priority))
	line("owner", deref(r.Owner))
	line("reporter", deref(r.Reporter))
	if len(r.Labels) > 0 {
		line("labels", strings.Join(r.Labels, ", "))
	}
	line("epic", deref(r.Epic))
	if len(r.BlockedBy) > 0 {
		line("blocked_by", strings.Join(r.BlockedBy, ", "))
	}
	line("created", r.Created)
	line("updated", r.Updated)
	if strings.TrimSpace(r.Body) != "" {
		fmt.Fprintf(w, "\n%s", r.Body)
	}
}

func deref(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
