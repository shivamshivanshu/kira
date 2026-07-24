package core

import (
	"slices"
	"strconv"
	"strings"

	"github.com/shivamshivanshu/kira/internal/codec"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/entityschema"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/setx"
)

// changedExtraKeys reports the extra keys whose value differs between a and
// b. datamodel.ChangedFields doesn't see extras at all (CORE-42 covers making
// it, and 3-way merge arbitration, extras-aware); a mutation that only
// touches an extra still needs to be recognized as a real change so the write
// path doesn't silently drop it.
func changedExtraKeys(a, b *datamodel.Item) []string {
	seen := setx.NewDeduper[string]()
	var out []string
	check := func(k string) {
		if !seen.Add(k) {
			return
		}
		if a.ExtraScalars[k] != b.ExtraScalars[k] || !slices.Equal(a.ExtraLists[k], b.ExtraLists[k]) {
			out = append(out, k)
		}
	}
	for _, k := range a.ExtraKeys() {
		check(k)
	}
	for _, k := range b.ExtraKeys() {
		check(k)
	}
	slices.Sort(out)
	return out
}

func fieldEditableViaFlag(f entityschema.FieldDef) bool {
	return !f.System && !f.Immutable && !f.Guarded && f.Placement != entityschema.PlacementBody
}

func setSchemaExtra(it *datamodel.Item, f entityschema.FieldDef, value string) error {
	if f.List {
		return setExtraList(it, f, value)
	}
	return setExtraScalar(it, f, value)
}

func setExtraScalar(it *datamodel.Item, f entityschema.FieldDef, value string) error {
	if value == "" {
		delete(it.ExtraScalars, f.Name)
		return nil
	}
	typed, err := typedExtraValue(f, value)
	if err != nil {
		return err
	}
	rendered, err := codec.EncodeExtraScalar(typed)
	if err != nil {
		return errx.User("--field %q: %v", f.Name, err)
	}
	if it.ExtraScalars == nil {
		it.ExtraScalars = map[string]string{}
	}
	it.ExtraScalars[f.Name] = rendered
	return nil
}

func setExtraList(it *datamodel.Item, f entityschema.FieldDef, value string) error {
	if strings.TrimSpace(value) == "" {
		delete(it.ExtraLists, f.Name)
		return nil
	}
	if it.ExtraLists == nil {
		it.ExtraLists = map[string][]string{}
	}
	it.ExtraLists[f.Name] = datamodel.SplitCSV(value)
	return nil
}

// typedExtraValue parses value per f's scalar type; list elements are always
// string-representable (enforced at schema load), so only scalars parse here.
func typedExtraValue(f entityschema.FieldDef, value string) (any, error) {
	switch f.Type {
	case entityschema.FieldInt:
		n, err := strconv.Atoi(value)
		if err != nil {
			return nil, errx.User("--field %q: expected an integer, got %q", f.Name, value)
		}
		return n, nil
	case entityschema.FieldNumber:
		n, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return nil, errx.User("--field %q: expected a number, got %q", f.Name, value)
		}
		return n, nil
	case entityschema.FieldBool:
		b, err := strconv.ParseBool(value)
		if err != nil {
			return nil, errx.User("--field %q: expected a bool, got %q", f.Name, value)
		}
		return b, nil
	default:
		return value, nil
	}
}
