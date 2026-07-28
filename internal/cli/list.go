package cli

import (
	"fmt"
	"io"
	"slices"
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
			renderListResult(cmd.OutOrStdout(), res, pinnedColumns(cfg))
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
	cols := resolveColumns(columns, res.Columns)
	tw := newTabWriter(w)
	_, _ = fmt.Fprintln(tw, columnHeader(cols))
	for _, it := range res.Items {
		_, _ = fmt.Fprintln(tw, formatItemRow(cols, it))
	}
	_ = tw.Flush()
}

// pinnedColumns reports the columns the repo actually pins. The default config
// fills ui.list.columns in, so a value equal to the built-in default means "not
// customized" -- the same test `kira config explain` uses to tag a section
// default rather than repo.
func pinnedColumns(cfg *datamodel.Config) []string {
	if slices.Equal(cfg.UI.List.Columns, datamodel.DefaultListColumns) {
		return nil
	}
	return cfg.UI.List.Columns
}

// resolveColumns picks the table's columns: ui.list.columns wins, then what
// the matched types' schemas declare, then the built-in default. A configured
// name with no cell behind it is dropped rather than failing the listing,
// while a schema-declared one is kept — the schema names its own fields, and
// those render from the item's custom values.
func resolveColumns(configured, schemaDeclared []string) []string {
	builtins := make([]string, 0, len(configured))
	for _, c := range configured {
		if _, ok := listCells[c]; ok {
			builtins = append(builtins, c)
		}
	}
	switch {
	case len(builtins) > 0:
		return builtins
	case len(schemaDeclared) > 0:
		return schemaDeclared
	default:
		return datamodel.DefaultListColumns
	}
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
	if v, ok := it.Fields[column]; ok {
		return typedFieldValueAsText(v)
	}
	return "-"
}

var listCells = map[string]func(datamodel.ListItem) string{
	"number":      func(it datamodel.ListItem) string { return it.Number },
	"title":       func(it datamodel.ListItem) string { return it.Title },
	"type":        func(it datamodel.ListItem) string { return it.Type },
	"subtype":     func(it datamodel.ListItem) string { return ptr.DerefOr(it.Subtype, "-") },
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
