package webui

import (
	"context"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/shivamshivanshu/kira/internal/config"
	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/warningfmt"
)

func (s *server) loadPage(ctx context.Context, u *url.URL) (pageData, error) {
	q := u.Query()
	p := pageData{
		Root: s.store.Root(), View: q.Get("view"), Query: q.Get("q"), Filter: q.Get("filter"),
		BoardKey: q.Get("board"), Type: q.Get("type"), Sprint: q.Get("sprint"), Selected: q.Get("item"),
		State: q.Get("state"), Owner: q.Get("owner"), Priority: q.Get("priority"), Label: q.Get("label"),
		Epic: q.Get("epic"), Since: q.Get("since"),
		URL: localReturn(u.RequestURI()), Token: s.token,
	}
	if p.View == "" {
		p.View = "tree"
	}
	if p.Type == "" && p.View == "board" {
		p.Type = datamodel.TypeTicket
	}
	cfg, err := s.store.Config()
	if err != nil {
		return p, err
	}
	p.Config = cfg
	if p.View == "manage" {
		p.Settings, p.Explain = config.SetKeys(), core.Explain(cfg)
	}
	if p.View != "tree" && p.View != "board" && p.View != "stats" && p.View != "manage" {
		return p, errx.User("unknown dashboard view %q", p.View)
	}
	snapshot, err := s.store.Snapshot(cfg)
	var all *datamodel.ListResult
	if err == nil {
		all, err = snapshot.List(core.ListOpts{})
	}
	if err != nil {
		if p.View != "manage" {
			return p, err
		}
		p.Message = userError(err)
		for _, sprint := range cfg.Sprints {
			p.Sprints = append(p.Sprints, datamodel.SprintListRow{SprintView: datamodel.SprintView(sprint)})
		}
	} else {
		p.AllItems = all.Items
		p.Message = warningText(all.StderrNotes)
		p.Sprints = snapshot.SprintList().Sprints
		p.Active = snapshot.ActiveSummary()
	}
	if p.View == "manage" {
		return p, nil
	}
	if p.BoardKey != "" && p.View != "stats" {
		board, ok := cfg.BoardByKey(p.BoardKey)
		if !ok {
			return p, errx.User("no such board %q", p.BoardKey)
		}
		p.BoardKey = board.Key
	}
	if p.View == "stats" {
		if weeks := q.Get("weeks"); weeks != "" {
			p.Weeks, err = strconv.Atoi(weeks)
			if err != nil || p.Weeks < 1 || p.Weeks > 260 {
				return p, errx.User("weeks must be between 1 and 260")
			}
		}
		p.Stats, err = snapshot.Stats(core.StatsOpts{Sprint: p.Sprint, Parent: p.Epic, Since: p.Since, Weeks: p.Weeks})
		if err == nil && p.Stats.Scope != nil {
			p.Weeks = p.Stats.Scope.Weeks
		}
	} else {
		list := all
		if p.Query != "" || p.Filter != "" || p.Type != "" || p.BoardKey != "" || p.Sprint != "" ||
			p.State != "" || p.Owner != "" || p.Priority != "" || p.Label != "" {
			list, err = snapshot.List(core.ListOpts{
				Query: p.Query, Filter: p.Filter, Type: p.Type,
				Board: p.BoardKey, Sprint: p.Sprint, State: p.State,
				Owner: p.Owner, Priority: p.Priority, Label: p.Label,
			})
		}
		if err == nil {
			p.Items = list.Items
			p.Message = warningText(list.StderrNotes)
			if p.View == "board" {
				p.Board, err = snapshot.Board(core.BoardOpts{
					Type: p.Type, Query: p.Query, Filter: p.Filter,
					Board: p.BoardKey, Sprint: p.Sprint, State: p.State,
					Owner: p.Owner, Priority: p.Priority, Label: p.Label,
				})
			} else {
				p.Tree, err = treeRows(snapshot, p.Items, p.AllItems)
			}
		}
	}
	if err != nil {
		return p, err
	}
	needsProgress := slices.ContainsFunc(p.Tree, func(row treeRow) bool { return row.Item.Type == datamodel.TypeEpic })
	if p.View == "board" {
		needsProgress = slices.ContainsFunc(p.Items, func(item datamodel.ListItem) bool { return item.Type == datamodel.TypeEpic })
	}
	if needsProgress {
		p.Progress, err = snapshot.ContainerProgress()
		if err != nil {
			return p, err
		}
	}
	var detail *core.ItemDetail
	if p.Selected != "" {
		detail, err = snapshot.Detail(p.Selected)
		if err != nil {
			return p, err
		}
		p.Detail, p.Revision = detail.Item, detail.Revision
		p.Selected = p.Detail.Number
		p.Targets = core.MoveTargets(p.Detail.Type)
		p.Message = joinMessages(p.Message, warningText(p.Detail.StderrNotes))
	}
	if sha := q.Get("commit"); sha != "" {
		if p.Detail == nil {
			return p, errx.User("select an item before opening a linked commit")
		}
		ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		patch, err := detail.CommitPatch(ctx, sha)
		if err != nil {
			return p, err
		}
		p.Commit = &commitView{SHA: sha, Text: patch}
	}
	return p, nil
}

func treeRows(snapshot *core.Snapshot, matched, all []datamodel.ListItem) ([]treeRow, error) {
	tree, err := snapshot.Tree("")
	if err != nil {
		return nil, err
	}
	items := make(map[string]datamodel.ListItem, len(all))
	for _, it := range all {
		items[it.ID] = it
	}
	wanted := make(map[string]bool, len(matched))
	for _, it := range matched {
		wanted[it.ID] = true
	}
	var mark func(datamodel.TreeNode) bool
	mark = func(node datamodel.TreeNode) bool {
		keep := wanted[node.ID]
		for _, child := range node.Children {
			if mark(child) {
				keep = true
			}
		}
		wanted[node.ID] = keep
		return keep
	}
	rows := make([]treeRow, 0, len(all))
	var walk func(datamodel.TreeNode, int)
	walk = func(node datamodel.TreeNode, depth int) {
		if !wanted[node.ID] {
			return
		}
		rows = append(rows, treeRow{Item: items[node.ID], Depth: depth})
		for _, child := range node.Children {
			walk(child, depth+1)
		}
	}
	for _, node := range tree.Nodes {
		mark(node)
		walk(node, 0)
	}
	return rows, nil
}

func joinMessages(parts ...string) string {
	var nonempty []string
	for _, part := range parts {
		if part != "" {
			nonempty = append(nonempty, part)
		}
	}
	return strings.Join(nonempty, "\n")
}

func warningText(warnings []datamodel.Warning) string {
	var messages []string
	for _, warning := range warnings {
		messages = append(messages, warningfmt.Message(warning))
	}
	return joinMessages(messages...)
}
