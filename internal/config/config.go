// Package config models kira's `.kira/config.yaml` — the single tracked source
// of workflow, vocabulary, commit-mode, and merge-policy truth (see
// docs/design/02-data-model.md §9). It parses and validates that file, fills
// documented defaults for omitted fields, and exposes Default() for `kira init`.
// Its only internal dependency is the item codec, for the shared field formats
// (date layout) config values must match.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// IDStyle selects how the human-facing `number` is derived.
type IDStyle string

// ID styles: sequential allocates KIRA-n and is reconciled post-merge; hash
// derives the display ID from the ULID and needs no reconciliation.
const (
	IDStyleSequential IDStyle = "sequential"
	IDStyleHash       IDStyle = "hash"
)

var idStyles = []IDStyle{IDStyleSequential, IDStyleHash}

// CommitMode controls when a mutating command commits its writes.
type CommitMode string

// Commit modes: auto commits immediately, manual only stages, prompt asks first.
const (
	CommitAuto   CommitMode = "auto"
	CommitManual CommitMode = "manual"
	CommitPrompt CommitMode = "prompt"
)

var commitModes = []CommitMode{CommitAuto, CommitManual, CommitPrompt}

// MergePolicy selects how same-field merge conflicts on kira files are resolved.
type MergePolicy string

// Merge policies: auto applies the field-level three-way engine, manual surfaces
// an ordinary git conflict for a human to resolve.
const (
	MergeAuto   MergePolicy = "auto"
	MergeManual MergePolicy = "manual"
)

var mergePolicies = []MergePolicy{MergeAuto, MergeManual}

// IconMode controls glyph rendering in the TUI/nvim visual layers.
type IconMode string

// Icon modes: auto detects terminal capability, always/never force the choice.
const (
	IconAuto   IconMode = "auto"
	IconAlways IconMode = "always"
	IconNever  IconMode = "never"
)

var iconModes = []IconMode{IconAuto, IconAlways, IconNever}

// EstimateUnit is the unit of the item `estimate` field.
type EstimateUnit string

// Estimate units: points (unitless) or hours (enables estimate-vs-actual stats).
const (
	EstimatePoints EstimateUnit = "points"
	EstimateHours  EstimateUnit = "hours"
)

var estimateUnits = []EstimateUnit{EstimatePoints, EstimateHours}

// Category is the stable, config-independent class of a workflow state; telemetry
// keys off this, never off state-name strings (docs/design/02-data-model.md §6).
type Category string

// The three categories every state must declare.
const (
	CategoryTodo  Category = "todo"
	CategoryDoing Category = "doing"
	CategoryDone  Category = "done"
)

var categories = []Category{CategoryTodo, CategoryDoing, CategoryDone}

