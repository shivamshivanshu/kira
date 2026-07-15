package ptr_test

import (
	"testing"

	"github.com/shivamshivanshu/kira/internal/ptr"
)

func TestDeref(t *testing.T) {
	s := "a"
	if got := ptr.Deref(&s); got != "a" {
		t.Errorf("Deref(&s) = %q, want %q", got, "a")
	}
	if got := ptr.Deref[string](nil); got != "" {
		t.Errorf("Deref(nil) = %q, want empty", got)
	}
}

func TestDerefOr(t *testing.T) {
	s := "a"
	empty := ""
	cases := []struct {
		name string
		p    *string
		want string
	}{
		{"nil", nil, "fallback"},
		{"empty", &empty, "fallback"},
		{"set", &s, "a"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ptr.DerefOr(tc.p, "fallback"); got != tc.want {
				t.Errorf("DerefOr() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestTo(t *testing.T) {
	p := ptr.To("a")
	if p == nil || *p != "a" {
		t.Errorf("To(%q) = %v, want pointer to %q", "a", p, "a")
	}
}

func TestNilIfEmpty(t *testing.T) {
	if got := ptr.NilIfEmpty(""); got != nil {
		t.Errorf("NilIfEmpty(\"\") = %v, want nil", got)
	}
	if got := ptr.NilIfEmpty("a"); got == nil || *got != "a" {
		t.Errorf("NilIfEmpty(%q) = %v, want pointer to %q", "a", got, "a")
	}
}

func TestEqual(t *testing.T) {
	x, y := "a", "b"
	xCopy := "a"
	cases := []struct {
		a, b *string
		want bool
	}{
		{nil, nil, true},
		{&x, nil, false},
		{nil, &x, false},
		{&x, &xCopy, true},
		{&x, &y, false},
	}
	for _, tc := range cases {
		if got := ptr.Equal(tc.a, tc.b); got != tc.want {
			t.Errorf("Equal(%v, %v) = %v, want %v", tc.a, tc.b, got, tc.want)
		}
	}
}
