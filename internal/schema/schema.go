// Package schema generates a JSON Schema for kira's --json output shapes and
// its automation hook-stdin payload, keeping the datamodel result structs and
// automation.HookPayload the single source of truth.
package schema

import (
	"bytes"
	"encoding/json"
	"path"
	"reflect"
	"strings"

	"github.com/shivamshivanshu/kira/internal/automation"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/doctor"
	"github.com/shivamshivanshu/kira/internal/syncx"
)

//go:generate go test . -run TestArtifactFresh -update

func topLevelTypes() []reflect.Type {
	return []reflect.Type{
		reflect.TypeFor[datamodel.CreateResult](),
		reflect.TypeFor[datamodel.ListResult](),
		reflect.TypeFor[datamodel.TreeResult](),
		reflect.TypeFor[datamodel.ShowResult](),
		reflect.TypeFor[datamodel.MutationResult](),
		reflect.TypeFor[datamodel.BulkOutcome](),
		reflect.TypeFor[datamodel.DiffResult](),
		reflect.TypeFor[datamodel.ChangesResult](),
		reflect.TypeFor[datamodel.FindResult](),
		reflect.TypeFor[datamodel.InitResult](),
		reflect.TypeFor[datamodel.ConfigSetResult](),
		reflect.TypeFor[datamodel.ConfigInitResult](),
		reflect.TypeFor[datamodel.LabelCreateResult](),
		reflect.TypeFor[datamodel.LabelListResult](),
		reflect.TypeFor[datamodel.MergeResult](),
		reflect.TypeFor[datamodel.ResolveResult](),
		reflect.TypeFor[datamodel.MoveResult](),
		reflect.TypeFor[datamodel.CommentResult](),
		reflect.TypeFor[datamodel.FilterListResult](),
		reflect.TypeFor[datamodel.SprintCreateResult](),
		reflect.TypeFor[datamodel.SprintListResult](),
		reflect.TypeFor[datamodel.SprintActivateResult](),
		reflect.TypeFor[datamodel.SprintCloseResult](),
		reflect.TypeFor[datamodel.BoardCreateResult](),
		reflect.TypeFor[datamodel.BoardListResult](),
		reflect.TypeFor[datamodel.BoardUpdateResult](),
		reflect.TypeFor[datamodel.BoardMoveResult](),
		reflect.TypeFor[datamodel.IndexResult](),
		reflect.TypeFor[datamodel.StatsResult](),
		reflect.TypeFor[datamodel.BlameResult](),
		reflect.TypeFor[datamodel.LogResult](),
		reflect.TypeFor[datamodel.BoardResult](),
		reflect.TypeFor[datamodel.VersionResult](),
		reflect.TypeFor[datamodel.NowResult](),
		reflect.TypeFor[datamodel.CommitResult](),
		reflect.TypeFor[datamodel.HooksStatusResult](),
		reflect.TypeFor[datamodel.HooksUninstallResult](),
		reflect.TypeFor[datamodel.HooksInstallResult](),
		reflect.TypeFor[datamodel.HooksValidateResult](),
		reflect.TypeFor[datamodel.WorkonResult](),
		reflect.TypeFor[datamodel.AutomationListResult](),
		reflect.TypeFor[datamodel.AutomationTrustResult](),
		reflect.TypeFor[datamodel.CreateTemplateResult](),
		reflect.TypeFor[datamodel.ReconcileResult](),
		reflect.TypeFor[datamodel.ExplainResult](),
		// syncx.Report and doctor.Report both have bare name "Report"; list
		// syncx.Report first so it keeps the unqualified "Report" $defs key
		// (it was already reachable, and named that, via HookPayload.Sync
		// before doctor.Report existed in this schema) and doctor.Report
		// gets qualified to "DoctorReport" by defName's collision handling.
		reflect.TypeFor[syncx.Report](),
		reflect.TypeFor[doctor.Report](),
		reflect.TypeFor[automation.HookPayload](),
	}
}

// enumValues renders a named-string inventory slice (e.g. datamodel.WarnCodes)
// as the plain strings a JSON Schema "enum" constraint expects.
func enumValues[T ~string](vs []T) []string {
	out := make([]string, len(vs))
	for i, v := range vs {
		out[i] = string(v)
	}
	return out
}

// stringEnums maps each named string type that already has an exported
// inventory slice to its allowed values, so the generator can emit an
// "enum" constraint instead of a bare "type": "string".
var stringEnums = map[reflect.Type][]string{
	reflect.TypeFor[datamodel.WarnCode]():        enumValues(datamodel.WarnCodes),
	reflect.TypeFor[datamodel.DiffStatus]():      enumValues(datamodel.DiffStatuses),
	reflect.TypeFor[datamodel.EventName]():       enumValues(datamodel.AutomationEvents),
	reflect.TypeFor[datamodel.IDStyle]():         enumValues(datamodel.IDStyles),
	reflect.TypeFor[datamodel.CommitMode]():      enumValues(datamodel.CommitModes),
	reflect.TypeFor[datamodel.LinkMarker]():      enumValues(datamodel.LinkMarkers),
	reflect.TypeFor[datamodel.ReferenceMarker](): enumValues(datamodel.ReferenceMarkers),
	reflect.TypeFor[datamodel.MergePolicy]():     enumValues(datamodel.MergePolicies),
	reflect.TypeFor[datamodel.IconMode]():        enumValues(datamodel.IconModes),
	reflect.TypeFor[datamodel.Background]():      enumValues(datamodel.Backgrounds),
	reflect.TypeFor[datamodel.ColorMode]():       enumValues(datamodel.ColorModes),
	reflect.TypeFor[datamodel.EstimateUnit]():    enumValues(datamodel.EstimateUnits),
	reflect.TypeFor[datamodel.Casing]():          enumValues(datamodel.Casings),
	reflect.TypeFor[datamodel.SyncDirty]():       enumValues(datamodel.SyncDirties),
	reflect.TypeFor[datamodel.WipPolicy]():       enumValues(datamodel.WipPolicies),
	reflect.TypeFor[datamodel.Category]():        enumValues(datamodel.Categories),
}

