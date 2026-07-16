package datamodel

import (
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"

	"github.com/shivamshivanshu/kira/internal/ptr"
)

type FieldDescriptor struct {
	Key     string
	Guarded bool
	Changed func(a, b *Item) bool
	Get     func(*Item) string
	Copy    func(dst, src *Item)
	Set     func(it *Item, value string) error
	Present func(*Item) bool
}

func (d FieldDescriptor) withoutSet() FieldDescriptor {
	d.Set = nil
	return d
}

func (d FieldDescriptor) guarded() FieldDescriptor {
	d.Guarded = true
	return d
}

func (d FieldDescriptor) withoutPresent() FieldDescriptor {
	d.Present = nil
	return d
}

var Fields = []FieldDescriptor{
	ptrField(KeySubtype, func(it *Item) **string { return &it.Subtype }),
	strField(KeyTitle, func(it *Item) *string { return &it.Title }),
	strField(KeyState, func(it *Item) *string { return &it.State }).withoutPresent().guarded(),
	ptrField(KeyResolution, func(it *Item) **string { return &it.Resolution }),
	ptrField(KeyPriority, func(it *Item) **string { return &it.Priority }),
	ptrField(KeyRank, func(it *Item) **string { return &it.Rank }),
	ptrField(KeyOwner, func(it *Item) **string { return &it.Owner }),
	ptrField(KeyReporter, func(it *Item) **string { return &it.Reporter }),
	listField(KeyLabels, func(it *Item) *[]string { return &it.Labels }),
	ptrField(KeyEpic, func(it *Item) **string { return &it.Epic }),
	listField(KeyBlockedBy, func(it *Item) *[]string { return &it.BlockedBy }).withoutSet().withoutPresent(),
	linksField(),
	ptrField(KeySprint, func(it *Item) **string { return &it.Sprint }),
	ptrField(KeyDue, func(it *Item) **string { return &it.Due }),
	estimateField(),
	bodyField(),
}

var EditableFields = func() []string {
	var out []string
	for _, d := range Fields {
		if d.Set != nil {
			out = append(out, d.Key)
		}
	}
	slices.Sort(out)
	return out
}()

var MutableFields = func() []string {
	var out []string
	for _, d := range Fields {
		if d.Set != nil && !d.Guarded {
			out = append(out, d.Key)
		}
	}
	return out
}()

var fieldByKey = func() map[string]FieldDescriptor {
	m := make(map[string]FieldDescriptor, len(Fields))
	for _, d := range Fields {
		m[d.Key] = d
	}
	return m
}()

func Field(key string) (FieldDescriptor, bool) {
	d, ok := fieldByKey[key]
	return d, ok
}

func ChangedFields(a, b *Item) []string {
	var out []string
	for _, d := range Fields {
		if d.Changed(a, b) {
			out = append(out, d.Key)
		}
	}
	return out
}

func strField(key string, ref func(*Item) *string) FieldDescriptor {
	return FieldDescriptor{
		Key:     key,
		Changed: func(a, b *Item) bool { return *ref(a) != *ref(b) },
		Get:     func(it *Item) string { return *ref(it) },
		Copy:    func(dst, src *Item) { *ref(dst) = *ref(src) },
		Set:     func(it *Item, value string) error { *ref(it) = value; return nil },
		Present: func(it *Item) bool { return *ref(it) != "" },
	}
}

func ptrField(key string, ref func(*Item) **string) FieldDescriptor {
	return FieldDescriptor{
		Key:     key,
		Changed: func(a, b *Item) bool { return !ptr.Equal(*ref(a), *ref(b)) },
		Get: func(it *Item) string {
			if p := *ref(it); p != nil {
				return *p
			}
			return "-"
		},
		Copy:    func(dst, src *Item) { *ref(dst) = *ref(src) },
		Set:     func(it *Item, value string) error { *ref(it) = ptr.NilIfEmpty(value); return nil },
		Present: func(it *Item) bool { p := *ref(it); return p != nil && *p != "" },
	}
}

func listField(key string, ref func(*Item) *[]string) FieldDescriptor {
	return FieldDescriptor{
		Key:     key,
		Changed: func(a, b *Item) bool { return !slices.Equal(*ref(a), *ref(b)) },
		Get:     func(it *Item) string { return listString(*ref(it)) },
		Copy:    func(dst, src *Item) { *ref(dst) = slices.Clone(*ref(src)) },
		Set:     func(it *Item, value string) error { *ref(it) = splitCSV(value); return nil },
		Present: func(it *Item) bool { return len(*ref(it)) > 0 },
	}
}

func linksField() FieldDescriptor {
	return FieldDescriptor{
		Key:     KeyLinks,
		Changed: func(a, b *Item) bool { return !maps.EqualFunc(a.Links, b.Links, slices.Equal[[]string]) },
		Get: func(it *Item) string {
			parts := make([]string, 0, len(it.Links))
			for _, typ := range slices.Sorted(maps.Keys(it.Links)) {
				parts = append(parts, typ+":"+listString(it.Links[typ]))
			}
			return strings.Join(parts, " ")
		},
		Copy: func(dst, src *Item) { dst.Links = CloneLinks(src.Links) },
	}
}

func estimateField() FieldDescriptor {
	return FieldDescriptor{
		Key:     KeyEstimate,
		Changed: func(a, b *Item) bool { return !ptr.Equal(a.Estimate, b.Estimate) },
		Get: func(it *Item) string {
			if it.Estimate == nil {
				return "-"
			}
			return strconv.FormatFloat(*it.Estimate, 'f', -1, 64)
		},
		Copy: func(dst, src *Item) { dst.Estimate = src.Estimate },
		Set: func(it *Item, value string) error {
			if value == "" {
				it.Estimate = nil
				return nil
			}
			f, err := strconv.ParseFloat(value, 64)
			if err != nil {
				return fmt.Errorf("field %q: invalid number %q", KeyEstimate, value)
			}
			it.Estimate = &f
			return nil
		},
		Present: func(it *Item) bool { return it.Estimate != nil },
	}
}

func bodyField() FieldDescriptor {
	return FieldDescriptor{
		Key:     KeyBody,
		Changed: func(a, b *Item) bool { return a.Body != b.Body },
		Get:     func(*Item) string { return "(" + KeyBody + ")" },
		Copy:    func(dst, src *Item) { dst.Body = src.Body },
	}
}

func splitCSV(value string) []string {
	if strings.TrimSpace(value) == "" {
		return []string{}
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}

func listString(xs []string) string {
	if len(xs) == 0 {
		return "[]"
	}
	return "[" + strings.Join(xs, ", ") + "]"
}
