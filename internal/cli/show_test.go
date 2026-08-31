package cli

import (
	"strings"
	"testing"

	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func TestRenderShowFormatsCommentsInsteadOfRawMarkers(t *testing.T) {
	res := &datamodel.ShowResult{
		Number: "KIRA-1",
		Title:  "T",
		State:  "TODO",
		Body: "Description text.\n\n" +
			"<!-- kira:comment id=01HAAAAAAAAAAAAAAAAAAAAAAA author=alice ts=2026-07-01T00:00:00Z -->\n" +
			"first note\n" +
			"<!-- /kira:comment -->\n",
		Comments: []datamodel.CommentView{
			{ID: "01HAAAAAAAAAAAAAAAAAAAAAAA", Author: "alice", Ts: "2026-07-01T00:00:00Z", Text: "first note"},
		},
	}
	var buf strings.Builder
	renderShow(&buf, res)
	out := buf.String()
	if strings.Contains(out, "kira:comment") {
		t.Errorf("raw comment markers leaked into human output:\n%s", out)
	}
	for _, want := range []string{"Description text.", "Comments", "alice  2026-07-01T00:00:00Z", "first note"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}
