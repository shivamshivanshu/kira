// Package entityschema models entity structure — ticket, epic, board, and
// eventually user-defined types — as data (JSON schema files) rather than Go
// code. It depends only on datamodel and the standard library.
package entityschema

import (
	"slices"

	"github.com/shivamshivanshu/kira/internal/datamodel"
)

// FieldType is the closed set of typed primitives a FieldDef can declare.
// state and resolution are compositions over enum — state carries its value
// set inline (FieldDef.States) with a category per value; resolution names
// its vocabulary the same way a plain enum does (FieldDef.Enum).
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

// Placement says whether a field lives in the frontmatter or as a titled
// section in the Markdown body.
type Placement string

const (
	PlacementFrontmatter Placement = "frontmatter"
	PlacementBody        Placement = "body"
)

// Identity describes how a schema's instances are named/numbered. It is
// descriptive only in Phase 1 — nothing generates or enforces IDs from it yet.
type Identity struct {
	Style  string `json:"style,omitempty"`
	Prefix string `json:"prefix,omitempty"`
}

// EnumDef is a named, inline value set a schema file may declare directly.
// Named enums otherwise resolve against a config vocab supplied by the
// caller at validation time (see ResolveEnums).
type EnumDef struct {
	Name   string   `json:"name"`
	Values []string `json:"values"`
}

// Representation carries display concerns that are decoupled from the data
// contract: what a schema is called and how its instances list.
type Representation struct {
	Label       string   `json:"label,omitempty"`
	Icon        string   `json:"icon,omitempty"`
	ListColumns []string `json:"list_columns,omitempty"`
}

// StateValue is one value a state field may take, with the category
// (todo/doing/done) that drives board columns, done-detection, and
// resolution gating. State is a plain closed enum at the schema level: the
// transition graph between these values is a workflow concern (still owned
// by datamodel.Config/core, unchanged in Phase 1), not a schema constraint.
type StateValue struct {
	Key      string             `json:"key"`
	Category datamodel.Category `json:"category"`
}

// FieldDef declares one field of a Schema. Attribute meaning by Type:
//   - enum/resolution: Enum names the vocabulary, resolved via ResolveEnums.
//   - state: States lists the closed set of values (with category), inline.
//   - ref: Target names the referenced schema (or config-backed vocab) by name.
//   - Source, when set, is the config.yaml vocab this field's values are drawn
//     from (e.g. "priorities", "labels") — informational until a later phase
//     wires config-driven completion/editing.
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

// Schema is one entity type: ticket, epic, board, or a user-defined type
// layered in from .kira/schema/.
type Schema struct {
	Name           string         `json:"name"`
	Workflow       string         `json:"workflow,omitempty"`
	Identity       Identity       `json:"identity,omitempty"`
	Fields         []FieldDef     `json:"fields"`
	Enums          []EnumDef      `json:"enums,omitempty"`
	Representation Representation `json:"representation,omitempty"`
}

// Field looks up a field by name.
func (s Schema) Field(name string) (FieldDef, bool) {
	for _, f := range s.Fields {
		if f.Name == name {
			return f, true
		}
	}
	return FieldDef{}, false
}

// ResolveEnums merges a schema's inline enum definitions with externally
// supplied vocab (typically projected from datamodel.Config). Config vocab
// wins on a name conflict, so a built-in schema can describe an enum's shape
// while a repo's config.yaml supplies its actual values.
func ResolveEnums(schema Schema, configVocab map[string][]string) map[string][]string {
	out := make(map[string][]string, len(schema.Enums)+len(configVocab))
	for _, e := range schema.Enums {
		out[e.Name] = e.Values
	}
	for name, values := range configVocab {
		out[name] = values
	}
	return out
}