// Config is the parsed, defaulted, validated `.kira/config.yaml`.
type Config struct {
	Version     int                 `yaml:"version"`
	Project     Project             `yaml:"project"`
	ID          ID                  `yaml:"id"`
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

// Project identifies the tracker: Key prefixes sequential numbers, Name is display-only.
type Project struct {
	Key  string `yaml:"key"`
	Name string `yaml:"name"`
}

// ID holds identity-scheme settings.
type ID struct {
	Style IDStyle `yaml:"style"`
}

// State is one node in a workflow. Resolution is optional and only meaningful on
// a done-category state that is not a real completion (e.g. WONT_DO -> dropped).
// Wip is an advisory per-state item cap; 0 means unlimited.
type State struct {
	Key        string   `yaml:"key"`
	Category   Category `yaml:"category"`
	Wip        int      `yaml:"wip,omitempty"`
	Resolution string   `yaml:"resolution,omitempty"`
}

// Workflow is the state machine for one item type. Transitions is an adjacency map
// from state key to the transitions reachable in one move; EnforceTransitions
// makes off-graph moves fail unless forced.
type Workflow struct {
	States             []State                 `yaml:"states"`
	Initial            string                  `yaml:"initial"`
	Transitions        map[string][]Transition `yaml:"transitions"`
	EnforceTransitions bool                    `yaml:"enforce_transitions"`
}

// Transition is one edge of a workflow's adjacency map. A bare state key in the
// YAML (`REVIEW: [DONE]`) decodes to just To; the guard-map form
// (`{ to: DONE, require: [resolution], set: { resolution: done } }`) adds
// Require (fields that must be non-null before the move) and Set (field
// assignments applied on the move) — docs/design/02-data-model.md §6.
type Transition struct {
	To      string            `yaml:"to"`
	Require []string          `yaml:"require,omitempty"`
	Set     map[string]string `yaml:"set,omitempty"`
}

// UnmarshalYAML accepts both documented transition forms: a bare target state
// string, or the guard map.
func (t *Transition) UnmarshalYAML(n *yaml.Node) error {
	if n.Kind == yaml.ScalarNode {
		t.To = n.Value
		return nil
	}
	type raw Transition // shed the method to avoid recursion
	return n.Decode((*raw)(t))
}

// TransitionsTo builds a bare (guard-free) transition list from state keys —
// the shape Default() and tests use.
func TransitionsTo(states ...string) []Transition {
	ts := make([]Transition, len(states))
	for i, s := range states {
		ts[i] = Transition{To: s}
	}
	return ts
}

// Sprint is one scrum sprint entity; the item `sprint` field keys into this
// list. Start/End are RFC3339 dates.
type Sprint struct {
	Key   string `yaml:"key"`
	Name  string `yaml:"name"`
	Start string `yaml:"start"`
	End   string `yaml:"end"`
}

// Vocab is a controlled vocabulary. When Strict, unknown values are rejected
// (overridable per-write with --force); otherwise they only warn.
type Vocab struct {
	Known  []string `yaml:"known"`
	Strict bool     `yaml:"strict"`
}

// Commit holds commit-mode settings. Trailer is the commit-message trailer key
// used to link commits to items.
type Commit struct {
	Mode    CommitMode `yaml:"mode"`
	Trailer string     `yaml:"trailer"`
}

// Merge holds the merge-conflict policy.
type Merge struct {
	Policy MergePolicy `yaml:"policy"`
}

// Sync holds sync settings. Push makes a no-flag `kira sync` also publish after a
// clean doctor pass.
type Sync struct {
	Push bool `yaml:"push"`
}

// UI holds presentation settings for the frontends.
type UI struct {
	Icons IconMode `yaml:"icons"`
}

// Git holds git-integration settings. ScanSince (proposed) bounds the first
// trailer scan on a large pre-existing history to a date or ref.
type Git struct {
	ScanSince string `yaml:"scan_since,omitempty"`
}

// Estimate holds estimation settings. HoursPerDay converts cycle-time days to
// calendar hours for estimate-vs-actual stats when Unit is hours.
type Estimate struct {
	Unit        EstimateUnit `yaml:"unit"`
	HoursPerDay float64      `yaml:"hours_per_day"`
}

// SchemaVersion is the only config schema version this build understands.
const SchemaVersion = 1

// Default returns the documented default configuration (docs/design/02-data-model.md §9),
// used both as the baseline for filling omitted fields and as the config `kira init` writes.
// Filters and Sprints deviate from the §9 example deliberately — see the inline note.
func Default() *Config {
	return &Config{
		Version: SchemaVersion,
		Project: Project{Key: "KIRA", Name: "kira"},
		ID:      ID{Style: IDStyleSequential},
		Workflows: map[string]Workflow{
			"ticket": {
				States: []State{
					{Key: "TODO", Category: CategoryTodo},
					{Key: "IN_PROGRESS", Category: CategoryDoing, Wip: 3},
					{Key: "REVIEW", Category: CategoryDoing, Wip: 2},
					{Key: "DONE", Category: CategoryDone},
					{Key: "WONT_DO", Category: CategoryDone, Resolution: "dropped"},
				},
				Initial: "TODO",
				Transitions: map[string][]Transition{
					"TODO":        TransitionsTo("IN_PROGRESS", "WONT_DO"),
					"IN_PROGRESS": TransitionsTo("REVIEW", "TODO", "WONT_DO"),
					"REVIEW": {
						{To: "DONE", Require: []string{"resolution"}, Set: map[string]string{"resolution": "done"}},
						{To: "IN_PROGRESS"},
					},
					"DONE":    {},
					"WONT_DO": {},
				},
				EnforceTransitions: true,
			},
			"epic": {
				States: []State{
					{Key: "PLANNED", Category: CategoryTodo},
					{Key: "ACTIVE", Category: CategoryDoing},
					{Key: "DONE", Category: CategoryDone},
				},
				Initial: "PLANNED",
				Transitions: map[string][]Transition{
					"PLANNED": TransitionsTo("ACTIVE"),
					"ACTIVE":  TransitionsTo("DONE"),
					"DONE":    {},
				},
			},
		},
		Labels:      Vocab{Known: []string{"bug", "feature", "perf", "tech-debt", "orderbook", "infra", "p0", "p1", "p2"}},
		People:      Vocab{Known: []string{"shivam", "alice"}},
		Priorities:  []string{"P0", "P1", "P2", "P3"},
		Subtypes:    []string{"bug", "story", "task", "spike"},
		Resolutions: []string{"done", "dropped", "duplicate", "cannot-reproduce"},
		// Filters and Sprints default empty: the doc example's entries are
		// illustrations (dated queries, sample sprints), not defaults — a
		// project authors filters in config and sprints via `kira sprint`.
		Filters: map[string]string{},
		Sprints: nil,
		Commit:  Commit{Mode: CommitAuto, Trailer: "Kira-Ticket"},
		Merge:   Merge{Policy: MergeAuto},
		Sync:    Sync{Push: false},
		UI:      UI{Icons: IconAuto},
		Git:     Git{},
		Estimate: Estimate{
			Unit:        EstimatePoints,
			HoursPerDay: 8,
		},
		Fields: map[string]any{},
	}
}

// Load reads and parses `<root>/.kira/config.yaml`.
func Load(root string) (*Config, error) {
	path := filepath.Join(root, ".kira", "config.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("config: reading %s: %w", path, err)
	}
	return Parse(data)
}

// Parse decodes config YAML over the documented defaults, then validates the
// result. Omitted fields keep their default; an invalid value is rejected with
// an error naming the offending key — never silently defaulted.
func Parse(data []byte) (*Config, error) {
	cfg := Default()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}
