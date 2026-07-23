package entityschema

import (
	"fmt"
	"slices"
	"time"

	"github.com/shivamshivanshu/kira/internal/datamodel"
)

type Violation struct {
	Field   string
	Message string
}

func (v Violation) Error() string { return fmt.Sprintf("field %q: %s", v.Field, v.Message) }

// A nil RefResolver skips ref existence checks (phase 1 defers enforcement).
type RefResolver interface {
	Exists(target, id string) bool
}

// Validate is pure. values is keyed by FieldDef.Name (absent or nil means
// unset; list fields use []string). A field name absent from enums is an open
// vocabulary and is not membership-checked.
func Validate(schema Schema, values map[string]any, enums map[string][]string, refs RefResolver) []Violation {
	var out []Violation
	add := func(field, format string, args ...any) {
		out = append(out, Violation{Field: field, Message: fmt.Sprintf(format, args...)})
	}

	for _, f := range schema.Fields {
		v, present := values[f.Name]
		if !present || v == nil {
			if f.Required {
				add(f.Name, "required, missing")
			}
			continue
		}
		validateField(f, v, enums, refs, add)
	}
	return out
}

func validateField(f FieldDef, v any, enums map[string][]string, refs RefResolver, add func(field, format string, args ...any)) {
	if f.List {
		list, ok := v.([]string)
		if !ok {
			add(f.Name, "expected a list, got %T", v)
			return
		}
		seen := make(map[string]bool, len(list))
		for _, elem := range list {
			if f.Unique {
				if seen[elem] {
					add(f.Name, "duplicate value %q in a unique list", elem)
				}
				seen[elem] = true
			}
			validateScalar(f, elem, enums, refs, add)
		}
		return
	}
	validateScalar(f, v, enums, refs, add)
}

func validateScalar(f FieldDef, v any, enums map[string][]string, refs RefResolver, add func(field, format string, args ...any)) {
	if !checkType(f, v, add) {
		return
	}
	switch f.Type {
	case FieldEnum, FieldResolution:
		checkEnumMembership(f, v.(string), enums, add)
	case FieldState:
		checkStateMembership(f, v.(string), add)
	case FieldRef:
		checkRefExists(f, v.(string), refs, add)
	}
}

func checkType(f FieldDef, v any, add func(field, format string, args ...any)) bool {
	ok := false
	switch f.Type {
	case FieldString, FieldMarkdown, FieldEnum, FieldState, FieldResolution, FieldRef:
		_, ok = v.(string)
	case FieldInt:
		ok = isWholeNumber(v)
	case FieldNumber:
		ok = isNumber(v)
	case FieldBool:
		_, ok = v.(bool)
	case FieldDate:
		s, isStr := v.(string)
		ok = isStr && datamodel.ValidDate(s)
	case FieldDatetime:
		s, isStr := v.(string)
		ok = isStr && validDatetime(s)
	}
	if !ok {
		add(f.Name, "expected type %s, got %T", f.Type, v)
	}
	return ok
}

// stringRepresented reports whether checkType validates t as a string. List
// validation only handles []string, so non-string list elements are rejected
// at schema load. Keep in sync with checkType.
func stringRepresented(t FieldType) bool {
	switch t {
	case FieldString, FieldMarkdown, FieldEnum, FieldState, FieldResolution, FieldRef, FieldDate, FieldDatetime:
		return true
	}
	return false
}

func checkEnumMembership(f FieldDef, value string, enums map[string][]string, add func(field, format string, args ...any)) {
	if f.Enum == "" {
		return
	}
	allowed, ok := enums[f.Enum]
	if !ok {
		return
	}
	if !slices.Contains(allowed, value) {
		add(f.Name, "value %q is not in the %q vocabulary", value, f.Enum)
	}
}

func checkStateMembership(f FieldDef, value string, add func(field, format string, args ...any)) {
	if slices.ContainsFunc(f.States, func(s StateValue) bool { return s.Key == value }) {
		return
	}
	add(f.Name, "value %q is not one of the declared states", value)
}

func checkRefExists(f FieldDef, value string, refs RefResolver, add func(field, format string, args ...any)) {
	if refs == nil || f.Target == "" {
		return
	}
	if !refs.Exists(f.Target, value) {
		add(f.Name, "references a non-existent %s %q", f.Target, value)
	}
}

func isNumber(v any) bool {
	switch v.(type) {
	case float64, float32, int, int32, int64:
		return true
	}
	return false
}

func isWholeNumber(v any) bool {
	switch n := v.(type) {
	case int, int32, int64:
		return true
	case float64:
		return n == float64(int64(n))
	case float32:
		return n == float32(int64(n))
	}
	return false
}

func validDatetime(s string) bool {
	_, err := time.Parse(time.RFC3339, s)
	return err == nil
}
