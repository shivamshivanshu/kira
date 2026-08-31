package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/entityschema"
	"github.com/shivamshivanshu/kira/internal/ptr"
	"github.com/shivamshivanshu/kira/internal/warningfmt"
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
			schemas, err := s.Schemas()
			if err != nil {
				return err
			}
			cols, unknown := resolveColumns(cfg.UI.List.Columns, schemas)
			warnUnknownColumns(cmd.ErrOrStderr(), unknown)
			renderListResult(cmd.OutOrStdout(), res, cols)
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
		_, _ = fmt.Fprintln(w, msgPrefix, warningfmt.Message(n))
	}
}

func renderListResult(w io.Writer, res *datamodel.ListResult, cols []string) {
	if res.Tree != nil {
		renderTreeGroups(w, res, cols)
		return
	}
	renderList(w, res, cols)
}

func renderList(w io.Writer, res *datamodel.ListResult, cols []string) {
	if res.Count == 0 {
		_, _ = fmt.Fprintln(w, msgNoItems)
		return
	}
	tw := newTabWriter(w)
	_, _ = fmt.Fprintln(tw, columnHeader(cols))
	for _, it := range res.Items {
		_, _ = fmt.Fprintln(tw, formatItemRow(cols, it))
	}
	_ = tw.Flush()
}

func resolveColumns(columns []string, schemas map[string]entityschema.Schema) (cols, unknown []string) {
	for _, c := range columns {
		if _, ok := listCells[c]; ok || declaredExtraColumn(schemas, c) {
			cols = append(cols, c)
			continue
		}
		unknown = append(unknown, c)
	}
	if len(cols) == 0 {
		return datamodel.DefaultListColumns, unknown
	}
	return cols, unknown
}

func declaredExtraColumn(schemas map[string]entityschema.Schema, name string) bool {
	for typ := range schemas {
		if _, ok := entityschema.DeclaredExtra(schemas, typ, name); ok {
			return true
		}
	}
	return false
}

func warnUnknownColumns(w io.Writer, unknown []string) {
	msgs := make([]string, len(unknown))
	for i, c := range unknown {
		msgs[i] = fmt.Sprintf("ui.list.columns: unknown column %q; ignored", c)
	}
	emitWarningLines(w, msgs)
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
		cells[i] = cellValue(c, it)
	}
	return strings.Join(cells, "\t")
}

func cellValue(col string, it datamodel.ListItem) string {
	if cell, ok := listCells[col]; ok {
		return cell(it)
	}
	return extraCell(it.Extras[col])
}

func extraCell(v datamodel.ExtraJSON) string {
	switch v := v.(type) {
	case string:
		return v
	case []string:
		return strings.Join(v, ",")
	default:
		return "-"
	}
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
