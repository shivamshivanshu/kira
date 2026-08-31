package codec_test

import (
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/shivamshivanshu/kira/internal/codec"
	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func docWithExtras(extraLines string) string {
	return "---\n" +
		"id: 01J8X8Q7RZTN5Y3VXW2A9K4E7F\n" +
		"number: KIRA-1\n" +
		"aliases: []\n" +
		"type: ticket\n" +
		"title: \"t\"\n" +
		"state: TODO\n" +
		"labels: []\n" +
		"epic: null\n" +
		"blocked_by: []\n" +
		extraLines +
		"created: 2026-07-10T09:14:00+05:30\n" +
		"updated: 2026-07-12T11:02:00+05:30\n" +
		"---\nbody\n"
}

func TestExtrasRoundTripIdempotent(t *testing.T) {
	src := docWithExtras("team: infra\nwatchers: [alice, bob]\n")
	it, err := codec.Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	team := "infra"
	want := map[string]datamodel.ExtraValue{
		"team":     {Scalar: &team},
		"watchers": {List: []string{"alice", "bob"}},
	}
	if !reflect.DeepEqual(it.Extras, want) {
		t.Fatalf("Extras = %+v, want %+v", it.Extras, want)
	}
	out := codec.Serialize(it)
	for _, needle := range []string{"team: infra\n", "watchers: [alice, bob]\n"} {
		if !strings.Contains(out, needle) {
			t.Fatalf("serialized output lost extra %q:\n%s", needle, out)
		}
	}
	again, err := codec.Parse(out)
	if err != nil {
		t.Fatalf("re-parse: %v", err)
	}
	if !reflect.DeepEqual(again.Extras, it.Extras) {
		t.Fatalf("re-parsed Extras = %+v, want %+v", again.Extras, it.Extras)
	}
	if out2 := codec.Serialize(again); out2 != out {
		t.Fatalf("serialize not idempotent\n--- first ---\n%s\n--- second ---\n%s", out, out2)
	}
}

func TestExtrasNotCapturedShapes(t *testing.T) {
	tests := []struct {
		name string
		line string
		key  string
	}{
		{"mapping", "custom_map:\n  a: b\n", "custom_map"},
		{"sequence with mapping element", "weird: [{a: b}]\n", "weird"},
		{"null", "nothing: null\n", "nothing"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			it, err := codec.Parse(docWithExtras(tt.line))
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			if !slices.Contains(it.UnknownKeys, tt.key) {
				t.Fatalf("UnknownKeys = %v, want %q recorded", it.UnknownKeys, tt.key)
			}
			if _, ok := it.Extras[tt.key]; ok {
				t.Fatalf("Extras = %+v, %q must not be captured", it.Extras, tt.key)
			}
			out := codec.Serialize(it)
			if strings.Contains(out, tt.key) {
				t.Fatalf("uncaptured key %q must not be re-emitted:\n%s", tt.key, out)
			}
			again, err := codec.Parse(out)
			if err != nil {
				t.Fatalf("re-parse: %v", err)
			}
			if out2 := codec.Serialize(again); out2 != out {
				t.Fatalf("serialize not a fixed point\n%q\n%q", out, out2)
			}
		})
	}
}

func TestExtraScalarsKeepTheirYAMLType(t *testing.T) {
	tests := []struct {
		name string
		line string
	}{
		{"int", "points: 42\n"},
		{"float", "ratio: 1.5\n"},
		{"bool", "flag: true\n"},
		{"quoted digits stay quoted", "code: \"07\"\n"},
		{"plain string", "owner_note: whatever\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			it, err := codec.Parse(docWithExtras(tt.line))
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			out := codec.Serialize(it)
			if !strings.Contains(out, tt.line) {
				t.Fatalf("want %q emitted verbatim, got:\n%s", tt.line, out)
			}
		})
	}
}

func TestExtraListsKeepElementTypesFlowSafely(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"typed elements verbatim", "history: [1, \"2\", true, null]\n", "history: [1, \"2\", true, null]\n"},
		{"block element with comma quoted for flow", "k:\n  - a,b\n", "k: ['a,b']\n"},
		{"block element with bracket quoted for flow", "k:\n  - \"x]y\"\n", "k: [\"x]y\"]\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			it, err := codec.Parse(docWithExtras(tt.in))
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			out := codec.Serialize(it)
			if !strings.Contains(out, tt.want) {
				t.Fatalf("want %q emitted, got:\n%s", tt.want, out)
			}
			again, err := codec.Parse(out)
			if err != nil {
				t.Fatalf("re-parse: %v", err)
			}
			if !reflect.DeepEqual(again.Extras, it.Extras) {
				t.Fatalf("re-parsed Extras = %+v, want %+v", again.Extras, it.Extras)
			}
		})
	}
}

func TestExtraCommentAndAnchorMetadataStripped(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"line comment", "k: val # note\n", "k: val\n"},
		{"head comment", "k:\n  # note\n  val\n", "k: val\n"},
		{"anchor", "k: &a val\n", "k: val\n"},
		{"list element comment", "k:\n  - v # note\n", "k: [v]\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			it, err := codec.Parse(docWithExtras(tt.in))
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			out := codec.Serialize(it)
			if !strings.Contains(out, tt.want) {
				t.Fatalf("want bare value %q emitted, got:\n%s", tt.want, out)
			}
		})
	}
}