// generator builds the $defs graph for one Generate() call. defs holds the
// rendered JSON Schema node per def name; names/owners resolve def names
// from reflect.Type so that two distinct types sharing a bare Name() (e.g.
// doctor.Report and syncx.Report) don't silently collide.
type generator struct {
	defs   map[string]any
	names  map[reflect.Type]string
	owners map[string]reflect.Type
}

func newGenerator() *generator {
	return &generator{
		defs:   map[string]any{},
		names:  map[reflect.Type]string{},
		owners: map[string]reflect.Type{},
	}
}

func Generate() ([]byte, error) {
	g := newGenerator()
	types := topLevelTypes()
	oneOf := make([]any, 0, len(types)+1)
	for _, t := range types {
		oneOf = append(oneOf, map[string]any{"$ref": "#/$defs/" + g.register(t)})
	}
	oneOf = append(oneOf, map[string]any{
		"type":  "array",
		"items": map[string]any{"$ref": "#/$defs/" + g.register(reflect.TypeFor[datamodel.BulkOutcome]())},
	})
	doc := map[string]any{
		"$schema":     "https://json-schema.org/draft/2020-12/schema",
		"title":       "kira --json output and automation payload",
		"description": "Machine-readable shapes for kira commands run with --json and for the automation hook-stdin payload.",
		"oneOf":       oneOf,
		"$defs":       g.defs,
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(doc); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// claimedByOther reports whether name is already the $defs key of some type
// other than t.
func (g *generator) claimedByOther(name string, t reflect.Type) bool {
	owner, ok := g.owners[name]
	return ok && owner != t
}

// defName returns the $defs key for t, reusing t's bare name unless another
// distinct type already claimed it — in which case it qualifies the name
// with the last segment of t's package path. A clash that survives
// qualification means two distinct types share both name and package,
// which reflect.Type identity rules out; it can only mean a bug in this
// function, so it panics rather than silently mis-registering a $ref.
func (g *generator) defName(t reflect.Type) string {
	name := t.Name()
	if g.claimedByOther(name, t) {
		name = capitalize(path.Base(t.PkgPath())) + t.Name()
		if g.claimedByOther(name, t) {
			panic("schema: unresolvable $defs name collision for " + t.String())
		}
	}
	g.owners[name] = t
	return name
}

func (g *generator) register(t reflect.Type) string {
	if name, ok := g.names[t]; ok {
		return name
	}
	name := g.defName(t)
	g.names[t] = name
	g.defs[name] = map[string]any{} // reserve the slot before recursing, in case of cycles
	props := map[string]any{}
	required := []string{}
	g.addFields(t, props, &required)
	obj := map[string]any{"type": "object", "properties": props, "additionalProperties": false}
	if len(required) > 0 {
		obj["required"] = required
	}
	g.defs[name] = obj
	return name
}

func (g *generator) addFields(t reflect.Type, props map[string]any, required *[]string) {
	for i := range t.NumField() {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		name, opts, _ := strings.Cut(f.Tag.Get("json"), ",")
		if name == "-" {
			continue
		}
		if f.Anonymous && name == "" {
			embedded := f.Type
			if embedded.Kind() == reflect.Pointer {
				embedded = embedded.Elem()
			}
			g.addFields(embedded, props, required)
			continue
		}
		if name == "" {
			name = f.Name
		}
		props[name] = g.schemaForType(f.Type)
		if !strings.Contains(opts, "omitempty") {
			*required = append(*required, name)
		}
	}
}

func (g *generator) schemaForType(t reflect.Type) map[string]any {
	if values, ok := stringEnums[t]; ok {
		return map[string]any{"type": "string", "enum": values}
	}
	switch t.Kind() {
	case reflect.String:
		return map[string]any{"type": "string"}
	case reflect.Bool:
		return map[string]any{"type": "boolean"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return map[string]any{"type": "integer"}
	case reflect.Float32, reflect.Float64:
		return map[string]any{"type": "number"}
	case reflect.Pointer:
		return nullable(g.schemaForType(t.Elem()))
	case reflect.Slice, reflect.Array:
		return map[string]any{"type": "array", "items": g.schemaForType(t.Elem())}
	case reflect.Map:
		return map[string]any{"type": "object", "additionalProperties": g.schemaForType(t.Elem())}
	case reflect.Struct:
		name := g.register(t)
		return map[string]any{"$ref": "#/$defs/" + name}
	default:
		return map[string]any{}
	}
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func nullable(node map[string]any) map[string]any {
	if ref, ok := node["$ref"]; ok {
		return map[string]any{"anyOf": []any{
			map[string]any{"$ref": ref},
			map[string]any{"type": "null"},
		}}
	}
	if tp, ok := node["type"].(string); ok {
		node["type"] = []any{tp, "null"}
	}
	return node
}
