package webui

import (
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/gitx"
)

func TestCommitPreviewUsesLinkedPatch(t *testing.T) {
	d := newDashboardTest(t, Options{})
	item := d.create(t, core.CreateOpts{Title: "Linked work"})
	unrelated := d.create(t, core.CreateOpts{Title: "Unrelated work"})
	if _, err := d.store.CommitKira(d.cfg); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(d.store.Root(), "example.html"), []byte("<script>alert('code, not markup')</script>\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	repo := gitx.Repo{Dir: d.store.Root()}
	if err := repo.Stage("example.html"); err != nil {
		t.Fatal(err)
	}
	if err := repo.CommitParts("Linked implementation", "Kira-Ticket: "+item.Number); err != nil {
		t.Fatal(err)
	}
	sha, err := repo.Output("rev-parse", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	q := url.Values{"item": {item.ID}, "commit": {sha}}
	res := d.get("/?" + q.Encode())
	if res.Code != http.StatusOK {
		t.Fatalf("preview status %d: %s", res.Code, res.Body.String())
	}
	body := res.Body.String()
	for _, want := range []string{"Linked implementation", "diff --git a/example.html b/example.html", "&lt;script&gt;", "diff-add", "Linked work"} {
		if !strings.Contains(body, want) {
			t.Errorf("preview missing %q", want)
		}
	}
	if strings.Contains(body, "<script>alert") {
		t.Fatal("commit source was rendered as executable HTML")
	}
	for _, fields := range []url.Values{
		{"commit": {sha}},
		{"item": {unrelated.ID}, "commit": {sha}},
		{"item": {item.ID}, "commit": {"HEAD"}},
		{"item": {item.ID}, "commit": {"--output=/tmp/patch"}},
	} {
		if res := d.get("/?" + fields.Encode()); res.Code != http.StatusBadRequest {
			t.Errorf("invalid commit preview %v returned %d", fields, res.Code)
		}
	}
}
