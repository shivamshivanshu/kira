package core

import (
	"github.com/shivamshivanshu/kira/internal/config"
	"github.com/shivamshivanshu/kira/internal/item"
)

// The view types are the stable --json contract (docs/design/04-cli.md).
// Field names and JSON tags are the frozen shapes; keep them additive-only.

// CreateResult is the result of a successful create.
type CreateResult struct {
	ID     string `json:"id"`
	Number string `json:"number"`
	Type   string `json:"type"`
	Title  string `json:"title"`
	State  string `json:"state"`
	Path   string `json:"path"`
}

// ListItem is one row of list/query output.
type ListItem struct {
	ID       string   `json:"id"`
	Number   string   `json:"number"`
	Title    string   `json:"title"`
	Type     string   `json:"type"`
	State    string   `json:"state"`
	Category string   `json:"category"`
	Owner    *string  `json:"owner"`
	Labels   []string `json:"labels"`
	Epic     *string  `json:"epic"`
}

// ListResult is the list/query envelope. Tree is present only when tree
// rendering is requested (list --tree / query's default), grouping the items by
// parent epic (docs/design/04-cli.md query).
type ListResult struct {
	Items []ListItem  `json:"items"`
	Count int         `json:"count"`
	Tree  []TreeGroup `json:"tree,omitempty"`

	StderrNotes []string `json:"-"`
}

// TreeGroup is one epic's bucket in the tree grouping: the epic's ULID and
// display number (both null for the orphan bucket of items with no epic) and
// the ULIDs of its member items, in list order.
type TreeGroup struct {
	Epic       *string  `json:"epic"`
	EpicNumber *string  `json:"epic_number"`
	Items      []string `json:"items"`
}

// TreeNode is one node in the kira tree hierarchy render; Children recurses.
type TreeNode struct {
	ID       string     `json:"id"`
	Number   string     `json:"number"`
	Type     string     `json:"type"`
	Title    string     `json:"title"`
	Children []TreeNode `json:"children"`
}

// TreeResult is the kira tree envelope. Root is the id the render was scoped to
// (null for the whole forest); Nodes are the top-level nodes.
type TreeResult struct {
	Root  *string    `json:"root"`
	Nodes []TreeNode `json:"nodes"`
}

// CommentView is one comment in a ShowResult.
type CommentView struct {
	ID     string `json:"id"`
	Author string `json:"author"`
	Ts     string `json:"ts"`
	Text   string `json:"text"`
}

// ShowResult is the full item detail. blocks, linked_commits, and history_tail
// are index-derived (docs/design/01-architecture.md §4) and stay empty until
// the index lands in M2; they are always present so the shape never changes.
type ShowResult struct {
	ID            string              `json:"id"`
	Number        string              `json:"number"`
	Aliases       []string            `json:"aliases"`
	Type          string              `json:"type"`
	Subtype       *string             `json:"subtype"`
	Title         string              `json:"title"`
	State         string              `json:"state"`
	Category      string              `json:"category"`
	Resolution    *string             `json:"resolution"`
	Priority      *string             `json:"priority"`
	Rank          *string             `json:"rank"`
	Sprint        *string             `json:"sprint"`
	Due           *string             `json:"due"`
	Owner         *string             `json:"owner"`
	Reporter      *string             `json:"reporter"`
	Labels        []string            `json:"labels"`
	Epic          *string             `json:"epic"`
	BlockedBy     []string            `json:"blocked_by"`
	Links         map[string][]string `json:"links"`
	Blocks        []string            `json:"blocks"`
	Estimate      *float64            `json:"estimate"`
	Created       string              `json:"created"`
	Updated       string              `json:"updated"`
	Body          string              `json:"body"`
	Comments      []CommentView       `json:"comments"`
	LinkedCommits []any               `json:"linked_commits"`
	HistoryTail   []any               `json:"history_tail"`
}

func categoryString(cfg *config.Config, typ, state string) string {
	if c, ok := categoryOf(cfg, typ, state); ok {
		return string(c)
	}
	return ""
}

func listItemOf(cfg *config.Config, it *item.Item) ListItem {
	return ListItem{
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

func showResultOf(cfg *config.Config, it *item.Item) ShowResult {
	comments := item.ParseComments(it.Body)
	views := make([]CommentView, len(comments))
	for i, c := range comments {
		views[i] = CommentView{ID: c.ID, Author: c.Author, Ts: c.Ts, Text: c.Body}
	}
	return ShowResult{
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

// linksView renders the links map for JSON with every known link type present
// (empty list, never null, for an absent type), so the shape is fixed across
// items — the documented show shape (docs/design/04-cli.md show).
func linksView(links map[string][]string) map[string][]string {
	view := make(map[string][]string, len(item.LinkTypes))
	for _, typ := range item.LinkTypes {
		view[typ] = nonNil(links[typ])
	}
	return view
}

// nonNil normalizes a nil slice to a non-nil empty slice so JSON renders `[]`,
// never `null`, for a required-but-empty list field.
func nonNil[T any](xs []T) []T {
	if xs == nil {
		return []T{}
	}
	return xs
}
