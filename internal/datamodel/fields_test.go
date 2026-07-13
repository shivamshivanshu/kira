package datamodel

import (
	"slices"
	"testing"
)

func fieldsBaseItem() *Item {
	sub := "bug"
	return &Item{
		Title:     "t",
		State:     "todo",
		Subtype:   &sub,
		Labels:    []string{"x"},
		BlockedBy: []string{"01A"},
		Links:     map[string][]string{string(LinkRelates): {"01B"}},
		Body:      "body",
	}
}

func TestChangedFieldsNoChange(t *testing.T) {
	if got := ChangedFields(fieldsBaseItem(), fieldsBaseItem()); len(got) != 0 {
		t.Fatalf("ChangedFields on identical items = %v, want none", got)
	}
}

func TestChangedFieldsPerKind(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Item)
		want   string
	}{
		{"title", func(it *Item) { it.Title = "u" }, KeyTitle},
		{"state", func(it *Item) { it.State = "doing" }, KeyState},
		{"subtype change", func(it *Item) { s := "task"; it.Subtype = &s }, KeySubtype},
		{"subtype clear", func(it *Item) { it.Subtype = nil }, KeySubtype},
		{"priority set", func(it *Item) { p := "high"; it.Priority = &p }, KeyPriority},
		{"labels", func(it *Item) { it.Labels = []string{"y"} }, KeyLabels},
		{"blocked_by", func(it *Item) { it.BlockedBy = []string{"01C"} }, KeyBlockedBy},
		{"links", func(it *Item) { it.Links = map[string][]string{string(LinkRelates): {"01Z"}} }, KeyLinks},
		{"estimate", func(it *Item) { e := 3.0; it.Estimate = &e }, KeyEstimate},
		{"body", func(it *Item) { it.Body = "new" }, KeyBody},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a, b := fieldsBaseItem(), fieldsBaseItem()
			tt.mutate(b)
			got := ChangedFields(a, b)
			if !slices.Equal(got, []string{tt.want}) {
				t.Fatalf("ChangedFields = %v, want [%s]", got, tt.want)
			}
		})
	}
}

func TestFieldLookup(t *testing.T) {
	if _, ok := Field(KeyTitle); !ok {
		t.Errorf("Field(%q) not found", KeyTitle)
	}
	if _, ok := Field("bogus"); ok {
		t.Errorf("Field(bogus) found, want miss")
	}
}

func TestFieldDescriptorGet(t *testing.T) {
	est := 2.5
	it := &Item{
		Title:    "hello",
		Priority: nil,
		Labels:   nil,
		Estimate: &est,
		Body:     "text",
	}
	cases := []struct {
		key  string
		want string
	}{
		{KeyTitle, "hello"},
		{KeyPriority, "-"},
		{KeyLabels, "[]"},
		{KeyEstimate, "2.5"},
		{KeyBody, "(body)"},
	}
	for _, c := range cases {
		d, ok := Field(c.key)
		if !ok {
			t.Fatalf("Field(%q) missing", c.key)
		}
		if got := d.Get(it); got != c.want {
			t.Errorf("%s Get = %q, want %q", c.key, got, c.want)
		}
	}
	it.Labels = []string{"a", "b"}
	if got, _ := Field(KeyLabels); got.Get(it) != "[a, b]" {
		t.Errorf("labels Get = %q, want [a, b]", got.Get(it))
	}
}

func TestFieldDescriptorCopyClonesList(t *testing.T) {
	d, _ := Field(KeyLabels)
	src := &Item{Labels: []string{"a"}}
	dst := &Item{}
	d.Copy(dst, src)
	src.Labels[0] = "mutated"
	if !slices.Equal(dst.Labels, []string{"a"}) {
		t.Fatalf("Copy did not clone: dst.Labels = %v", dst.Labels)
	}
}

func TestEqualPtr(t *testing.T) {
	x, y := "a", "b"
	xCopy := "a"
	tests := []struct {
		a, b *string
		want bool
	}{
		{nil, nil, true},
		{&x, nil, false},
		{nil, &x, false},
		{&x, &xCopy, true},
		{&x, &y, false},
	}
	for _, tt := range tests {
		if got := EqualPtr(tt.a, tt.b); got != tt.want {
			t.Errorf("EqualPtr(%v, %v) = %v, want %v", tt.a, tt.b, got, tt.want)
		}
	}
}
