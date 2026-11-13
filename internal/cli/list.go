package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/ptr"
	"github.com/shivamshivanshu/kira/internal/warningfmt"
)

func newListCmd(g *globalFlags) *cobra.Command {
	var opts core.ListOpts
	var fields []string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List tickets and epics",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			filters, err := parseFieldFilters(fields)
			if err != nil {
				return err
			}
			opts.Fields = filters
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
			columns, unknown := knownColumns(cfg.UI.List.Columns)
			warnUnknownColumns(cmd.ErrOrStderr(), unknown)
			renderListResult(cmd.OutOrStdout(), res, columnsOrDefault(columns))
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
	f.StringVar(&opts.Board, "board", "", "filter to one board")
	f.StringVar(&opts.Parent, "parent", "", "filter by parent")
	f.StringVar(&opts.Parent, "epic", "", "deprecated alias for --parent")
	_ = f.MarkHidden("epic")
	f.StringVar(&opts.Priority, "priority", "", "filter by priority")
	f.StringVar(&opts.Sprint, "sprint", "", "filter by sprint key ('active' resolves the active sprint)")
	f.StringVar(&opts.Filter, "filter", "", "apply a named saved query from config filters")
	f.StringVar(&opts.Query, "query", "", "filter by a query expression (ANDed with the flags)")
	f.StringArrayVar(&fields, "field", nil, "key=value equality filter on any item field (repeatable, ANDed)")
	f.BoolVar(&opts.Tree, "tree", false, "group results by parent")
	f.StringVar(&opts.At, "at", "", "read state at a git ref or date (YYYY-MM-DD), anchored on HEAD")
	return cmd
}

func parseFieldFilters(fields []string) ([]core.FieldFilter, error) {
	out := make([]core.FieldFilter, 0, len(fields))
	for _, f := range fields {
		k, v, ok := strings.Cut(f, "=")
		if !ok || k == "" {
			return nil, errx.User("--field %q: expected key=value", f).WithHint("example: --field state=TODO")
		}
		if v == "" {
			return nil, errx.User("--field %q: empty value", f).WithHint("use `--query '%s IS EMPTY'` to match items with no %s", k, k)
		}
		out = append(out, core.FieldFilter{Key: k, Value: v})
	}
	return out, nil
}

func emitStderrNotes(w io.Writer, notes []datamodel.Warning) {
	for _, n := range notes {
		_, _ = fmt.Fprintln(w, msgPrefix, warningfmt.Message(n))
	}
}

func renderListResult(w io.Writer, res *datamodel.ListResult, columns []string) {
	if res.Tree != nil {
		renderTreeGroups(w, res, columns)
		return
	}
	renderList(w, res, columns)
}

func renderList(w io.Writer, res *datamodel.ListResult, columns []string) {
	if res.Count == 0 {
		_, _ = fmt.Fprintln(w, msgNoItems)
		return
	}
	tw := newTabWriter(w)
	_, _ = fmt.Fprintln(tw, columnHeader(columns))
	for _, it := range res.Items {
		_, _ = fmt.Fprintln(tw, formatItemRow(columns, it))
	}
	_ = tw.Flush()
}

// knownColumns splits configured column names into those a list cell can
// render and those nothing can.
func knownColumns(configured []string) (known, unknown []string) {
	for _, name := range configured {
		if _, builtin := listCells[name]; builtin {
			known = append(known, name)
			continue
		}
		unknown = append(unknown, name)
	}
	return known, unknown
}

func warnUnknownColumns(w io.Writer, unknown []string) {
	warns := make([]string, 0, len(unknown))
	for _, name := range unknown {
		warns = append(warns, fmt.Sprintf("ui.list.columns: unknown column %q; ignored", name))
	}
	emitWarningLines(w, warns)
}

func columnsOrDefault(columns []string) []string {
	if len(columns) == 0 {
		return datamodel.DefaultListColumns
	}
	return columns
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
		cells[i] = itemCell(c, it)
	}
	return strings.Join(cells, "\t")
}

func itemCell(column string, it datamodel.ListItem) string {
	if cell, ok := listCells[column]; ok {
		return cell(it)
	}
	return "-"
}

var listCells = map[string]func(datamodel.ListItem) string{
	"number":        func(it datamodel.ListItem) string { return it.Number },
	"title":         func(it datamodel.ListItem) string { return it.Title },
	"type":          func(it datamodel.ListItem) string { return it.Type },
	"subtype":       func(it datamodel.ListItem) string { return ptr.DerefOr(it.Subtype, "-") },
	"state":         func(it datamodel.ListItem) string { return it.State },
	"category":      func(it datamodel.ListItem) string { return it.Category },
	"priority":      func(it datamodel.ListItem) string { return ptr.DerefOr(it.Priority, "-") },
	"owner":         func(it datamodel.ListItem) string { return ptr.DerefOr(it.Owner, "-") },
	"labels":        func(it datamodel.ListItem) string { return strings.Join(it.Labels, ",") },
	"parent":        func(it datamodel.ListItem) string { return ptr.DerefOr(it.Parent, "-") },
	"parent_number": func(it datamodel.ListItem) string { return ptr.DerefOr(it.ParentNumber, "-") },
	"resolution":    func(it datamodel.ListItem) string { return ptr.DerefOr(it.Resolution, "-") },
	"due":           func(it datamodel.ListItem) string { return ptr.DerefOr(it.Due, "-") },
	"id":            func(it datamodel.ListItem) string { return it.ID },
}
