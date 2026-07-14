package codec_test

import (
	"errors"
	"math/rand"
	"reflect"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/shivamshivanshu/kira/internal/codec"
	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func TestEmitListFlowSafeScalars(t *testing.T) {
	cases := []struct {
		name string
		xs   []string
		want string
	}{
		{"empty", nil, "[]"},
		{"plain", []string{"bug", "orderbook"}, "[bug, orderbook]"},
		{"comma", []string{"a, b"}, "['a, b']"},
		{"closing bracket", []string{"x]y"}, "['x]y']"},
		{"opening bracket", []string{"x[y"}, "['x[y']"},
		{"braces", []string{"a{b", "a}b"}, "['a{b', 'a}b']"},
		{"keywords stay quoted", []string{"null", "true", "123"}, `["null", "true", "123"]`},
		{"empty string stays double-quoted", []string{""}, `[""]`},
		{"mixed", []string{"safe", "a, b"}, "[safe, 'a, b']"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := codec.EmitList(tc.xs)
			if got != tc.want {
				t.Fatalf("EmitList(%q) = %s, want %s", tc.xs, got, tc.want)
			}
			var back []string
			if err := yaml.Unmarshal([]byte(got), &back); err != nil {
				t.Fatalf("emitted list is not valid YAML: %v", err)
			}
			if len(tc.xs) == 0 {
				tc.xs, back = nil, nil
			}
			if !reflect.DeepEqual(back, tc.xs) {
				t.Fatalf("round-trip = %q, want %q", back, tc.xs)
			}
		})
	}
}

func TestEmitListItemRoundTrip(t *testing.T) {
	for _, xs := range [][]string{
		{"a, b"},
		{"x]y", "x[y"},
		{"safe", "a, b", "null", ""},
	} {
		it, err := codec.Parse(readExample(t))
		if err != nil {
			t.Fatalf("parse: %v", err)
		}
		it.Labels = xs
		out := codec.Serialize(it)
		got, err := codec.Parse(out)
		if err != nil {
			t.Fatalf("labels %q: re-parse failed: %v\n%s", xs, err, out)
		}
		if !reflect.DeepEqual(got.Labels, xs) {
			t.Fatalf("labels %q round-tripped as %q", xs, got.Labels)
		}
		if codec.Serialize(got) != out {
			t.Fatalf("labels %q: serialize not idempotent", xs)
		}
	}
}

func TestEmitListLongListStaysSingleLine(t *testing.T) {
	labels := make([]string, 12)
	for i := range labels {
		labels[i] = strings.Repeat("verylonglabel", 2) + string(rune('a'+i))
	}
	labels = append(labels, "with, comma", "and ]bracket",
		strings.Repeat("spacey words past the emitter width ", 4)+"end")

	it, err := codec.Parse(readExample(t))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	it.Labels = labels
	out := codec.Serialize(it)

	var line string
	for _, l := range strings.Split(out, "\n") {
		if strings.HasPrefix(l, "labels: ") {
			line = l
			break
		}
	}
	if !strings.HasSuffix(line, "]") {
		t.Fatalf("labels must stay on a single line, got %q", line)
	}
	got, err := codec.Parse(out)
	if err != nil {
		t.Fatalf("re-parse: %v", err)
	}
	if !reflect.DeepEqual(got.Labels, labels) {
		t.Fatalf("labels = %q, want %q", got.Labels, labels)
	}
	if codec.Serialize(got) != out {
		t.Fatal("serialize not idempotent for a long list")
	}
}

func emitScalarReference(tb testing.TB, s string) string {
	tb.Helper()
	plain, err := yaml.Marshal(s)
	if err == nil && strings.TrimSuffix(string(plain), "\n") == s {
		return s
	}
	quoted, err := yaml.Marshal(&yaml.Node{Kind: yaml.ScalarNode, Style: yaml.DoubleQuotedStyle, Value: s})
	if err != nil {
		tb.Fatalf("marshal %q: %v", s, err)
	}
	return strings.TrimSuffix(string(quoted), "\n")
}

func TestEmitScalarMatchesYAMLReference(t *testing.T) {
	corpus := append([]string{}, tokens...)
	corpus = append(corpus,
		"null", "Null", "NULL", "~", "true", "True", "TRUE", "false", "False", "FALSE",
		"yes", "Yes", "YES", "no", "No", "NO", "on", "On", "ON", "off", "Off", "OFF",
		"y", "Y", "n", "N", "nan", "inf", "2026-07-20", "0x1F", "1e3", "-dash", "_под",
	)
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_-,:[]{}#'\" ."
	r := rand.New(rand.NewSource(7))
	for i := 0; i < 5000; i++ {
		n := 1 + r.Intn(8)
		b := make([]byte, n)
		for j := range b {
			b[j] = charset[r.Intn(len(charset))]
		}
		corpus = append(corpus, string(b))
	}
	for _, s := range corpus {
		if got, want := codec.EmitScalar(s), emitScalarReference(t, s); got != want {
			t.Fatalf("EmitScalar(%q) = %s, want %s", s, got, want)
		}
		var back []string
		if err := yaml.Unmarshal([]byte(codec.EmitList([]string{s})), &back); err != nil {
			t.Fatalf("EmitList([%q]) not valid YAML: %v", s, err)
		}
		if len(back) != 1 || back[0] != s {
			t.Fatalf("EmitList([%q]) round-tripped as %q", s, back)
		}
	}
}

func TestParseDuplicateKeyReported(t *testing.T) {
	src := strings.Replace(readExample(t), "state: IN_PROGRESS\n", "state: IN_PROGRESS\nstate: DONE\n", 1)
	_, err := codec.Parse(src)
	var pe *datamodel.ParseError
	if !errors.As(err, &pe) {
		t.Fatalf("want *datamodel.ParseError, got %v", err)
	}
	if !strings.Contains(err.Error(), `field "state": duplicate key`) {
		t.Fatalf("error must name the duplicate key: %v", err)
	}
}

func TestParseNonMappingFrontmatter(t *testing.T) {
	for _, src := range []string{
		"---\n- a\n- b\n---\nbody\n",
		"---\njust a scalar\n---\nbody\n",
	} {
		_, err := codec.Parse(src)
		if err == nil || !strings.Contains(err.Error(), "mapping") {
			t.Errorf("Parse(%q) error = %v, want a clear non-mapping error", src, err)
		}
	}
}

func TestParseEmptyFrontmatter(t *testing.T) {
	it, err := codec.Parse("---\n---\nbody\n")
	var pe *datamodel.ParseError
	if !errors.As(err, &pe) {
		t.Fatalf("empty frontmatter must yield missing-field errors, got %v", err)
	}
	if strings.Contains(err.Error(), "fence") {
		t.Fatalf("empty frontmatter must not be a fence error: %v", err)
	}
	if it == nil || it.Body != "body\n" {
		t.Fatalf("body must survive empty frontmatter, got %+v", it)
	}
}

func TestParseTerminalFenceWithoutTrailingNewline(t *testing.T) {
	src := readExample(t)
	front := src[:strings.Index(src, "\n---\n")+1] + "---"
	it, err := codec.Parse(front)
	if err != nil {
		t.Fatalf("terminal fence without trailing newline must parse: %v", err)
	}
	if it.Body != "" {
		t.Fatalf("body = %q, want empty", it.Body)
	}
}
