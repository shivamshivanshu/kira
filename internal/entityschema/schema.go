// Package entityschema models entity structure (ticket, epic, board, and
// user-defined types) as data rather than Go code.
package entityschema

import (
	"slices"

	"github.com/shivamshivanshu/kira/internal/datamodel"
)

type FieldType string

const (
	FieldString     FieldType = "string"
	FieldMarkdown   FieldType = "markdown"
	FieldInt        FieldType = "int"
	FieldNumber     FieldType = "number"
	FieldBool       FieldType = "bool"
	FieldDate       FieldType = "date"
	FieldDatetime   FieldType = "datetime"
	FieldEnum       FieldType = "enum"
	FieldRef        FieldType = "ref"
	FieldState      FieldType = "state"
	FieldResolution FieldType = "resolution"
)

var fieldTypes = []FieldType{
	FieldString, FieldMarkdown, FieldInt, FieldNumber, FieldBool,
	FieldDate, FieldDatetime, FieldEnum, FieldRef, FieldState, FieldResolution,
}

func (t FieldType) Valid() bool { return slices.Contains(fieldTypes, t) }

type Placement string

const (
	PlacementFrontmatter Placement = "frontmatter"
	PlacementBody        Placement = "body"
)

// Identity is descriptive only in phase 1 — nothing generates IDs from it yet.
type Identity struct {
	Style  string `json:"style,omitempty"`
	Prefix string `json:"prefix,omitempty"`
}

type EnumDef struct {
	Name   string   `json:"name"`
	Values []string `json:"values"`
}

type Representation struct {
	Label       string   `json:"label,omitempty"`
	Icon        string   `json:"icon,omitempty"`
	ListColumns []string `json:"list_columns,omitempty"`
}

// StateValue pairs a state value with its category; the transition graph
// between states is a workflow concern, not a schema constraint.
type StateValue struct {
	Key      string             `json:"key"`
	Category datamodel.Category `json:"category"`
}

// FieldDef declares one field. By Type: enum/resolution use Enum, state uses
// States, ref uses Target. Source (config vocab) is informational in phase 1.
type FieldDef struct {
	Name      string       `json:"name"`
	Type      FieldType    `json:"type"`
	List      bool         `json:"list,omitempty"`
	Unique    bool         `json:"unique,omitempty"`
	Required  bool         `json:"required,omitempty"`
	Immutable bool         `json:"immutable,omitempty"`
	Guarded   bool         `json:"guarded,omitempty"`
	System    bool         `json:"system,omitempty"`
	Enum      string       `json:"enum,omitempty"`
	States    []StateValue `json:"states,omitempty"`
	Target    string       `json:"target,omitempty"`
	Source    string       `json:"source,omitempty"`
	Placement Placement    `json:"placement,omitempty"`
	Section   string       `json:"section,omitempty"`
}

type Schema struct {
	Name           string         `json:"name"`
	Workflow       string         `json:"workflow,omitempty"`
	Identity       Identity       `json:"identity,omitempty"`
	Fields         []FieldDef     `json:"fields"`
	Enums          []EnumDef      `json:"enums,omitempty"`
	Representation Representation `json:"representation,omitempty"`
}

func (s Schema) Field(name string) (FieldDef, bool) {
	for _, f := range s.Fields {
		if f.Name == name {
			return f, true
		}
	}
	return FieldDef{}, false
}

// ResolveEnums merges inline enums with config vocab; config wins on a name
// conflict.
func ResolveEnums(schema Schema, configVocab map[string][]string) map[string][]string {
	out := make(map[string][]string, len(schema.Enums)+len(configVocab))
	for _, e := range schema.Enums {
		out[e.Name] = slices.Clone(e.Values)
	}
	for name, values := range configVocab {
		out[name] = values
	}
	return out
}
