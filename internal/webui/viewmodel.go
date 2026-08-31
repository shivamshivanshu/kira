package webui

import (
	"encoding/json"
	"maps"
	"net/url"
	"slices"
	"strings"

	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/showfmt"
	"github.com/shivamshivanshu/kira/internal/timex"
)

type pageData struct {
	Root, View, Query, Filter, BoardKey, Type, Sprint string
	State, Owner, Priority, Label                     string
	Selected, URL, Token, Message, Error              string
	Revision, Action                                  string
	Epic, Since                                       string
	Weeks                                             int
	Form                                              url.Values
	Config                                            *datamodel.Config
	Items                                             []datamodel.ListItem
	AllItems                                          []datamodel.ListItem
	Tree                                              []treeRow
	Board                                             *datamodel.BoardResult
	Stats                                             *datamodel.StatsResult
	Detail                                            *datamodel.ShowResult
	Active                                            *core.ActiveItemSummary
	Sprints                                           []datamodel.SprintListRow
	Progress                                          map[string]datamodel.EpicProgress
	Targets                                           []string
	Settings                                          []string
	Explain                                           *datamodel.ExplainResult
	Bulk                                              []bulkOutcome
	Commit                                            *commitView
	cache                                             *pageCache
}

type pageCache struct {
	query    url.Values
	names    map[string]string
	selected map[string]bool
}

func (p *pageData) prepare() {
	c := &pageCache{query: pageQuery(p.URL), names: make(map[string]string, len(p.AllItems)*2), selected: make(map[string]bool, len(p.Form["ids"]))}
	items := p.AllItems
	if len(items) == 0 {
		items = p.Items
	}
	for _, it := range items {
		if _, exists := c.names[it.ID]; !exists {
			c.names[it.ID] = it.Number
		}
		if _, exists := c.names[it.Number]; !exists {
			c.names[it.Number] = it.Number
		}
	}
	for _, id := range p.Form["ids"] {
		c.selected[id] = true
	}
	p.cache = c
}

func pageQuery(raw string) url.Values {
	u, err := url.Parse(raw)
	if err != nil {
		return url.Values{}
	}
	q := u.Query()
	q.Del("message")
	q.Del("error")
	q.Del("_notice")
	return q
}

type pickerItem struct {
	ID       string `json:"id"`
	Number   string `json:"number"`
	Title    string `json:"title"`
	State    string `json:"state"`
	Category string `json:"category"`
}

func (p pageData) PickerItems() []pickerItem {
	items := make([]pickerItem, len(p.AllItems))
	for i, it := range p.AllItems {
		items[i] = pickerItem{it.ID, it.Number, it.Title, it.State, it.Category}
	}
	return items
}

// CopyValues keeps row data compact; format labels and buttons are shared.
func (p pageData) CopyValues(id, number, title string) string {
	forms := p.CopyForms(id, number, title)
	values := make([]string, len(forms))
	for i, form := range forms {
		values[i] = form.Value
	}
	encoded, _ := json.Marshal(values)
	return string(encoded)
}

func (p pageData) CopyLabels() []copyOption {
	options := make([]copyOption, len(showfmt.Forms))
	for i, form := range showfmt.Forms {
		options[i] = copyOption{Name: string(form), Label: form.Label()}
	}
	return options
}

type commitView struct {
	SHA, Text string
}

type commitCard struct {
	datamodel.CommitLink
	URL string
}

type copyOption struct {
	Name, Label, Value string
}

type diffLine struct {
	Class, Text string
}

type treeRow struct {
	Item  datamodel.ListItem
	Depth int
}

type itemSummary struct {
	Total, InProgress, Done, Overdue int
}

func (p pageData) Link(pairs ...string) string {
	var q url.Values
	if p.cache != nil {
		q = maps.Clone(p.cache.query)
	} else {
		q = pageQuery(p.URL)
	}
	for i := 0; i+1 < len(pairs); i += 2 {
		if pairs[i] == "item" || pairs[i] == "view" {
			q.Del("commit")
		}
		if pairs[i] == "item" && pairs[i+1] != "" && p.View == "manage" {
			q.Set("view", "tree")
		}
		if pairs[i+1] == "" {
			q.Del(pairs[i])
		} else {
			q.Set(pairs[i], pairs[i+1])
		}
	}
	return (&url.URL{Path: "/", RawQuery: q.Encode()}).String()
}

func (p pageData) Summary() itemSummary {
	s := itemSummary{Total: len(p.Items)}
	now := timex.Now()
	for _, item := range p.Items {
		switch datamodel.Category(item.Category) {
		case datamodel.CategoryDoing:
			s.InProgress++
		case datamodel.CategoryDone:
			s.Done++
		}
		if datamodel.IsOverdue(item.Due, item.Category, now) {
			s.Overdue++
		}
	}
	return s
}

