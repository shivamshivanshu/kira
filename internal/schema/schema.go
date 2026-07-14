// Package schema generates a JSON Schema for kira's --json output shapes,
// keeping the datamodel result structs the single source of truth.
package schema

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"

	"github.com/shivamshivanshu/kira/internal/datamodel"
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
	}
}

func Generate() ([]byte, error) {
	defs := map[string]any{}
	for _, t := range topLevelTypes() {
		register(t, defs)
	}
	doc := map[string]any{
		"$schema":     "https://json-schema.org/draft/2020-12/schema",
		"title":       "kira --json output",
		"description": "Machine-readable output shapes for kira commands run with --json.",
		"$defs":       defs,
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

func register(t reflect.Type, defs map[string]any) {
	if _, seen := defs[t.Name()]; seen {
		return
	}
	defs[t.Name()] = map[string]any{}
	props := map[string]any{}
	required := []string{}
	addFields(t, props, &required, defs)
	obj := map[string]any{"type": "object", "properties": props, "additionalProperties": false}
	if len(required) > 0 {
		obj["required"] = required
	}
	defs[t.Name()] = obj
}

func addFields(t reflect.Type, props map[string]any, required *[]string, defs map[string]any) {
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
			addFields(f.Type, props, required, defs)
			continue
		}
		if name == "" {
			name = f.Name
		}
		props[name] = schemaForType(f.Type, defs)
		if !strings.Contains(opts, "omitempty") && f.Type.Kind() != reflect.Pointer {
			*required = append(*required, name)
		}
	}
}

func schemaForType(t reflect.Type, defs map[string]any) map[string]any {
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
		return nullable(schemaForType(t.Elem(), defs))
	case reflect.Slice, reflect.Array:
		return map[string]any{"type": "array", "items": schemaForType(t.Elem(), defs)}
	case reflect.Map:
		return map[string]any{"type": "object", "additionalProperties": schemaForType(t.Elem(), defs)}
	case reflect.Struct:
		register(t, defs)
		return map[string]any{"$ref": "#/$defs/" + t.Name()}
	default:
		return map[string]any{}
	}
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
