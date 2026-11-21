package contract

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func TestReviewCommand(t *testing.T) {
	t.Parallel()
	dir := oneTicket(t)
	gitCmd(t, dir, "tag", "base")
	mustKira(t, dir, "move", "KIRA-1", "IN_PROGRESS")
	gitCmd(t, dir, "commit", "--allow-empty", "-m", "Finalize <tag>", "-m", "Kira-Closes: KIRA-1")
	out, stderr, code := kira(t, dir, "review", "--base", "base", "--json")
	if code != 0 || stderr != "" {
		t.Fatalf("review --json: exit %d, stderr %q", code, stderr)
	}
	checkSchemaConformance(t, out)
	var result datamodel.ReviewResult
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatal(err)
	}
	if len(result.Linked) != 1 || len(result.StateChanges) != 1 || len(result.Closes) != 1 {
		t.Fatalf("review JSON = %+v", result)
	}
	markdown, stderr, code := kira(t, dir, "review", "--base", "base")
	if code != 0 || stderr != "" {
		t.Fatalf("review: exit %d, stderr %q", code, stderr)
	}
	for _, text := range []string{"# Branch review", "## Linked tickets", "Finalize &lt;tag&gt;", "TODO → IN\\_PROGRESS", "## Kira-Closes preview"} {
		if !strings.Contains(markdown, text) {
			t.Errorf("review Markdown missing %q in %q", text, markdown)
		}
	}
}
