package core

import (
	"github.com/shivamshivanshu/kira/internal/codec"
	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func categoryString(cfg *datamodel.Config, typ, state string) string {
	if c, ok := categoryOf(cfg, typ, state); ok {
		return string(c)
	}
	return ""
}

func listItemOf(cfg *datamodel.Config, it *datamodel.Item) datamodel.ListItem {
	return datamodel.ListItem{
		ID:       it.ID,
		Number:   it.Number,
		Title:    it.Title,
		Type:     it.Type,
		State:    it.State,
		Category: categoryString(cfg, it.Type, it.State),
		Owner:    it.Owner,
		Labels:   nonNil(it.Labels),
		Epic:     it.Epic,
	}
}

func showResultOf(cfg *datamodel.Config, it *datamodel.Item) datamodel.ShowResult {
	comments := codec.ParseComments(it.Body)
	views := make([]datamodel.CommentView, len(comments))
	for i, c := range comments {
		views[i] = datamodel.CommentView{ID: c.ID, Author: c.Author, Ts: c.Ts, Text: c.Body}
	}
	return datamodel.ShowResult{
		ID:            it.ID,
		Number:        it.Number,
		Aliases:       nonNil(it.Aliases),
		Type:          it.Type,
		Subtype:       it.Subtype,
		Title:         it.Title,
		State:         it.State,
		Category:      categoryString(cfg, it.Type, it.State),
		Resolution:    it.Resolution,
		Priority:      it.Priority,
		Rank:          it.Rank,
		Sprint:        it.Sprint,
		Due:           it.Due,
		Owner:         it.Owner,
		Reporter:      it.Reporter,
		Labels:        nonNil(it.Labels),
		Epic:          it.Epic,
		BlockedBy:     nonNil(it.BlockedBy),
		Links:         linksView(it.Links),
		Blocks:        []string{},
		Estimate:      it.Estimate,
		Created:       it.Created,
		Updated:       it.Updated,
		Body:          it.Body,
		Comments:      views,
		LinkedCommits: []any{},
		HistoryTail:   []any{},
	}
}

func linksView(links map[string][]string) map[string][]string {
	view := make(map[string][]string, len(datamodel.LinkTypes))
	for _, typ := range datamodel.LinkTypes {
		view[typ] = nonNil(links[typ])
	}
	return view
}
