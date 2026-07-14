package datamodel

type WarnCode string

const (
	WarnIndexFallback  WarnCode = "index_fallback"
	WarnNoActiveSprint WarnCode = "no_active_sprint"
	WarnCloseUnknown   WarnCode = "close_unknown"
	WarnCloseFailed    WarnCode = "close_failed"
	WarnLiteral        WarnCode = "literal"
	WarnOrphanType     WarnCode = "orphan_type"
)

var WarnCodes = []WarnCode{
	WarnIndexFallback, WarnNoActiveSprint, WarnCloseUnknown,
	WarnCloseFailed, WarnLiteral, WarnOrphanType,
}

type Warning struct {
	Code WarnCode
	Args []string
}

type Skew struct {
	Ref   string `json:"ref"`
	At    string `json:"at"`
	AtID  string `json:"at_id"`
	NowID string `json:"now_id"`
}

type CreateResult struct {
	ID         string   `json:"id"`
	Number     string   `json:"number"`
	Board      string   `json:"board"`
	Type       string   `json:"type"`
	Title      string   `json:"title"`
	State      string   `json:"state"`
	Category   string   `json:"category"`
	Owner      *string  `json:"owner"`
	Labels     []string `json:"labels"`
	Epic       *string  `json:"epic"`
	EpicNumber *string  `json:"epic_number"`
	Priority   *string  `json:"priority"`
	Resolution *string  `json:"resolution"`
	Path       string   `json:"path"`
}

type ListItem struct {
	ID         string   `json:"id"`
	Number     string   `json:"number"`
	Board      string   `json:"board"`
	Title      string   `json:"title"`
	Type       string   `json:"type"`
	State      string   `json:"state"`
	Category   string   `json:"category"`
	Owner      *string  `json:"owner"`
	Labels     []string `json:"labels"`
	Epic       *string  `json:"epic"`
	EpicNumber *string  `json:"epic_number"`
	Priority   *string  `json:"priority"`
	Resolution *string  `json:"resolution"`
	Due        *string  `json:"due"`
}

type ListResult struct {
	Items []ListItem  `json:"items"`
	Count int         `json:"count"`
	Tree  []TreeGroup `json:"tree,omitempty"`

	StderrNotes []Warning `json:"-"`
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

	StderrNotes []Warning `json:"-"`
}

type CommentView struct {
	ID     string `json:"id"`
	Author string `json:"author"`
	Ts     string `json:"ts"`
	Text   string `json:"text"`
}

type CommitLink struct {
	SHA     string `json:"sha"`
	Subject string `json:"subject"`
	Author  string `json:"author"`
	Ts      string `json:"ts"`
}

type HistoryEvent struct {
	Ts    string  `json:"ts"`
	Field string  `json:"field"`
	From  *string `json:"from"`
	To    *string `json:"to"`
}

type ShowResult struct {
	ID            string              `json:"id"`
	Number        string              `json:"number"`
	Board         string              `json:"board"`
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
	LinkedCommits []CommitLink        `json:"linked_commits"`
	ReferencedBy  []CommitLink        `json:"referenced_by"`
	HistoryTail   []HistoryEvent      `json:"history_tail"`
	Skew          *Skew               `json:"skew,omitempty"`

	StderrNotes []Warning `json:"-"`
}

type MutationResult struct {
	ID      string   `json:"id"`
	Number  string   `json:"number"`
	Changed []string `json:"changed"`
}

type BulkOutcome struct {
	Ref    string `json:"ref"`
	Result any    `json:"result,omitempty"`
	Error  string `json:"error,omitempty"`
}

type DiffResult struct {
	From  string     `json:"from"`
	To    string     `json:"to"`
	Items []DiffItem `json:"items"`
}

type DiffItem struct {
	ID         string         `json:"id"`
	Number     string         `json:"number"`
	Title      string         `json:"title"`
	Status     DiffStatus     `json:"status"`
	Renumbered *RenumberEvent `json:"renumbered,omitempty"`
	Changes    []FieldChange  `json:"changes,omitempty"`
}

type RenumberEvent struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type FieldChange struct {
	Field string `json:"field"`
	From  string `json:"from"`
	To    string `json:"to"`
}

type DiffStatus string

const (
	DiffCreated DiffStatus = "created"
	DiffDeleted DiffStatus = "deleted"
	DiffChanged DiffStatus = "changed"
)

var DiffStatuses = []DiffStatus{DiffCreated, DiffDeleted, DiffChanged}

type ChangesResult struct {
	Since string        `json:"since"`
	Head  string        `json:"head"`
	Items []ChangedItem `json:"items"`
}

type ChangedItem struct {
	ID     string     `json:"id"`
	Number string     `json:"number"`
	Title  string     `json:"title"`
	Status DiffStatus `json:"status"`
	Body   *BodyDelta `json:"body,omitempty"`
	Events []Event    `json:"events"`
}

