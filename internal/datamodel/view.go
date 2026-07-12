package datamodel

type CreateResult struct {
	ID     string `json:"id"`
	Number string `json:"number"`
	Type   string `json:"type"`
	Title  string `json:"title"`
	State  string `json:"state"`
	Path   string `json:"path"`
}

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

type ListResult struct {
	Items []ListItem  `json:"items"`
	Count int         `json:"count"`
	Tree  []TreeGroup `json:"tree,omitempty"`

	StderrNotes []string `json:"-"`
}

type TreeGroup struct {
	Epic       *string  `json:"epic"`
	EpicNumber *string  `json:"epic_number"`
	Items      []string `json:"items"`
}

type TreeNode struct {
	ID       string     `json:"id"`
	Number   string     `json:"number"`
	Type     string     `json:"type"`
	Title    string     `json:"title"`
	Children []TreeNode `json:"children"`
}

type TreeResult struct {
	Root  *string    `json:"root"`
	Nodes []TreeNode `json:"nodes"`
}

type CommentView struct {
	ID     string `json:"id"`
	Author string `json:"author"`
	Ts     string `json:"ts"`
	Text   string `json:"text"`
}

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

type MutationResult struct {
	ID      string   `json:"id"`
	Number  string   `json:"number"`
	Changed []string `json:"changed"`
}

type Match struct {
	ID     string `json:"id"`
	Number string `json:"number"`
	Line   int    `json:"line"`
	Text   string `json:"text"`
}

type FindResult struct {
	Matches []Match `json:"matches"`
}

type InitResult struct {
	Initialized bool   `json:"initialized"`
	Path        string `json:"path"`
	ProjectKey  string `json:"project_key"`
}

type MoveResult struct {
	ID        string   `json:"id"`
	Number    string   `json:"number"`
	From      string   `json:"from"`
	To        string   `json:"to"`
	Activated bool     `json:"activated"`
	Warnings  []string `json:"warnings,omitempty"`
}

type CommentResult struct {
	ID        string `json:"id"`
	Number    string `json:"number"`
	CommentID string `json:"comment_id"`
}

type FilterView struct {
	Name  string `json:"name"`
	Query string `json:"query"`
}

type FilterListResult struct {
	Filters []FilterView `json:"filters"`
}

type SprintJSON struct {
	Key   string `json:"key"`
	Name  string `json:"name"`
	Start string `json:"start"`
	End   string `json:"end"`
}

type SprintCreateResult struct {
	Created bool       `json:"created"`
	Sprint  SprintJSON `json:"sprint"`
}

type SprintItemCounts struct {
	Total int `json:"total"`
	Done  int `json:"done"`
}

type SprintListRow struct {
	SprintJSON
	Active bool             `json:"active"`
	Items  SprintItemCounts `json:"items"`
}

type SprintListResult struct {
	Sprints []SprintListRow `json:"sprints"`
}

type SprintActivateResult struct {
	Activated string `json:"activated"`
	Previous  string `json:"previous,omitempty"`
}

type SprintCloseResult struct {
	Closed     string   `json:"closed"`
	WasActive  bool     `json:"was_active"`
	Unfinished []string `json:"unfinished"`
	MovedTo    string   `json:"moved_to,omitempty"`
}

type StatsResult struct {
	Burndown *Burndown `json:"burndown,omitempty"`
	Velocity *Velocity `json:"velocity,omitempty"`
}

type BurndownDay struct {
	Date      string  `json:"date"`
	Remaining float64 `json:"remaining"`
	Ideal     float64 `json:"ideal"`
}

type Burndown struct {
	Sprint      string        `json:"sprint"`
	Start       string        `json:"start"`
	End         string        `json:"end"`
	Unit        string        `json:"unit"`
	Days        []BurndownDay `json:"days"`
	Unestimated int           `json:"unestimated"`
	DegradedN   int           `json:"degraded_n"`
}

type VelocitySprint struct {
	Key       string  `json:"key"`
	Completed float64 `json:"completed"`
}

type Velocity struct {
	Unit      string           `json:"unit"`
	Sprints   []VelocitySprint `json:"sprints"`
	Trailing3 float64          `json:"trailing_3"`
}
