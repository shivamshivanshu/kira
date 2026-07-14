package workon_test

import (
	"testing"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/workon"
)

func TestSlugCasing(t *testing.T) {
	t.Parallel()
	cases := []struct {
		title  string
		casing datamodel.Casing
		want   string
	}{
		{"Add Widget!", datamodel.CasingKebab, "add-widget"},
		{"Add Widget!", datamodel.CasingSnake, "add_widget"},
		{"  Fix   the BUG  ", datamodel.CasingKebab, "fix-the-bug"},
		{"a/b:c", datamodel.CasingKebab, "a-b-c"},
		{"---", datamodel.CasingKebab, ""},
	}
	for _, c := range cases {
		if got := workon.Slug(c.title, c.casing); got != c.want {
			t.Errorf("Slug(%q, %s) = %q, want %q", c.title, c.casing, got, c.want)
		}
	}
}

func TestRenderBranch(t *testing.T) {
	t.Parallel()
	got := workon.RenderBranch("{key}/{number}-{slug}", "KIRA", "KIRA-142", "Fix the bug", datamodel.CasingKebab)
	if want := "kira/kira-142-fix-the-bug"; got != want {
		t.Fatalf("RenderBranch = %q, want %q", got, want)
	}
	got = workon.RenderBranch("{key}/{number}-{slug}", "KIRA", "KIRA-142", "Fix the bug", datamodel.CasingSnake)
	if want := "kira/kira_142-fix_the_bug"; got != want {
		t.Fatalf("RenderBranch snake = %q, want %q", got, want)
	}
}

func TestMatchBranchIsIdempotentAcrossSlugs(t *testing.T) {
	t.Parallel()
	branches := []string{"main", "kira/kira-142-original-title", "kira/kira-2-other"}
	pat := "{key}/{number}-{slug}"

	// A re-run with a different title still matches the existing branch.
	if b, ok := workon.MatchBranch(branches, pat, "KIRA", "KIRA-142", datamodel.CasingKebab); !ok || b != "kira/kira-142-original-title" {
		t.Fatalf("expected match on existing branch, got %q ok=%v", b, ok)
	}
	// The bare prefix (no slug) matches too.
	if b, ok := workon.MatchBranch([]string{"kira/kira-142"}, pat, "KIRA", "KIRA-142", datamodel.CasingKebab); !ok || b != "kira/kira-142" {
		t.Fatalf("expected bare-prefix match, got %q ok=%v", b, ok)
	}
	// A different number does not match, and a numeric superstring does not either.
	if _, ok := workon.MatchBranch([]string{"kira/kira-1420-x"}, pat, "KIRA", "KIRA-142", datamodel.CasingKebab); ok {
		t.Fatal("KIRA-142 must not match branch for KIRA-1420")
	}
	if _, ok := workon.MatchBranch(branches, pat, "KIRA", "KIRA-9", datamodel.CasingKebab); ok {
		t.Fatal("unexpected match for absent number")
	}
}

func TestInferNumber(t *testing.T) {
	t.Parallel()
	keys := []string{"KIRA", "XYZ", "AB"}
	cases := []struct {
		branch string
		want   string
		ok     bool
	}{
		{"kira/kira-142-fix", "KIRA-142", true},
		{"KIRA-7", "KIRA-7", true},
		{"feature/no-ticket", "", false},
		{"kira/KIRA-9-caps", "KIRA-9", true},
		{"xyz/xyz-3-thing", "XYZ-3", true},
		{"ab/ab-5", "AB-5", true},
	}
	for _, c := range cases {
		got, ok := workon.InferNumber(c.branch, keys)
		if ok != c.ok || got != c.want {
			t.Errorf("InferNumber(%q) = (%q, %v), want (%q, %v)", c.branch, got, ok, c.want, c.ok)
		}
	}
}

func TestActivePointerRoundTrip(t *testing.T) {
	t.Parallel()
	p := workon.ActivePointer{Ticket: "01ULID", Branch: "kira/kira-1-x"}
	got, ok := workon.ParseActive(p.Marshal())
	if !ok || got != p {
		t.Fatalf("round-trip = (%+v, %v), want %+v", got, ok, p)
	}
}

func TestParseActiveLegacyBareULID(t *testing.T) {
	t.Parallel()
	got, ok := workon.ParseActive([]byte("01ULID\n"))
	if !ok || got.Ticket != "01ULID" || got.Branch != "" {
		t.Fatalf("legacy parse = (%+v, %v), want ticket only", got, ok)
	}
}

func TestParseActiveEmpty(t *testing.T) {
	t.Parallel()
	if _, ok := workon.ParseActive([]byte("  \n")); ok {
		t.Fatal("empty pointer must not parse")
	}
}
