package codec_test

import (
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
		"parent: null\n" +
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
	wantScalars := map[string]datamodel.Rendered{"team": "infra"}
	wantLists := map[string][]datamodel.Rendered{"watchers": {"alice", "bob"}}
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

// A structured extra -- a mapping, or a sequence whose elements are not all
// scalars -- rides as an opaque block. kira never interprets it, so a rewrite
// must hand it back unchanged, and a second rewrite must change nothing at all.
func TestNestedExtrasRoundTripAsBlocks(t *testing.T) {
	tests := []struct {
		name  string
		lines string
		want  datamodel.Rendered
	}{
		{
			name:  "mapping",
			lines: "sla:\n  response: 4h\n  resolve: 24h\n",
			want:  "response: 4h\nresolve: 24h",
		},
		{
			name:  "sequence of mappings",
			lines: "steps:\n  - name: build\n    cmd: make\n  - name: test\n",
			want:  "- name: build\n  cmd: make\n- name: test",
		},
		{
			name:  "mapping holding a sequence of mappings three levels down",
			lines: "policy:\n  escalation:\n    - after: 1h\n      notify: [oncall, lead]\n    - after: 4h\n  owner: sre\n",
			want:  "escalation:\n  - after: 1h\n    notify: [oncall, lead]\n  - after: 4h\nowner: sre",
		},
		{
			name:  "flow sequence with a mapping element",
			lines: "weird: [{a: b}]\n",
			want:  "[{a: b}]",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, _, _ := strings.Cut(tt.lines, ":")
			it, err := codec.Parse(docWithExtras(tt.lines))
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			if got := it.ExtraBlocks[key]; got != tt.want {
				t.Fatalf("ExtraBlocks[%q] = %q, want %q", key, got, tt.want)
			}
			if _, isScalar := it.ExtraScalars[key]; isScalar {
				t.Fatalf("%q must not also land in ExtraScalars", key)
			}
			if _, isList := it.ExtraLists[key]; isList {
				t.Fatalf("%q must not also land in ExtraLists", key)
			}

			out := codec.Serialize(it)
			if !strings.Contains(out, indentedBlock(key, string(tt.want))) {
				t.Fatalf("serialized output lost the block for %q:\n%s", key, out)
			}
			again, err := codec.Parse(out)
			if err != nil {
				t.Fatalf("re-parse: %v", err)
			}
			if !reflect.DeepEqual(again.ExtraBlocks, it.ExtraBlocks) {
				t.Fatalf("re-parsed blocks = %+v, want %+v", again.ExtraBlocks, it.ExtraBlocks)
			}
			if out2 := codec.Serialize(again); out2 != out {
				t.Fatalf("serialize not idempotent\n--- first ---\n%s\n--- second ---\n%s", out, out2)
			}
		})
	}
}

func indentedBlock(key, block string) string {
	out := key + ":\n"
	for _, line := range strings.Split(block, "\n") {
		out += "  " + line + "\n"
	}
	return out
}

// Storing a block with the key's own indentation stripped is what lets merge
// treat two files that say the same thing as unchanged, however each was
// indented.
func TestBlockStorageIgnoresSourceIndentation(t *testing.T) {
	twoSpace, err := codec.Parse(docWithExtras("sla:\n  response: 4h\n  nested:\n    x: 1\n"))
	if err != nil {
		t.Fatalf("parse two-space: %v", err)
	}
	fourSpace, err := codec.Parse(docWithExtras("sla:\n    response: 4h\n    nested:\n        x: 1\n"))
	if err != nil {
		t.Fatalf("parse four-space: %v", err)
	}
	if !reflect.DeepEqual(twoSpace.ExtraBlocks, fourSpace.ExtraBlocks) {
		t.Fatalf("blocks differ by source indentation: %+v vs %+v", twoSpace.ExtraBlocks, fourSpace.ExtraBlocks)
	}
	if codec.Serialize(twoSpace) != codec.Serialize(fourSpace) {
		t.Fatal("two files differing only in indentation must serialize identically")
	}
}

func TestScalarSequenceStaysAList(t *testing.T) {
	for _, lines := range []string{"tags: [a, b]\n", "tags:\n  - a\n  - b\n"} {
		it, err := codec.Parse(docWithExtras(lines))
		if err != nil {
			t.Fatalf("parse %q: %v", lines, err)
		}
		if _, isBlock := it.ExtraBlocks["tags"]; isBlock {
			t.Fatalf("%q: a sequence of scalars must stay in ExtraLists, got a block", lines)
		}
		want := map[string][]datamodel.Rendered{"tags": {"a", "b"}}
		if !reflect.DeepEqual(it.ExtraLists, want) {
			t.Fatalf("%q: ExtraLists = %+v, want %+v", lines, it.ExtraLists, want)
		}
		if !strings.Contains(codec.Serialize(it), "tags: [a, b]\n") {
			t.Fatalf("%q: a scalar list must still serialize to flow form", lines)
		}
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
	draft := "---\ntitle: t\nteam: infra\nwatchers: [alice, bob]\nsla:\n  response: 4h\n---\nbody\n"
	scalars, lists, blocks, err := codec.ExtraFrontmatter(draft)
	if err != nil {
		t.Fatalf("ExtraFrontmatter: %v", err)
	}
	if !reflect.DeepEqual(scalars, map[string]datamodel.Rendered{"team": "infra"}) {
		t.Fatalf("scalars = %+v, want team only (title is builtin)", scalars)
	}
	if !reflect.DeepEqual(lists, map[string][]datamodel.Rendered{"watchers": {"alice", "bob"}}) {
		t.Fatalf("lists = %+v, want watchers", lists)
	}
	if !reflect.DeepEqual(blocks, map[string]datamodel.Rendered{"sla": "response: 4h"}) {
		t.Fatalf("blocks = %+v, want sla", blocks)
	}

	if _, _, _, err := codec.ExtraFrontmatter("---\ntitle: t\nanchored: &a 1\naliasing:\n  x: *a\n---\n"); err == nil {
		t.Fatal("a value kira cannot write back must be reported rather than silently dropped")
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
