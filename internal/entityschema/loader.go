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
	"unicode"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/setx"
)

//go:embed builtin/*.json
var builtinFiles embed.FS

const builtinDir = "builtin"

// File is one user schema document, named for error messages.
type File struct {
	Name string
	Data []byte
}

// Load layers user .kira/schema/*.json over the embedded built-ins, overriding
// by name; a missing dir yields the built-ins alone.
func Load(dir string) (map[string]Schema, error) {
	if dir == "" {
		return LoadFiles(nil)
	}
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return LoadFiles(nil)
	}
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", dir, err)
	}
	var files []File
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", e.Name(), err)
		}
		files = append(files, File{Name: e.Name(), Data: data})
	}
	return LoadFiles(files)
}

// LoadFiles layers already-read schema documents over the embedded built-ins,
// for schema files that come from somewhere other than a working tree — a git
// tree-ish, say.
func LoadFiles(files []File) (map[string]Schema, error) {
	schemas, err := loadBuiltins()
	if err != nil {
		return nil, err
	}
	for _, f := range files {
		schema, err := parseSchema(f.Name, f.Data)
		if err != nil {
			return nil, err
		}
		schemas[schema.Name] = schema
	}
	if err := checkExtrasAgreement(schemas); err != nil {
		return nil, err
	}
	return schemas, nil
}

// checkExtrasAgreement rejects two schemas declaring the same extra with a
// different type or list-ness: the query layer builds one spec per field name
// across all schemas, so a disagreement would let one schema's traits silently
// win for both types.
func checkExtrasAgreement(schemas map[string]Schema) error {
	type declaration struct {
		schema string
		field  FieldDef
	}
	first := map[string]declaration{}
	for _, name := range slices.Sorted(maps.Keys(schemas)) {
		for _, f := range schemas[name].Fields {
			if !f.IsCustomFrontmatter() {
				continue
			}
			prev, ok := first[f.Name]
			if !ok {
				first[f.Name] = declaration{schema: name, field: f}
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
func WriteDefaults(dir string) ([]string, error) {
	entries, err := builtinFiles.ReadDir(builtinDir)
	if err != nil {
		return nil, fmt.Errorf("reading embedded schemas: %w", err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("creating %s: %w", dir, err)
	}
	var written []string
	for _, e := range entries {
		dst := filepath.Join(dir, e.Name())
		if _, err := os.Stat(dst); err == nil {
			continue // don't clobber a user-edited schema
		} else if !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("checking %s: %w", dst, err)
		}
		data, err := builtinFiles.ReadFile(filepath.Join(builtinDir, e.Name()))
		if err != nil {
			return nil, fmt.Errorf("reading embedded %s: %w", e.Name(), err)
		}
		if err := os.WriteFile(dst, data, 0o644); err != nil {
			return nil, fmt.Errorf("writing %s: %w", dst, err)
		}
		written = append(written, e.Name())
	}
	return written, nil
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

// Displayed schema text must be a single printable run, because every renderer
// measures it in display cells and then trusts that measurement for layout. A
// newline defeats that at the root: it measures as the widest line while
// occupying two, so the tui wraps rather than clips (doubling every tree row
// and pushing the status bar off screen) and the cli's tabwriter loses column
// sync for the whole table, not just the offending row. Format characters are
// rejected for the same reason from the other direction -- they measure zero,
// so U+202E reorders the rest of the line invisibly -- and a leading combining
// mark composes onto whatever the renderer emitted before it.
func checkDisplayText(schemaName, field, text string) error {
	for i, r := range text {
		switch {
		case unicode.IsControl(r):
			return fmt.Errorf("schema %q: representation %s contains a control character %q", schemaName, field, r)
		case unicode.Is(unicode.Cf, r):
			return fmt.Errorf("schema %q: representation %s contains the zero-width format character %q", schemaName, field, r)
		case i == 0 && unicode.In(r, unicode.Mn, unicode.Me, unicode.Mc):
			return fmt.Errorf("schema %q: representation %s starts with the combining mark %q, which composes onto the preceding character", schemaName, field, r)
		}
	}
	return nil
}

func checkSchemaShape(s Schema) error {
	if s.Name == "" {
		return errors.New(`schema missing "name"`)
	}
	for _, text := range []struct{ field, value string }{
		{"icon", s.Representation.Icon},
		{"label", s.Representation.Label},
	} {
		if err := checkDisplayText(s.Name, text.field, text.value); err != nil {
			return err
		}
	}
	seen := setx.NewDeduper[string]()
	for _, f := range s.Fields {
		if f.Name == "" {
			return fmt.Errorf("schema %q: field missing \"name\"", s.Name)
		}
		if !seen.Add(f.Name) {
			return fmt.Errorf("schema %q: duplicate field %q", s.Name, f.Name)
		}
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
				return fmt.Errorf("schema %q: field %q: body placement requires \"section\"", s.Name, f.Name)
			}
		default:
			return fmt.Errorf("schema %q: field %q: unknown placement %q (valid: %q, %q)", s.Name, f.Name, f.Placement, PlacementFrontmatter, PlacementBody)
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
