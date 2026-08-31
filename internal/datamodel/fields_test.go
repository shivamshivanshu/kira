package datamodel

import (
	"maps"
	"slices"
	"testing"
)

func TestMutableFieldsOrder(t *testing.T) {
	want := []string{
		KeySubtype, KeyTitle, KeyResolution, KeyPriority, KeyRank, KeyOwner,
		KeyReporter, KeyLabels, KeyEpic, KeySprint, KeyDue, KeyEstimate,
	}
	if !slices.Equal(MutableFields, want) {
		t.Fatalf("MutableFields = %v, want %v", MutableFields, want)
	}
}

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
		{KeyLinks, ""},
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
	it.Links = map[string][]string{
		string(LinkRelates):     {"01B", "01C"},
		string(LinkDuplicateOf): {"01X"},
	}
	want := "duplicate_of:[01X] relates:[01B, 01C]"
	if got, _ := Field(KeyLinks); got.Get(it) != want {
		t.Errorf("links Get = %q, want %q", got.Get(it), want)
	}
}

func TestPtrFieldGetDistinguishesNilFromPresentEmpty(t *testing.T) {
	empty := ""
	it := &Item{Owner: &empty}
	d, _ := Field(KeyOwner)
	if got := d.Get(it); got != "" {
		t.Errorf("Get on a present-but-empty pointer = %q, want empty string (not the nil placeholder)", got)
	}
	it.Owner = nil
	if got := d.Get(it); got != "-" {
		t.Errorf("Get on a nil pointer = %q, want \"-\"", got)
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

func TestFieldDescriptorCopyClonesLinks(t *testing.T) {
	d, _ := Field(KeyLinks)
	src := &Item{Links: map[string][]string{string(LinkRelates): {"01B"}}}
	dst := &Item{}
	d.Copy(dst, src)
	src.Links[string(LinkRelates)][0] = "mutated"
	src.Links[string(LinkDuplicateOf)] = []string{"01X"}
	want := map[string][]string{string(LinkRelates): {"01B"}}
	if !maps.EqualFunc(dst.Links, want, slices.Equal[[]string]) {
		t.Fatalf("Copy did not deep-clone: dst.Links = %v", dst.Links)
	}
}

func scalarExtra(s string) ExtraValue { return ExtraValue{Scalar: &s} }

func TestChangedFieldsExtras(t *testing.T) {
	tests := []struct {
		name string
		a, b map[string]ExtraValue
		want []string
	}{
		{
			"equal",
			map[string]ExtraValue{"team": scalarExtra("infra"), "watchers": {List: []string{"a"}}},
			map[string]ExtraValue{"team": scalarExtra("infra"), "watchers": {List: []string{"a"}}},
			nil,
		},
		{"added", nil, map[string]ExtraValue{"team": scalarExtra("infra")}, []string{"team"}},
		{"removed", map[string]ExtraValue{"team": scalarExtra("infra")}, nil, []string{"team"}},
		{
			"modified scalar",
			map[string]ExtraValue{"team": scalarExtra("infra")},
			map[string]ExtraValue{"team": scalarExtra("core")},
			[]string{"team"},
		},
		{
			"modified list",
			map[string]ExtraValue{"watchers": {List: []string{"a"}}},
			map[string]ExtraValue{"watchers": {List: []string{"a", "b"}}},
			[]string{"watchers"},
		},
		{
			"independent keys sorted",
			map[string]ExtraValue{"k2": scalarExtra("y")},
			map[string]ExtraValue{"k1": scalarExtra("x")},
			[]string{"k1", "k2"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a, b := fieldsBaseItem(), fieldsBaseItem()
			a.Extras, b.Extras = tt.a, tt.b
			if got := ChangedFields(a, b); !slices.Equal(got, tt.want) {
				t.Fatalf("ChangedFields = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtraFieldScalar(t *testing.T) {
	d := ExtraField("team", false)
	it := &Item{}

	if d.Present(it) || d.Get(it) != "-" {
		t.Fatalf("absent extra: Present=%v Get=%q", d.Present(it), d.Get(it))
	}
	if err := d.Set(it, "infra"); err != nil {
		t.Fatal(err)
	}
	if !d.Present(it) || d.Get(it) != "infra" {
		t.Fatalf("set extra: Present=%v Get=%q", d.Present(it), d.Get(it))
	}
	other := &Item{}
	if !d.Changed(it, other) {
		t.Fatal("Changed must see set-vs-absent")
	}
	d.Copy(other, it)
	if d.Get(other) != "infra" || d.Changed(it, other) {
		t.Fatalf("Copy did not carry the value: %q", d.Get(other))
	}
	if err := d.Set(it, ""); err != nil {
		t.Fatal(err)
	}
	if d.Present(it) {
		t.Fatal("empty Set must clear the extra")
	}
	d.Copy(other, it)
	if d.Present(other) {
		t.Fatal("Copy from an absent source must clear the destination")
	}
}

func TestExtraFieldList(t *testing.T) {
	d := ExtraField("watchers", true)
	it := &Item{}

	if err := d.Set(it, "a, b"); err != nil {
		t.Fatal(err)
	}
	if !d.Present(it) || d.Get(it) != "[a, b]" {
		t.Fatalf("list extra: Present=%v Get=%q", d.Present(it), d.Get(it))
	}
	if !slices.Equal(it.Extras["watchers"].List, []string{"a", "b"}) {
		t.Fatalf("stored list = %v", it.Extras["watchers"].List)
	}
}
