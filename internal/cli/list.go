package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/ptr"
)

func newListCmd(g *globalFlags) *cobra.Command {
	var opts core.ListOpts
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List tickets and epics",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			s, cfg, err := openStore(g)
			if err != nil {
				return err
			}
			res, err := s.List(cfg, opts)
			if err != nil {
				return err
			}
			emitStderrNotes(cmd.ErrOrStderr(), res.StderrNotes)
			if g.json {
				return emitJSON(cmd.OutOrStdout(), res)
			}
			renderListResult(cmd.OutOrStdout(), res, cfg.UI.List.Columns)
			return nil
		},
	}
	f := cmd.Flags()
	f.StringVar(&opts.Type, "type", "", "filter by type (ticket|epic)")
	f.StringVar(&opts.Subtype, "subtype", "", "filter by subtype")
	f.StringVar(&opts.State, "state", "", "filter by state")
	f.StringVar(&opts.Category, "category", "", "filter by category (todo|doing|done)")
	f.StringVar(&opts.Owner, "owner", "", "filter by owner ('@me' resolves to the git user)")
	f.StringVar(&opts.Label, "label", "", "filter by label")
	f.StringVar(&opts.Epic, "epic", "", "filter by parent epic")
	f.StringVar(&opts.Priority, "priority", "", "filter by priority")
	f.StringVar(&opts.Sprint, "sprint", "", "filter by sprint key ('active' resolves the active sprint)")
	f.StringVar(&opts.Filter, "filter", "", "apply a named saved query from config filters")
	f.StringVar(&opts.Query, "query", "", "filter by a query expression (ANDed with the flags)")
	f.BoolVar(&opts.Tree, "tree", false, "group results by epic")
	f.StringVar(&opts.At, "at", "", "read state at a git ref or date (YYYY-MM-DD), anchored on HEAD")
	return cmd
}

func emitStderrNotes(w io.Writer, notes []datamodel.Warning) {
	for _, n := range notes {
		_, _ = fmt.Fprintln(w, msgPrefix, renderWarning(n))
	}
}

func renderListResult(w io.Writer, res *datamodel.ListResult, columns []string) {
	if res.Tree != nil {
		renderTreeGroups(w, res)
		return
	}
	renderList(w, res, columns)
}

func renderList(w io.Writer, res *datamodel.ListResult, columns []string) {
	if res.Count == 0 {
		_, _ = fmt.Fprintln(w, msgNoItems)
		return
	}
	cols := resolveColumns(columns)
	tw := newTabWriter(w)
	_, _ = fmt.Fprintln(tw, columnHeader(cols))
	for _, it := range res.Items {
		_, _ = fmt.Fprintln(tw, formatItemRow(cols, it))
	}
	_ = tw.Flush()
}

func resolveColumns(columns []string) []string {
	out := make([]string, 0, len(columns))
	for _, c := range columns {
		if _, ok := listCells[c]; ok {
			out = append(out, c)
		}
	}
	if len(out) == 0 {
		return datamodel.DefaultListColumns
	}
	return out
}

func columnHeader(cols []string) string {
	heads := make([]string, len(cols))
	for i, c := range cols {
		heads[i] = strings.ToUpper(c)
	}
	return strings.Join(heads, "\t")
}

func formatItemRow(cols []string, it datamodel.ListItem) string {
	cells := make([]string, len(cols))
	for i, c := range cols {
		cells[i] = listCells[c](it)
	}
	return strings.Join(cells, "\t")
}

var listCells = map[string]func(datamodel.ListItem) string{
	"number":      func(it datamodel.ListItem) string { return it.Number },
	"title":       func(it datamodel.ListItem) string { return it.Title },
	"type":        func(it datamodel.ListItem) string { return it.Type },
	"state":       func(it datamodel.ListItem) string { return it.State },
	"category":    func(it datamodel.ListItem) string { return it.Category },
	"priority":    func(it datamodel.ListItem) string { return ptr.DerefOr(it.Priority, "-") },
	"owner":       func(it datamodel.ListItem) string { return ptr.DerefOr(it.Owner, "-") },
	"labels":      func(it datamodel.ListItem) string { return strings.Join(it.Labels, ",") },
	"epic":        func(it datamodel.ListItem) string { return ptr.DerefOr(it.Epic, "-") },
	"epic_number": func(it datamodel.ListItem) string { return ptr.DerefOr(it.EpicNumber, "-") },
	"resolution":  func(it datamodel.ListItem) string { return ptr.DerefOr(it.Resolution, "-") },
	"due":         func(it datamodel.ListItem) string { return ptr.DerefOr(it.Due, "-") },
	"id":          func(it datamodel.ListItem) string { return it.ID },
}
