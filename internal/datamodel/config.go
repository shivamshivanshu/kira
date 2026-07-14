package datamodel

type Config struct {
	Version     int                 `yaml:"version"`
	Project     Project             `yaml:"project"`
	ID          Identity            `yaml:"id"`
	Workflows   map[string]Workflow `yaml:"workflows"`
	Labels      Vocab               `yaml:"labels"`
	People      People              `yaml:"people"`
	Priorities  EnumVocab           `yaml:"priorities"`
	Subtypes    EnumVocab           `yaml:"subtypes"`
	Resolutions EnumVocab           `yaml:"resolutions"`

	ResolutionsDropped []string `yaml:"resolutions_dropped"`

	Filters    map[string]string `yaml:"filters"`
	Sprints    []Sprint          `yaml:"sprints"`
	Commit     Commit            `yaml:"commit"`
	Merge      Merge             `yaml:"merge"`
	Sync       Sync              `yaml:"sync"`
	Workon     Workon            `yaml:"workon"`
	UI         UI                `yaml:"ui"`
	Git        Git               `yaml:"git"`
	Estimate   Estimate          `yaml:"estimate"`
	Automation []AutomationHook  `yaml:"automation"`
	Fields     map[string]any    `yaml:"fields"`

	UserAutomation []AutomationHook `yaml:"-" json:"-"`
}

type Project struct {
	Key  string `yaml:"key"`
	Name string `yaml:"name"`
}

type Identity struct {
	Style IDStyle `yaml:"style"`
}

type Vocab struct {
	Known  []string `yaml:"known"`
	Strict bool     `yaml:"strict"`
}

type Commit struct {
	Mode          CommitMode `yaml:"mode"`
	Trailer       string     `yaml:"trailer"`
	CloseTrailer  string     `yaml:"close_trailer"`
	SubjectPrefix string     `yaml:"subject_prefix"`
}

type Merge struct {
	Policy MergePolicy `yaml:"policy"`
}

type Sync struct {
	Push  bool      `yaml:"push"`
	Dirty SyncDirty `yaml:"dirty"`
}

type Workon struct {
	BranchPattern string `yaml:"branch_pattern"`
	Casing        Casing `yaml:"casing"`
	Worktree      bool   `yaml:"worktree"`
	WorktreeDir   string `yaml:"worktree_dir"`
}

type UI struct {
	Icons      IconMode          `yaml:"icons"`
	Background Background        `yaml:"background"`
	Color      ColorMode         `yaml:"color"`
	List       UIList            `yaml:"list"`
	Theme      map[string]string `yaml:"theme"`
	Tui        UITui             `yaml:"tui"`
	AutoTUI    bool              `yaml:"auto_tui"`
}

type UIList struct {
	Columns []string `yaml:"columns"`
}

type UITui struct {
	Split   float64 `yaml:"split"`
	Refresh string  `yaml:"refresh"`
}

type Git struct {
	ScanSince string `yaml:"scan_since,omitempty"`
	LandedRef string `yaml:"landed_ref,omitempty"`
}

type Estimate struct {
	Unit EstimateUnit `yaml:"unit"`
}

const SchemaVersion = 1

func (c *Config) VocabFor(field string) ([]string, bool) {
	switch field {
	case "priority":
		return c.Priorities.Values, true
	case "subtype":
		return c.Subtypes.Values, true
	case "resolution":
		return c.Resolutions.Values, true
	}
	return nil, false
}