type BodyDelta struct {
	Added   int `json:"added"`
	Removed int `json:"removed"`
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

type ConfigSetResult struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type ConfigInitResult struct {
	Path    string   `json:"path"`
	Created bool     `json:"created"`
	Files   []string `json:"files"`
}

type LabelCreateResult struct {
	Created      []string `json:"created"`
	AlreadyKnown []string `json:"already_known"`
}

type LabelCount struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type LabelListResult struct {
	Labels []LabelCount `json:"labels"`
}

type MergeResult struct {
	ID         string   `json:"id"`
	Number     string   `json:"number"`
	Arbitrated []string `json:"arbitrated,omitempty"`
}

type ResolveResult struct {
	Resolved []MergeResult `json:"resolved"`
	Skipped  []string      `json:"skipped,omitempty"`
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

type SprintView struct {
	Key   string `json:"key"`
	Name  string `json:"name"`
	Start string `json:"start"`
	End   string `json:"end"`
}

type SprintCreateResult struct {
	Created bool       `json:"created"`
	Sprint  SprintView `json:"sprint"`
}

type SprintItemCounts struct {
	Total int `json:"total"`
	Done  int `json:"done"`
}

type SprintListRow struct {
	SprintView
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

type BoardView struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Default     bool   `json:"default,omitempty"`
	Archived    bool   `json:"archived,omitempty"`
}

type BoardCreateResult struct {
	Created bool      `json:"created"`
	Board   BoardView `json:"board"`
}

type BoardListResult struct {
	Boards []BoardView `json:"boards"`
}

type BoardUpdateResult struct {
	Board BoardView `json:"board"`
}

type BoardMoveResult struct {
	ID    string `json:"id"`
	From  string `json:"from"`
	To    string `json:"to"`
	Board string `json:"board"`
}

type Event struct {
	Ts        string `json:"ts"`
	Field     string `json:"field"`
	Old       string `json:"old"`
	New       string `json:"new"`
	CommitSHA string `json:"commit"`
}

type LogEntry struct {
	Kind    string `json:"kind"`
	Ts      string `json:"ts"`
	Field   string `json:"field,omitempty"`
	Old     string `json:"old,omitempty"`
	New     string `json:"new,omitempty"`
	SHA     string `json:"sha,omitempty"`
	Subject string `json:"subject,omitempty"`
	Author  string `json:"author,omitempty"`
}

type LogResult struct {
	ID      string     `json:"id"`
	Number  string     `json:"number"`
	Entries []LogEntry `json:"entries"`
}

type IndexResult struct {
	Action string   `json:"action"`
	Reason string   `json:"reason"`
	Items  int      `json:"items"`
	Closed []string `json:"closed"`

	StderrNotes []Warning `json:"-"`
}

type StatsResult struct {
	Scope      *StatsScope  `json:"scope,omitempty"`
	Completion *Completion  `json:"completion,omitempty"`
	CycleTime  *Percentiles `json:"cycle_time_days,omitempty"`
	LeadTime   *Percentiles `json:"lead_time_days,omitempty"`
	Throughput []int        `json:"throughput_per_week,omitempty"`
	Reopens    *Reopens     `json:"reopens,omitempty"`
}

type StatsScope struct {
	Epic       string `json:"epic,omitempty"`
	EpicNumber string `json:"epic_number,omitempty"`
	Sprint     string `json:"sprint,omitempty"`
	Since      string `json:"since,omitempty"`
	Weeks      int    `json:"weeks"`
}

type Completion struct {
	Done    int     `json:"done"`
	Total   int     `json:"total"`
	Dropped int     `json:"dropped"`
	Pct     float64 `json:"pct"`
}

type Percentiles struct {
	P50       float64 `json:"p50"`
	P90       float64 `json:"p90"`
	N         int     `json:"n"`
	DegradedN int     `json:"degraded_n,omitempty"`
}

type Reopens struct {
	Count int      `json:"count"`
	Items []string `json:"items"`
}

type BlameField struct {
	Field      string `json:"field"`
	Value      string `json:"value"`
	When       string `json:"when"`
	By         string `json:"by"`
	SourceKind string `json:"source_kind"`
	Degraded   bool   `json:"degraded,omitempty"`
}

type BlameResult struct {
	ID     string       `json:"id"`
	Number string       `json:"number"`
	Fields []BlameField `json:"fields"`
}

const JSONContractVersion = 1

type VersionResult struct {
	Version      string `json:"version"`
	JSONContract int    `json:"json_contract"`
	Go           string `json:"go"`
	Commit       string `json:"commit,omitempty"`
}

type BoardColumn struct {
	State    string     `json:"state"`
	Category string     `json:"category"`
	Wip      int        `json:"wip"`
	Count    int        `json:"count"`
	Items    []ListItem `json:"items"`
}

type BoardResult struct {
	Type    string        `json:"type"`
	Columns []BoardColumn `json:"columns"`
}

func (r *BoardResult) Empty() bool {
	for _, c := range r.Columns {
		if len(c.Items) > 0 {
			return false
		}
	}
	return true
}

type HookStatus struct {
	Name      string `json:"name"`
	Installed bool   `json:"installed"`
	Chained   bool   `json:"chained"`
}

type HooksInstallResult struct {
	Hooks       []HookStatus `json:"hooks"`
	MergeDriver bool         `json:"merge_driver"`
}

type HooksValidateResult struct {
	Hooks       []HookStatus `json:"hooks"`
	MergeDriver bool         `json:"merge_driver"`
	OK          bool         `json:"ok"`
}

type WorkonResult struct {
	ID            string `json:"id"`
	Number        string `json:"number"`
	Branch        string `json:"branch"`
	BranchCreated bool   `json:"branch_created"`
	Worktree      string `json:"worktree,omitempty"`
	Moved         bool   `json:"moved"`
	From          string `json:"from,omitempty"`
	To            string `json:"to,omitempty"`
}

type Renumbering struct {
	ID   string `json:"id"`
	From string `json:"from"`
	To   string `json:"to"`
}

type ReconcileResult struct {
	Renumbered []Renumbering `json:"renumbered"`
}