func (p pageData) BoardLink(key string) string {
	if p.View == "stats" || p.View == "manage" {
		return p.Link("board", key, "view", "tree")
	}
	return p.Link("board", key)
}

func (p pageData) BoardSelected(key string) bool {
	return (p.View == "tree" || p.View == "board") && p.BoardKey == key
}

func (p pageData) SprintLink(key string) string {
	if p.View == "manage" {
		return p.Link("sprint", key, "view", "tree")
	}
	return p.Link("sprint", key)
}

func (p pageData) StateOptions() []string {
	states := []string{p.State}
	if p.Config != nil {
		for typ, workflow := range p.Config.Workflows {
			if p.Type != "" && typ != p.Type {
				continue
			}
			for _, state := range workflow.States {
				states = append(states, state.Key)
			}
		}
	}
	return sortedOptions(states)
}

func (p pageData) OwnerOptions() []string {
	owners := []string{p.Owner}
	if p.Config != nil {
		for _, person := range p.Config.People.Known {
			owners = append(owners, person.Name)
		}
	}
	for _, item := range p.AllItems {
		if item.Owner != nil {
			owners = append(owners, *item.Owner)
		}
	}
	return slices.DeleteFunc(sortedOptions(owners), func(owner string) bool { return owner == "@me" })
}

func (p pageData) LabelOptions() []string {
	labels := []string{p.Label}
	if p.Config != nil {
		labels = append(labels, p.Config.Labels.Known...)
	}
	for _, item := range p.AllItems {
		labels = append(labels, item.Labels...)
	}
	return sortedOptions(labels)
}

func sortedOptions(values []string) []string {
	slices.Sort(values)
	values = slices.Compact(values)
	if len(values) > 0 && values[0] == "" {
		values = values[1:]
	}
	return values
}

func (p pageData) HasRelationships() bool {
	if p.Detail == nil {
		return false
	}
	if p.Detail.Epic != nil || len(p.Detail.BlockedBy) > 0 || len(p.Detail.Blocks) > 0 {
		return true
	}
	for _, targets := range p.Detail.Links {
		if len(targets) > 0 {
			return true
		}
	}
	return false
}

func (p pageData) Heading() string {
	if p.View == "manage" {
		return "Project settings"
	}
	if p.View == "stats" {
		return "Project insights"
	}
	if p.BoardKey != "" && p.Config != nil {
		if board, ok := p.Config.BoardByKey(p.BoardKey); ok && board.Name != "" {
			return board.Name
		}
		return p.BoardKey
	}
	return "All work"
}

func (p pageData) ItemName(ref string) string {
	if p.cache != nil {
		if name, ok := p.cache.names[ref]; ok {
			return name
		}
		return ref
	}
	items := p.AllItems
	if len(items) == 0 {
		items = p.Items
	}
	for _, item := range items {
		if item.ID == ref || item.Number == ref {
			return item.Number
		}
	}
	return ref
}

func (p pageData) BulkSelected(ref string) bool {
	if p.cache != nil {
		return p.cache.selected[ref]
	}
	return slices.Contains(p.Form["ids"], ref)
}

func (p pageData) CommitCard(itemID string, commit datamodel.CommitLink) commitCard {
	return commitCard{CommitLink: commit, URL: p.Link("item", itemID, "commit", commit.SHA)}
}

func (p pageData) CopyForms(id, number, title string) []copyOption {
	var w datamodel.Workon
	if p.Config != nil {
		w = p.Config.Workon
	}
	options := make([]copyOption, 0, len(showfmt.Forms))
	for _, form := range showfmt.Forms {
		value, err := showfmt.Format(form, showfmt.Item{ID: id, Number: number, Title: title}, w)
		if err == nil {
			options = append(options, copyOption{Name: string(form), Label: form.Label(), Value: value})
		}
	}
	return options
}

func (c *commitView) Lines() []diffLine {
	lines := strings.Split(c.Text, "\n")
	result := make([]diffLine, len(lines))
	for i, line := range lines {
		class := ""
		switch {
		case strings.HasPrefix(line, "diff "), strings.HasPrefix(line, "@@"), strings.HasPrefix(line, "+++"), strings.HasPrefix(line, "---"):
			class = "diff-header"
		case strings.HasPrefix(line, "+"):
			class = "diff-add"
		case strings.HasPrefix(line, "-"):
			class = "diff-remove"
		}
		result[i] = diffLine{Class: class, Text: line}
	}
	return result
}

func (p pageData) FormValue(action, key, fallback string) string {
	if p.Action == action && p.Form.Has(key) {
		return p.Form.Get(key)
	}
	return fallback
}

func (p pageData) ObjectFormValue(action, object, key, fallback string) string {
	if p.Form.Get("key") == object {
		return p.FormValue(action, key, fallback)
	}
	return fallback
}
