package datamodel

import (
	"maps"
	"slices"
	"strconv"
	"strings"
)

func EqualPtr[T comparable](a, b *T) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

type FieldDescriptor struct {
	Key     string
	Changed func(a, b *Item) bool
	Get     func(*Item) string
	Copy    func(dst, src *Item)
}

var Fields = []FieldDescriptor{
	ptrField(KeySubtype, func(it *Item) **string { return &it.Subtype }),
	strField(KeyTitle, func(it *Item) *string { return &it.Title }),
	strField(KeyState, func(it *Item) *string { return &it.State }),
	ptrField(KeyResolution, func(it *Item) **string { return &it.Resolution }),
	ptrField(KeyPriority, func(it *Item) **string { return &it.Priority }),
	ptrField(KeyRank, func(it *Item) **string { return &it.Rank }),
	ptrField(KeyOwner, func(it *Item) **string { return &it.Owner }),
	ptrField(KeyReporter, func(it *Item) **string { return &it.Reporter }),
	listField(KeyLabels, func(it *Item) *[]string { return &it.Labels }),
	ptrField(KeyEpic, func(it *Item) **string { return &it.Epic }),
	listField(KeyBlockedBy, func(it *Item) *[]string { return &it.BlockedBy }),
	linksField(),
	ptrField(KeySprint, func(it *Item) **string { return &it.Sprint }),
	ptrField(KeyDue, func(it *Item) **string { return &it.Due }),
	estimateField(),
	bodyField(),
}

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
	}
}

func ptrField(key string, ref func(*Item) **string) FieldDescriptor {
	return FieldDescriptor{
		Key:     key,
		Changed: func(a, b *Item) bool { return !EqualPtr(*ref(a), *ref(b)) },
		Get:     func(it *Item) string { return derefOrDash(*ref(it)) },
		Copy:    func(dst, src *Item) { *ref(dst) = *ref(src) },
	}
}

func listField(key string, ref func(*Item) *[]string) FieldDescriptor {
	return FieldDescriptor{
		Key:     key,
		Changed: func(a, b *Item) bool { return !slices.Equal(*ref(a), *ref(b)) },
		Get:     func(it *Item) string { return listString(*ref(it)) },
		Copy:    func(dst, src *Item) { *ref(dst) = slices.Clone(*ref(src)) },
	}
}

func linksField() FieldDescriptor {
	return FieldDescriptor{
		Key:     KeyLinks,
		Changed: func(a, b *Item) bool { return !maps.EqualFunc(a.Links, b.Links, slices.Equal[[]string]) },
		Get:     func(it *Item) string { return strings.Join(slices.Sorted(maps.Keys(it.Links)), ",") },
		Copy:    func(dst, src *Item) { dst.Links = src.Links },
	}
}

func estimateField() FieldDescriptor {
	return FieldDescriptor{
		Key:     KeyEstimate,
		Changed: func(a, b *Item) bool { return !EqualPtr(a.Estimate, b.Estimate) },
		Get: func(it *Item) string {
			if it.Estimate == nil {
				return "-"
			}
			return strconv.FormatFloat(*it.Estimate, 'f', -1, 64)
		},
		Copy: func(dst, src *Item) { dst.Estimate = src.Estimate },
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

func derefOrDash(p *string) string {
	if p == nil {
		return "-"
	}
	return *p
}

func listString(xs []string) string {
	if len(xs) == 0 {
		return "[]"
	}
	return "[" + strings.Join(xs, ", ") + "]"
}
