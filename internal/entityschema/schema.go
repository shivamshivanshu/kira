// Package entityschema models entity structure (ticket, epic, board, and
// user-defined types) as data rather than Go code.
package entityschema

import (
	"maps"
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

// Identity says where a type's instances live. Only "config" is read, by
// ConfigBacked: board and sprint are config.yaml entries while ticket and epic
// are files. Display numbers come from project.key and boards[].key, never from
// here -- a prefix declared here was ignored, and wrong whenever project.key
// differed, so the field is gone.
type Identity struct {
	Style string `json:"style,omitempty"`
}

const IdentityStyleConfig = "config"

// ConfigBacked reports whether instances of this entity live as config.yaml
// entries (board, sprint) rather than as item files, which is what makes them
// absent from anything that reads, filters, or renders the item set.
func (s Schema) ConfigBacked() bool { return s.Identity.Style == IdentityStyleConfig }

type EnumDef struct {
	Name   string   `json:"name"`
	Values []string `json:"values"`
}

type Representation struct {
	Label       string   `json:"label,omitempty"`
	Icon        string   `json:"icon,omitempty"`
	ListColumns []string `json:"list_columns,omitempty"`
}

// TypeLabel is how a type names itself in output. A schema that declares no
// label is named by its type, which is the rule for every type -- kira holds no
// display name of its own.
func (s Schema) TypeLabel(typ string) string {
	if s.Representation.Label != "" {
		return s.Representation.Label
	}
	return typ
}

// StateValue pairs a state value with its category.
type StateValue struct {
	Key      string             `json:"key"`
	Category datamodel.Category `json:"category"`
}

// FieldDef declares one field. By Type: enum/resolution use Enum, state uses
// States, ref uses Target. Source names a config vocab for a config-backed ref
// (e.g. people, sprints); its presence routes ref validation to membership
// rather than entity existence.
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
	// TargetsContainer says the ref points at any type declaring the container
	// capability, rather than at the one type Target names. Naming a single type
	// cannot express "any container", and leaving Target empty would drop this
	// field out of TargetsItemType and silently stop reporting dangling refs.
	TargetsContainer bool      `json:"targets_container,omitempty"`
	Source           string    `json:"source,omitempty"`
	Placement        Placement `json:"placement,omitempty"`
	Section          string    `json:"section,omitempty"`
}

// TargetsItemType reports whether this field references another item. A type
// exists because config declares a workflow for it, not because it has a
// schema file, so the caller supplies the type set rather than this testing
// the schema map -- a ref to a config-only type is still a ref. A Source names
// a config vocabulary instead, so such a ref points at a vocabulary entry even
// when its Target happens to name an item type.
func (f FieldDef) TargetsItemType(itemTypes map[string]bool) bool {
	return f.Type == FieldRef && f.Source == "" && (itemTypes[f.Target] || f.TargetsContainer)
}

type Schema struct {
	Name           string         `json:"name"`
	Workflow       string         `json:"workflow,omitempty"`
	Identity       Identity       `json:"identity,omitempty"`
	Container      bool           `json:"container,omitempty"`
	Fields         []FieldDef     `json:"fields"`
	Enums          []EnumDef      `json:"enums,omitempty"`
	Representation Representation `json:"representation,omitempty"`
}

// IsContainer reports whether instances of this type may parent other items
// and roll up their progress. It is what "epic" used to mean by name: any type
// declaring it groups others, so nothing above needs to know which types those
// are.
func (s Schema) IsContainer() bool { return s.Container }

// ContainerTypes is the set of types declaring the container capability,
// shaped for the predicates that walk items without consulting schemas
// themselves.
func ContainerTypes(schemas map[string]Schema) map[string]bool {
	out := make(map[string]bool, len(schemas))
	for typ, schema := range schemas {
		if schema.IsContainer() {
			out[typ] = true
		}
	}
	return out
}

func (s Schema) Field(name string) (FieldDef, bool) {
	for _, f := range s.Fields {
		if f.Name == name {
			return f, true
		}
	}
	return FieldDef{}, false
}

// DeclaredExtra reports whether typ's schema declares name as a frontmatter
// extra (a non-builtin key promoted to a first-class field). A type with no
// schema declares nothing, so its extras stay unknown.
func DeclaredExtra(schemas map[string]Schema, typ, name string) (FieldDef, bool) {
	schema, ok := schemas[typ]
	if !ok {
		return FieldDef{}, false
	}
	f, ok := schema.Field(name)
	if !ok || !f.IsCustomFrontmatter() {
		return FieldDef{}, false
	}
	return f, true
}

// ResolveEnums merges inline enums with config vocab; config wins on a name
// conflict.
func ResolveEnums(schema Schema, configVocab map[string][]string) map[string][]string {
	out := make(map[string][]string, len(schema.Enums)+len(configVocab))
	for _, e := range schema.Enums {
		out[e.Name] = slices.Clone(e.Values)
	}
	maps.Copy(out, configVocab)
	return out
}
