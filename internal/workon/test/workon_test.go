package workon_test

import (
	"testing"

	"github.com/shivamshivanshu/kira/internal/workon"
)

func TestSlugCasing(t *testing.T) {
	t.Parallel()
	cases := []struct {
		title string
		sep   string
		want  string
	}{
		{"Add Widget!", "-", "add-widget"},
		{"Add Widget!", "_", "add_widget"},
		{"  Fix   the BUG  ", "-", "fix-the-bug"},
		{"a/b:c", "-", "a-b-c"},
		{"---", "-", ""},
	}
	for _, c := range cases {
		if got := workon.Slug(c.title, c.sep); got != c.want {
			t.Errorf("Slug(%q, %q) = %q, want %q", c.title, c.sep, got, c.want)
		}
	}
}

func TestRenderBranch(t *testing.T) {
	t.Parallel()
	got := workon.RenderBranch("{key}/{number}-{slug}", "KIRA", "KIRA-142", "Fix the bug", "-")
	if want := "kira/kira-142-fix-the-bug"; got != want {
		t.Fatalf("RenderBranch = %q, want %q", got, want)
	}
	got = workon.RenderBranch("{key}/{number}-{slug}", "KIRA", "KIRA-142", "Fix the bug", "_")
	if want := "kira/kira_142-fix_the_bug"; got != want {
		t.Fatalf("RenderBranch snake = %q, want %q", got, want)
	}
}

func TestRenderWorktreeDir(t *testing.T) {
	t.Parallel()
	got := workon.RenderWorktreeDir("../{repo}-wt/{branch}", "kira", "kira/kira-142-fix", "KIRA", "KIRA-142")
	if want := "../kira-wt/kira-kira-142-fix"; got != want {
		t.Fatalf("RenderWorktreeDir = %q, want %q", got, want)
	}
}

func TestMatchBranch(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		branches []string
		pattern  string
		number   string
		sep      string
		want     string
		ok       bool
	}{
		{"kebab slug rerun matches existing", []string{"main", "kira/kira-142-original-title", "kira/kira-2-other"}, "{key}/{number}-{slug}", "KIRA-142", "-", "kira/kira-142-original-title", true},
		{"kebab bare prefix matches", []string{"kira/kira-142"}, "{key}/{number}-{slug}", "KIRA-142", "-", "kira/kira-142", true},
		{"kebab numeric superstring rejected", []string{"kira/kira-1420-x"}, "{key}/{number}-{slug}", "KIRA-142", "-", "", false},
		{"kebab absent number rejected", []string{"main", "kira/kira-142-original-title"}, "{key}/{number}-{slug}", "KIRA-9", "-", "", false},
		{"snake slug matches", []string{"main", "kira/kira_142-fix_the_bug"}, "{key}/{number}-{slug}", "KIRA-142", "_", "kira/kira_142-fix_the_bug", true},
		{"snake bare matches", []string{"kira/kira_142"}, "{key}/{number}-{slug}", "KIRA-142", "_", "kira/kira_142", true},
		{"snake numeric superstring rejected", []string{"kira/kira_1420-x"}, "{key}/{number}-{slug}", "KIRA-142", "_", "", false},
		{"snake config matches kebab branch", []string{"kira/kira-142-old-title"}, "{key}/{number}-{slug}", "KIRA-142", "_", "kira/kira-142-old-title", true},
		{"kebab config matches snake branch", []string{"kira/kira_142-old_title"}, "{key}/{number}-{slug}", "KIRA-142", "-", "kira/kira_142-old_title", true},
		{"slugless exact match", []string{"kira/kira-142"}, "{key}/{number}", "KIRA-142", "-", "kira/kira-142", true},
		{"slugless superstring rejected", []string{"kira/kira-1420"}, "{key}/{number}", "KIRA-142", "-", "", false},
		{"slugless suffixed branch rejected", []string{"kira/kira-142-fix"}, "{key}/{number}", "KIRA-142", "-", "", false},
		{"slug-first matches", []string{"main", "fix-the-bug-kira-142"}, "{slug}-{number}", "KIRA-142", "-", "fix-the-bug-kira-142", true},
		{"slug-first never falls back to first branch", []string{"main", "develop"}, "{slug}-{number}", "KIRA-142", "-", "", false},
		{"number only exact", []string{"kira-142"}, "{number}", "KIRA-142", "-", "kira-142", true},
		{"number only superstring rejected", []string{"kira-1420"}, "{number}", "KIRA-142", "-", "", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, ok := workon.MatchBranch(c.branches, c.pattern, "KIRA", c.number, c.sep)
			if ok != c.ok || got != c.want {
				t.Fatalf("MatchBranch(%v, %q, %q, sep=%q) = (%q, %v), want (%q, %v)", c.branches, c.pattern, c.number, c.sep, got, ok, c.want, c.ok)
			}
		})
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
		{"kira/kira_142-fix", "KIRA-142", true},
		{"kira/kira_142", "KIRA-142", true},
		{"fix_the_bug_kira_142", "KIRA-142", true},
		{"akira-142", "", false},
		{"kira-142abc", "", false},
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

func TestParseActiveRejectsCorruptPointers(t *testing.T) {
	t.Parallel()
	cases := []string{
		"  \n",
		`{"ticket":"","branch":"kira/kira-1-x"}`,
		`{}`,
		`{"tick`,
		`"quoted"`,
		"not a bare token",
	}
	for _, c := range cases {
		if got, ok := workon.ParseActive([]byte(c)); ok {
			t.Errorf("ParseActive(%q) = (%+v, true), want ok=false", c, got)
		}
	}
}
