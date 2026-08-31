package showfmt_test

import (
	"testing"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/showfmt"
)

var sample = showfmt.Item{ID: "T1", Number: "KIRA-140", Title: "Fix snapshot dedup"}

var kebab = datamodel.Workon{BranchPattern: "{key}/{number}-{slug}", Casing: datamodel.CasingKebab}

func TestFormat(t *testing.T) {
	cases := []struct {
		form showfmt.Form
		want string
	}{
		{showfmt.FormID, "T1"},
		{showfmt.FormNumber, "KIRA-140"},
		{showfmt.FormNumberTitle, "KIRA-140 Fix snapshot dedup"},
		{showfmt.FormMarkdown, "[KIRA-140 Fix snapshot dedup](KIRA-140)"},
		{showfmt.FormBranch, "kira/kira-140-fix-snapshot-dedup"},
	}
	for _, tc := range cases {
		got, err := showfmt.Format(tc.form, sample, kebab)
		if err != nil {
			t.Fatalf("%s: %v", tc.form, err)
		}
		if got != tc.want {
			t.Errorf("%s = %q, want %q", tc.form, got, tc.want)
		}
	}
}

func TestBranchFollowsWorkonConfig(t *testing.T) {
	cases := []struct {
		name string
		w    datamodel.Workon
		want string
	}{
		{"snake", datamodel.Workon{BranchPattern: "{key}/{number}-{slug}", Casing: datamodel.CasingSnake}, "kira/kira_140-fix_snapshot_dedup"},
		{"custom pattern", datamodel.Workon{BranchPattern: "ticket/{number}", Casing: datamodel.CasingKebab}, "ticket/kira-140"},
	}
	for _, tc := range cases {
		got, _ := showfmt.Format(showfmt.FormBranch, sample, tc.w)
		if got != tc.want {
			t.Errorf("%s: branch = %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestBranchSlugCollapsesPunctuation(t *testing.T) {
	got, _ := showfmt.Format(showfmt.FormBranch, showfmt.Item{Number: "KIRA-9", Title: "Fix: race (in) merge!!"}, kebab)
	if want := "kira/kira-9-fix-race-in-merge"; got != want {
		t.Fatalf("branch = %q, want %q", got, want)
	}
}

func TestUnknownFormErrors(t *testing.T) {
	if _, err := showfmt.Format(showfmt.Form("bogus"), sample, kebab); err == nil {
		t.Fatal("expected error for unknown form")
	}
}
