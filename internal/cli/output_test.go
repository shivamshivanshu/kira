package cli

import (
	"testing"

	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func TestRenderWarning(t *testing.T) {
	cases := []struct {
		warning datamodel.Warning
		want    string
	}{
		{datamodel.Warning{Code: datamodel.WarnIndexFallback, Args: []string{"boom"}}, "index unavailable (boom), using linear scan"},
		{datamodel.Warning{Code: datamodel.WarnNoActiveSprint}, "no active sprint set; sprint=active matches nothing (run `kira sprint activate <key>`)"},
		{datamodel.Warning{Code: datamodel.WarnCloseUnknown, Args: []string{"KIRA-9", "Kira-Closes"}}, "unknown ticket KIRA-9 in Kira-Closes"},
		{datamodel.Warning{Code: datamodel.WarnCloseFailed, Args: []string{"KIRA-2", "boom"}}, "failed to close KIRA-2: boom"},
		{datamodel.Warning{Code: datamodel.WarnLiteral, Args: []string{"skipped a file"}}, "skipped a file"},
	}
	for _, c := range cases {
		if got := renderWarning(c.warning); got != c.want {
			t.Errorf("renderWarning(%+v) = %q, want %q", c.warning, got, c.want)
		}
	}
}

func TestRenderSkew(t *testing.T) {
	sk := &datamodel.Skew{Ref: "KIRA-1", At: "HEAD~2", AtID: "01AAA", NowID: "01BBB"}
	want := "KIRA-1 at HEAD~2 is 01AAA; currently it is a different item (01BBB)"
	if got := renderSkew(sk); got != want {
		t.Errorf("renderSkew = %q, want %q", got, want)
	}
}
