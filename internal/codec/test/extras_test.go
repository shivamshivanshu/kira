package codec_test

import (
	"errors"
	"reflect"
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
	wantScalars := map[string]string{"team": "infra"}
	wantLists := map[string][]string{"watchers": {"alice", "bob"}}
	if !reflect.DeepEqual(it.ExtraScalars, wantScalars) {
		t.Fatalf("ExtraScalars = %+v, want %+v", it.ExtraScalars, wantScalars)
	}
	if !reflect.DeepEqual(it.ExtraLists, wantLists) {
		t.Fatalf("ExtraLists = %+v, want %+v", it.ExtraLists, wantLists)
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
	if !reflect.DeepEqual(again.ExtraScalars, it.ExtraScalars) || !reflect.DeepEqual(again.ExtraLists, it.ExtraLists) {
		t.Fatalf("re-parsed extras = %+v/%+v, want %+v/%+v", again.ExtraScalars, again.ExtraLists, it.ExtraScalars, it.ExtraLists)
	}
	if out2 := codec.Serialize(again); out2 != out {
		t.Fatalf("serialize not idempotent\n--- first ---\n%s\n--- second ---\n%s", out, out2)
	}
}

func TestExtraListWithNonScalarElementRejected(t *testing.T) {
	_, err := codec.Parse(docWithExtras("weird: [{a: b}]\n"))
	var pe *datamodel.ParseError
	if !errors.As(err, &pe) {
		t.Fatalf("a list extra with a mapping element must be a parse error, got %v", err)
	}
	if !strings.Contains(err.Error(), "weird") {
		t.Fatalf("error must name the offending field: %v", err)
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

func TestExtraListElementsQuotedForFlow(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"element with comma", "k:\n  - a,b\n", "k: ['a,b']\n"},
		{"element with bracket", "k:\n  - \"x]y\"\n", "k: ['x]y']\n"},
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
			if !reflect.DeepEqual(again.ExtraLists, it.ExtraLists) {
				t.Fatalf("re-parsed ExtraLists = %+v, want %+v", again.ExtraLists, it.ExtraLists)
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

func TestExtraFrontmatterReadsDraftFields(t *testing.T) {
	draft := "---\ntitle: t\nteam: infra\nwatchers: [alice, bob]\n---\nbody\n"
	scalars, lists, err := codec.ExtraFrontmatter(draft)
	if err != nil {
		t.Fatalf("ExtraFrontmatter: %v", err)
	}
	if !reflect.DeepEqual(scalars, map[string]string{"team": "infra"}) {
		t.Fatalf("scalars = %+v, want team only (title is builtin)", scalars)
	}
	if !reflect.DeepEqual(lists, map[string][]string{"watchers": {"alice", "bob"}}) {
		t.Fatalf("lists = %+v, want watchers", lists)
	}

	if _, _, err := codec.ExtraFrontmatter("---\ntitle: t\nnested:\n  a: b\n---\n"); err == nil {
		t.Fatal("a nested mapping must be reported rather than silently dropped")
	}
}

func TestExtraListElementsKeepTheirYAMLType(t *testing.T) {
	const line = "history: [1, \"2\", true, null, plain]\n"
	it, err := codec.Parse(docWithExtras(line))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	out := codec.Serialize(it)
	if !strings.Contains(out, line) {
		t.Fatalf("want %q emitted verbatim, got:\n%s", line, out)
	}
	again, err := codec.Parse(out)
	if err != nil {
		t.Fatalf("re-parse: %v", err)
	}
	if out2 := codec.Serialize(again); out2 != out {
		t.Fatalf("serialize not idempotent\n--- first ---\n%s\n--- second ---\n%s", out, out2)
	}
}
