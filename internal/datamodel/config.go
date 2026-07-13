package datamodel

type Config struct {
	Version     int                 `yaml:"version"`
	Project     Project             `yaml:"project"`
	ID          Identity            `yaml:"id"`
	Workflows   map[string]Workflow `yaml:"workflows"`
	Labels      Vocab               `yaml:"labels"`
	People      Vocab               `yaml:"people"`
	Priorities  []string            `yaml:"priorities"`
	Subtypes    []string            `yaml:"subtypes"`
	Resolutions []string            `yaml:"resolutions"`
	Filters     map[string]string   `yaml:"filters"`
	Sprints     []Sprint            `yaml:"sprints"`
	Commit      Commit              `yaml:"commit"`
	Merge       Merge               `yaml:"merge"`
	Sync        Sync                `yaml:"sync"`
	UI          UI                  `yaml:"ui"`
	Git         Git                 `yaml:"git"`
	Estimate    Estimate            `yaml:"estimate"`
	Fields      map[string]any      `yaml:"fields"`
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
	Mode         CommitMode `yaml:"mode"`
	Trailer      string     `yaml:"trailer"`
	CloseTrailer string     `yaml:"close_trailer"`
}

type Merge struct {
	Policy MergePolicy `yaml:"policy"`
}

type Sync struct {
	Push bool `yaml:"push"`
}

type UI struct {
	Icons IconMode `yaml:"icons"`
}

type Git struct {
	ScanSince string `yaml:"scan_since,omitempty"`
	LandedRef string `yaml:"landed_ref,omitempty"`
}

type Estimate struct {
	Unit        EstimateUnit `yaml:"unit"`
	HoursPerDay float64      `yaml:"hours_per_day"`
}

const SchemaVersion = 1

func (c *Config) VocabFor(field string) ([]string, bool) {
	switch field {
	case "priority":
		return c.Priorities, true
	case "subtype":
		return c.Subtypes, true
	case "resolution":
		return c.Resolutions, true
	}
	return nil, false
}
