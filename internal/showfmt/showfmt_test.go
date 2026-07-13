package showfmt_test

import (
	"testing"

	"github.com/shivamshivanshu/kira/internal/showfmt"
)

var sample = showfmt.Item{ID: "T1", Number: "KIRA-140", Title: "Fix snapshot dedup"}

func TestFormat(t *testing.T) {
	cases := []struct {
		form showfmt.Form
		want string
	}{
		{showfmt.FormID, "T1"},
		{showfmt.FormNumber, "KIRA-140"},
		{showfmt.FormNumberTitle, "KIRA-140 Fix snapshot dedup"},
		{showfmt.FormMarkdown, "[KIRA-140 Fix snapshot dedup](KIRA-140)"},
		{showfmt.FormBranch, "kira-140-fix-snapshot-dedup"},
	}
	for _, tc := range cases {
		got, err := showfmt.Format(tc.form, sample)
		if err != nil {
			t.Fatalf("%s: %v", tc.form, err)
		}
		if got != tc.want {
			t.Errorf("%s = %q, want %q", tc.form, got, tc.want)
		}
	}
}

func TestBranchSlugCollapsesPunctuation(t *testing.T) {
	got, _ := showfmt.Format(showfmt.FormBranch, showfmt.Item{Number: "KIRA-9", Title: "Fix: race (in) merge!!"})
	if want := "kira-9-fix-race-in-merge"; got != want {
		t.Fatalf("branch = %q, want %q", got, want)
	}
}

func TestUnknownFormErrors(t *testing.T) {
	if _, err := showfmt.Format(showfmt.Form("bogus"), sample); err == nil {
		t.Fatal("expected error for unknown form")
	}
}
