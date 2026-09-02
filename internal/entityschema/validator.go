package entityschema

import (
	"fmt"
	"slices"
	"time"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/setx"
)

type ViolationCode string

const (
	CodeRequired ViolationCode = "required"
	CodeType     ViolationCode = "type"
	CodeEnum     ViolationCode = "enum"
	CodeState    ViolationCode = "state"
	CodeRef      ViolationCode = "ref"
)

type Violation struct {
	Field   string
	Code    ViolationCode
	Message string
}

func (v Violation) Error() string { return fmt.Sprintf("field %q: %s", v.Field, v.Message) }

// A nil RefResolver skips ref existence checks (phase 1 defers enforcement).
type RefResolver interface {
	Exists(target, id string) bool
}

type addFunc func(field string, code ViolationCode, format string, args ...any)

// Validate is pure. values is keyed by FieldDef.Name (absent, nil, or an empty
// string means unset; list fields use []string). A field name absent from
// enums is an open vocabulary and is not membership-checked.
func Validate(schema Schema, values map[string]any, enums map[string][]string, refs RefResolver) []Violation {
	var out []Violation
	add := func(field string, code ViolationCode, format string, args ...any) {
		out = append(out, Violation{Field: field, Code: code, Message: fmt.Sprintf(format, args...)})
	}

	for _, f := range schema.Fields {
		v, present := values[f.Name]
		if !present || v == nil || (!f.List && isBlank(v)) {
			if f.Required {
				add(f.Name, CodeRequired, "required, missing")
			}
			continue
		}
		validateField(f, v, enums, refs, add)
	}
	return out
}

func isBlank(v any) bool {
	s, ok := v.(string)
	return ok && s == ""
}

func validateField(f FieldDef, v any, enums map[string][]string, refs RefResolver, add addFunc) {
	if f.List {
		list, ok := v.([]string)
		if !ok {
			add(f.Name, CodeType, "expected a list, got %T", v)
			return
		}
		seen := setx.NewDeduper[string]()
		for _, elem := range list {
			if f.Unique {
				if !seen.Add(elem) {
					add(f.Name, CodeType, "duplicate value %q in a unique list", elem)
				}
			}
			validateScalar(f, elem, enums, refs, add)
		}
		return
	}
	validateScalar(f, v, enums, refs, add)
}

// validateScalar checks a ref field's existence against RefResolver when it
// targets another entity (no config-vocab Source), and against enums when it
// is instead a config-vocab-backed reference (e.g. owner/reporter -> person).
func validateScalar(f FieldDef, v any, enums map[string][]string, refs RefResolver, add addFunc) {
	if !checkType(f, v, add) {
		return
	}
	switch f.Type {
	case FieldEnum, FieldResolution:
		checkEnumMembership(f.Name, f.Enum, v.(string), enums, add)
	case FieldState:
		checkStateMembership(f, v.(string), add)
	case FieldRef:
		if f.Source != "" {
			checkEnumMembership(f.Name, f.Target, v.(string), enums, add)
		} else {
			checkRefExists(f, v.(string), refs, add)
		}
	}
}

func checkType(f FieldDef, v any, add addFunc) bool {
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
		ok = isTemporal(v, datamodel.ValidDate)
	case FieldDatetime:
		ok = isTemporal(v, validDatetime)
	}
	if !ok {
		add(f.Name, CodeType, "expected type %s, got %T", f.Type, v)
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

func checkEnumMembership(field, vocabName, value string, enums map[string][]string, add addFunc) {
	if vocabName == "" {
		return
	}
	allowed, ok := enums[vocabName]
	if !ok {
		return
	}
	if !slices.Contains(allowed, value) {
		add(field, CodeEnum, "value %q is not in the %q vocabulary", value, vocabName)
	}
}

func checkStateMembership(f FieldDef, value string, add addFunc) {
	if slices.ContainsFunc(f.States, func(s StateValue) bool { return s.Key == value }) {
		return
	}
	add(f.Name, CodeState, "value %q is not one of the declared states", value)
}

func checkRefExists(f FieldDef, value string, refs RefResolver, add addFunc) {
	if refs == nil || f.Target == "" {
		return
	}
	if !refs.Exists(f.Target, value) {
		add(f.Name, CodeRef, "references a non-existent %s %q", f.Target, value)
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

// A hand-edited unquoted date decodes to time.Time rather than a string, so
// both spellings have to satisfy a temporal field.
func isTemporal(v any, validText func(string) bool) bool {
	switch t := v.(type) {
	case string:
		return validText(t)
	case time.Time:
		return true
	}
	return false
}
