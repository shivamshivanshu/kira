package datamodel

import "strings"

type Config struct {
	Version int      `yaml:"version"`
	Project Project  `yaml:"project"`
	ID      Identity `yaml:"id"`
	Labels  Vocab    `yaml:"labels"`
	People  People   `yaml:"people"`

	Filters  map[string]string `yaml:"filters"`
	Boards   []Board           `yaml:"boards,omitempty"`
	Sprints  []Sprint          `yaml:"sprints"`
	Commit   Commit            `yaml:"commit"`
	Merge    Merge             `yaml:"merge"`
	Sync     Sync              `yaml:"sync"`
	Workon   Workon            `yaml:"workon"`
	UI       UI                `yaml:"ui"`
	Git      Git               `yaml:"git"`
	Estimate Estimate          `yaml:"estimate"`

	UserCommitSubject string     `yaml:"-"`
	UIProvenance      Provenance `yaml:"-"`
	WorkonProvenance  Provenance `yaml:"-"`
}

type Project struct {
	Key  string `yaml:"key"`
	Name string `yaml:"name"`
}

type Board struct {
	Key         string `yaml:"key"`
	Name        string `yaml:"name"`
	Description string `yaml:"description,omitempty"`
	Default     bool   `yaml:"default,omitempty"`
	Archived    bool   `yaml:"archived,omitempty"`
}

type Identity struct {
	Style IDStyle `yaml:"style"`
}

type Vocab struct {
	Known  []string `yaml:"known"`
	Strict bool     `yaml:"strict"`
}

type Commit struct {
	Mode             CommitMode        `yaml:"mode"`
	Trailer          string            `yaml:"trailer"`
	CloseTrailer     string            `yaml:"close_trailer"`
	SubjectPrefix    string            `yaml:"subject_prefix"`
	LinkMarkers      []LinkMarker      `yaml:"link_markers"`
	ReferenceMarkers []ReferenceMarker `yaml:"reference_markers"`
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
	Worktree      bool   `yaml:"worktree"`
	WorktreeDir   string `yaml:"worktree_dir"`
}

type UI struct {
	Icons      IconMode          `yaml:"icons"`
	Background Background        `yaml:"background"`
	Color      ColorMode         `yaml:"color"`
	Editor     string            `yaml:"editor"`
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
	LandedRef string `yaml:"landed_ref,omitempty"`
}

type Estimate struct {
	Unit EstimateUnit `yaml:"unit"`
}

const (
	InitialSchemaVersion = 1
	BoardsSchemaVersion  = 2
	SchemaVersion        = BoardsSchemaVersion
)

func (c *Config) EffectiveBoards() []Board {
	if c.Boards != nil {
		return c.Boards
	}
	if c.Project.Key == "" {
		return nil
	}
	name := c.Project.Name
	if name == "" {
		name = c.Project.Key
	}
	return []Board{{Key: c.Project.Key, Name: name, Default: true}}
}

func (c *Config) DefaultBoard() (Board, bool) {
	for _, b := range c.EffectiveBoards() {
		if b.Default && !b.Archived {
			return b, true
		}
	}
	return Board{}, false
}

func (c *Config) BoardByKey(key string) (Board, bool) {
	for _, b := range c.EffectiveBoards() {
		if strings.EqualFold(b.Key, key) {
			return b, true
		}
	}
	return Board{}, false
}

func (c *Config) ActiveBoards() []Board {
	all := c.EffectiveBoards()
	out := make([]Board, 0, len(all))
	for _, b := range all {
		if !b.Archived {
			out = append(out, b)
		}
	}
	return out
}

func (c *Config) BoardKeys() []string {
	all := c.EffectiveBoards()
	keys := make([]string, len(all))
	for i, b := range all {
		keys[i] = b.Key
	}
	return keys
}
