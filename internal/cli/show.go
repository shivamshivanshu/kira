package cli

import (
	"cmp"
	"fmt"
	"io"
	"maps"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/shivamshivanshu/kira/internal/codec"
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
			if format != "" {
				emitShowNotes(cmd, res)
				out, err := showfmt.Format(showfmt.Form(format), showfmt.Item{ID: res.ID, Number: res.Number, Title: res.Title}, cfg.Workon)
				if err != nil {
					return err
				}
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), out)
				return nil
			}
			return printShow(cmd, g, res)
		},
	}
	cmd.Flags().StringVar(&at, "at", "", "read state at a git ref or date (YYYY-MM-DD), anchored on HEAD")
	cmd.Flags().StringVar(&format, "format", "", "print a single reference form: "+strings.Join(showfmt.Names(), "|"))
	return cmd
}

func renderSkew(sk *datamodel.Skew) string {
	return fmt.Sprintf("%s at %s is %s; currently it is a different item (%s)", sk.Ref, sk.At, sk.AtID, sk.NowID)
}

// emitShowNotes prints a ShowResult's stderr-only side channel: cache
// staleness notes and any ref-skew warning.
func emitShowNotes(cmd *cobra.Command, res *datamodel.ShowResult) {
	emitStderrNotes(cmd.ErrOrStderr(), res.StderrNotes)
	if res.Skew != nil {
		_, _ = fmt.Fprintln(cmd.ErrOrStderr(), msgPrefix, renderSkew(res.Skew))
	}
}

// printShow emits a ShowResult as JSON or human-readable text, sharing the
// output block between `kira show` and `kira discover --action show`.
func printShow(cmd *cobra.Command, g *globalFlags, res *datamodel.ShowResult) error {
	emitShowNotes(cmd, res)
	if g.json {
		return emitJSON(cmd.OutOrStdout(), res)
	}
	renderShow(cmd.OutOrStdout(), res)
	return nil
}

func renderShow(w io.Writer, r *datamodel.ShowResult) {
	_, _ = fmt.Fprintf(w, "%s  %s  [%s]\n", r.Number, r.Title, r.State)
	// The label column is sized from the labels actually present: link types
	// and schema-declared field names are both arbitrary, so any fixed width
	// eventually misaligns.
	type row struct{ label, value string }
	var rows []row
	line := func(label, value string) {
		if value != "" {
			rows = append(rows, row{label + ":", value})
		}
	}
	line("id", r.ID)
	line("type", cmp.Or(r.TypeLabel, r.Type))
	line("subtype", ptr.Deref(r.Subtype))
	line("category", r.Category)
	line("priority", ptr.Deref(r.Priority))
	line("owner", ptr.Deref(r.Owner))
	line("reporter", ptr.Deref(r.Reporter))
	if len(r.Labels) > 0 {
		line("labels", strings.Join(r.Labels, ", "))
	}
	line("epic", ptr.DerefOr(r.EpicNumber, ptr.Deref(r.Epic)))
	if len(r.BlockedByNumbers) > 0 {
		line("blocked_by", strings.Join(r.BlockedByNumbers, ", "))
	}
	if len(r.BlocksNumbers) > 0 {
		line("blocks", strings.Join(r.BlocksNumbers, ", "))
	}
	for _, typ := range slices.Sorted(maps.Keys(r.LinksNumbers)) {
		line(typ, strings.Join(r.LinksNumbers[typ], ", "))
	}
	line("created", r.Created)
	line("updated", r.Updated)
	for _, name := range slices.Sorted(maps.Keys(r.Fields)) {
		line(name, typedFieldValueAsText(r.Fields[name]))
	}
	width := 0
	for _, rw := range rows {
		width = max(width, len(rw.label))
	}
	for _, rw := range rows {
		_, _ = fmt.Fprintf(w, "%-*s %s\n", width, rw.label, rw.value)
	}
	if desc := codec.Description(r.Body); strings.TrimSpace(desc) != "" {
		_, _ = fmt.Fprintf(w, "\n%s\n", desc)
	}
	if len(r.Comments) > 0 {
		_, _ = fmt.Fprintf(w, "\n%s\n", showfmt.CommentsHeader)
		for _, c := range r.Comments {
			_, _ = fmt.Fprintf(w, "%s\n%s\n", showfmt.CommentByline(c.Author, c.Ts), showfmt.CommentBody(c.Text))
		}
	}
}

func typedFieldValueAsText(value any) string {
	if list, isList := value.([]string); isList {
		return strings.Join(list, ", ")
	}
	return fmt.Sprint(value)
}
