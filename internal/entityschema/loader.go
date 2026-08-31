package entityschema

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/shivamshivanshu/kira/internal/datamodel"
)

//go:embed builtin/*.json
var builtinFiles embed.FS

const builtinDir = "builtin"

// Load layers user .kira/schema/*.json over the embedded built-ins, overriding
// by name; a missing dir yields the built-ins alone.
func Load(dir string) (map[string]Schema, error) {
	schemas, err := loadBuiltins()
	if err != nil {
		return nil, err
	}
	if err := overlayUserSchemas(schemas, dir); err != nil {
		return nil, err
	}
	if err := checkExtrasAgreement(schemas); err != nil {
		return nil, err
	}
	return schemas, nil
}

func overlayUserSchemas(schemas map[string]Schema, dir string) error {
	if dir == "" {
		return nil
	}
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("reading %s: %w", dir, err)
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return fmt.Errorf("reading %s: %w", e.Name(), err)
		}
		schema, err := parseSchema(e.Name(), data)
		if err != nil {
			return err
		}
		schemas[schema.Name] = schema
	}
	return nil
}

// checkExtrasAgreement rejects two schemas declaring the same frontmatter
// extra with different traits: the query layer builds one spec per field name
// across all schemas, so a disagreement would let one schema's traits
// silently win for every type.
func checkExtrasAgreement(schemas map[string]Schema) error {
	type origin struct {
		schema string
		field  FieldDef
	}
	first := map[string]origin{}
	for _, name := range slices.Sorted(maps.Keys(schemas)) {
		for _, f := range schemas[name].Fields {
			if f.Placement != PlacementFrontmatter {
				continue
			}
			prev, ok := first[f.Name]
			if !ok {
				first[f.Name] = origin{schema: name, field: f}
				continue
			}
			if prev.field.Type != f.Type || prev.field.List != f.List {
				return fmt.Errorf("field %q is declared as %s in schema %q but as %s in schema %q; schemas sharing a field must agree on its type and list-ness",
					f.Name, fieldTrait(prev.field), prev.schema, fieldTrait(f), name)
			}
		}
	}
	return nil
}

func fieldTrait(f FieldDef) string {
	if f.List {
		return "list of " + string(f.Type)
	}
	return string(f.Type)
}

func BuiltinNames() ([]string, error) {
	schemas, err := loadBuiltins()
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(schemas))
	for name := range schemas {
		names = append(names, name)
	}
	sort.Strings(names)
	return names, nil
}

// WriteDefaults materializes the embedded built-in schemas into dir, never
// overwriting an existing file so re-running it never clobbers a user edit.
func WriteDefaults(dir string) error {
	entries, err := builtinFiles.ReadDir(builtinDir)
	if err != nil {
		return fmt.Errorf("reading embedded schemas: %w", err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating %s: %w", dir, err)
	}
	for _, e := range entries {
		dst := filepath.Join(dir, e.Name())
		if _, err := os.Stat(dst); err == nil {
			continue // don't clobber a user-edited schema
		} else if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("checking %s: %w", dst, err)
		}
		data, err := builtinFiles.ReadFile(filepath.Join(builtinDir, e.Name()))
		if err != nil {
			return fmt.Errorf("reading embedded %s: %w", e.Name(), err)
		}
		if err := os.WriteFile(dst, data, 0o644); err != nil {
			return fmt.Errorf("writing %s: %w", dst, err)
		}
	}
	return nil
}

func loadBuiltins() (map[string]Schema, error) {
	entries, err := builtinFiles.ReadDir(builtinDir)
	if err != nil {
		return nil, fmt.Errorf("reading embedded schemas: %w", err)
	}
	schemas := make(map[string]Schema, len(entries))
	for _, e := range entries {
		data, err := builtinFiles.ReadFile(filepath.Join(builtinDir, e.Name()))
		if err != nil {
			return nil, fmt.Errorf("reading embedded %s: %w", e.Name(), err)
		}
		schema, err := parseSchema(e.Name(), data)
		if err != nil {
			return nil, err
		}
		schemas[schema.Name] = schema
	}
	return schemas, nil
}

// parseSchema deliberately tolerates unknown JSON keys: kira init materializes
// the builtin schemas into .kira/schema/, so files on disk may carry keys a
// later kira dropped (e.g. "identity", removed as inert in KIRA-141) and must
// keep loading.
func parseSchema(filename string, data []byte) (Schema, error) {
	var s Schema
	if err := json.Unmarshal(data, &s); err != nil {
		return Schema{}, fmt.Errorf("%s: %w", filename, err)
	}
	if err := checkSchemaShape(s); err != nil {
		return Schema{}, fmt.Errorf("%s: %w", filename, err)
	}
	return s, nil
}

func checkSchemaShape(s Schema) error {
	if s.Name == "" {
		return errors.New(`schema missing "name"`)
	}
	seen := make(map[string]bool, len(s.Fields))
	for _, f := range s.Fields {
		if f.Name == "" {
			return fmt.Errorf("schema %q: field missing \"name\"", s.Name)
		}
		if seen[f.Name] {
			return fmt.Errorf("schema %q: duplicate field %q", s.Name, f.Name)
		}
		seen[f.Name] = true
		if !f.Type.Valid() {
			return fmt.Errorf("schema %q: field %q: unknown type %q", s.Name, f.Name, f.Type)
		}
		if f.List && !stringRepresented(f.Type) {
			return fmt.Errorf("schema %q: field %q: list is only supported for string-valued types, not %q", s.Name, f.Name, f.Type)
		}
		switch f.Placement {
		case "", PlacementFrontmatter:
		case PlacementBody:
			if f.Section == "" {
				return fmt.Errorf("schema %q: field %q: placement %q requires \"section\"", s.Name, f.Name, f.Placement)
			}
		default:
			return fmt.Errorf("schema %q: field %q: unknown placement %q (valid: %q, %q)", s.Name, f.Name, f.Placement, PlacementFrontmatter, PlacementBody)
		}
		if f.Placement == PlacementFrontmatter && datamodel.IsFrontmatterKey(f.Name) {
			return fmt.Errorf("schema %q: field %q: placement \"frontmatter\" shadows a builtin frontmatter key", s.Name, f.Name)
		}
		switch f.Type {
		case FieldEnum, FieldResolution:
			if f.Enum == "" {
				return fmt.Errorf("schema %q: field %q: type %q requires \"enum\"", s.Name, f.Name, f.Type)
			}
		case FieldState:
			if len(f.States) == 0 {
				return fmt.Errorf("schema %q: field %q: type \"state\" requires \"states\"", s.Name, f.Name)
			}
			for _, sv := range f.States {
				if sv.Key == "" {
					return fmt.Errorf("schema %q: field %q: a state entry is missing \"key\"", s.Name, f.Name)
				}
				if !slices.Contains(datamodel.Categories, sv.Category) {
					return fmt.Errorf("schema %q: field %q: state %q: invalid category %q", s.Name, f.Name, sv.Key, sv.Category)
				}
			}
		case FieldRef:
			if f.Target == "" {
				return fmt.Errorf("schema %q: field %q: type \"ref\" requires \"target\"", s.Name, f.Name)
			}
		}
	}
	return nil
}
