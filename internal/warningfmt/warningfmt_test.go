package warningfmt

import (
	"testing"

	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func TestMessage(t *testing.T) {
	t.Parallel()
	cases := []struct {
		warning datamodel.Warning
		want    string
	}{
		{datamodel.Warning{Code: datamodel.WarnIndexFallback, Args: []string{"boom"}}, "index unavailable (boom), using linear scan"},
		{datamodel.Warning{Code: datamodel.WarnNoActiveSprint}, "no active sprint set; sprint=active matches nothing (run `kira sprint activate <key>`)"},
		{datamodel.Warning{Code: datamodel.WarnCloseUnknown, Args: []string{"KIRA-9", "Kira-Closes"}}, "unknown ticket KIRA-9 in Kira-Closes (hint: run `kira config explain` to see the configured close trailer)"},
		{datamodel.Warning{Code: datamodel.WarnCloseFailed, Args: []string{"KIRA-2", "boom"}}, "failed to close KIRA-2: boom"},
		{datamodel.Warning{Code: datamodel.WarnLiteral, Args: []string{"skipped a file"}}, "skipped a file"},
		{datamodel.Warning{Code: datamodel.WarnOrphanType, Args: []string{"KIRA-3", "bug"}}, `KIRA-3 has no workflow for type "bug"; it is read-only until a workflow is configured`},
		{datamodel.Warning{Code: "future_warning", Args: []string{"one", "two"}}, "future_warning: one: two"},
		{datamodel.Warning{Code: "future_warning"}, "future_warning"},
		{datamodel.Warning{Code: datamodel.WarnIndexFallback}, "index unavailable (), using linear scan"},
		{datamodel.Warning{Code: datamodel.WarnCloseFailed, Args: []string{"KIRA-2"}}, "failed to close KIRA-2: "},
		{datamodel.Warning{Code: datamodel.WarnLiteral}, ""},
	}
	for _, tc := range cases {
		t.Run(string(tc.warning.Code), func(t *testing.T) {
			if got := Message(tc.warning); got != tc.want {
				t.Errorf("Message(%+v) = %q, want %q", tc.warning, got, tc.want)
			}
		})
	}
}

func TestMessageExhaustive(t *testing.T) {
	t.Parallel()
	for _, code := range datamodel.WarnCodes {
		w := datamodel.Warning{Code: code, Args: []string{"a", "b"}}
		if got := Message(w); got == "" || got == string(code)+": a: b" {
			t.Errorf("Message(%q) has no dedicated message; add a case", code)
		}
	}
}

// Message pads short Args through a fixed-size array, so every code must still
// render with fewer args than it uses. This is the canary for that guard: index
// w.Args directly instead and this panics.
func TestMessageRendersWithMissingArgs(t *testing.T) {
	t.Parallel()
	for _, code := range datamodel.WarnCodes {
		for _, args := range [][]string{nil, {"one"}} {
			got := Message(datamodel.Warning{Code: code, Args: args})
			// A literal warning's message IS its argument, so with none it is
			// legitimately empty. Every other code carries its own prose.
			if got == "" && code != datamodel.WarnLiteral {
				t.Errorf("code %q with %d args: rendered empty", code, len(args))
			}
		}
	}
}
