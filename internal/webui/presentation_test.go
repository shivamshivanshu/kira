package webui

import (
	"html"
	"io/fs"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strings"
	"testing"

	"github.com/shivamshivanshu/kira/internal/query"
)

func TestAppearanceAndQueryControls(t *testing.T) {
	d := newDashboardTest(t, Options{})
	res := d.get("/?view=board&state=TODO&owner=@me&priority=P1&label=web")
	if res.Code != http.StatusOK {
		t.Fatalf("filtered dashboard returned %d", res.Code)
	}
	body := res.Body.String()
	for _, id := range []string{"theme-preference", "density-preference", "sidebar-toggle", "command-palette", "palette-query", "state-filter", "owner-filter", "priority-filter", "label-filter"} {
		if !strings.Contains(body, `id="`+id+`"`) {
			t.Errorf("missing control %s", id)
		}
	}
	if strings.Index(body, assetURL("js/appearance.js")) > strings.Index(body, assetURL("css/main.css")) {
		t.Fatal("theme preferences must load before styles to avoid a light flash")
	}
	if strings.Contains(body, "<style") || strings.Contains(body, "style=") || strings.Contains(body, "<script>") {
		t.Fatal("presentation must use external assets, not inline styles or scripts")
	}
	examples := regexp.MustCompile(`data-query-example="([^"]+)"`).FindAllStringSubmatch(body, -1)
	if len(examples) < 4 {
		t.Fatal("query help must include working examples")
	}
	for _, example := range examples {
		expression := html.UnescapeString(example[1])
		if _, err := query.Compile(expression, query.Options{Me: "tester", Priorities: d.cfg.Priorities.Values}); err != nil {
			t.Errorf("example %q differs from core query syntax: %v", expression, err)
		}
	}
	match := regexp.MustCompile(`href="([^"]*)">Clear all`).FindStringSubmatch(body)
	if len(match) != 2 {
		t.Fatal("missing clear-filter link")
	}
	u, err := url.Parse(html.UnescapeString(match[1]))
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"q", "filter", "board", "type", "sprint", "state", "owner", "priority", "label"} {
		if u.Query().Has(key) {
			t.Errorf("clear filters retained %s", key)
		}
	}
}

func TestStaticImportsStayInsideVersionedBundle(t *testing.T) {
	imports := regexp.MustCompile(`(?:from\s+|@import\s+)["']([^"']+)["']`)
	err := fs.WalkDir(assets, "static", func(name string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		data, err := assets.ReadFile(name)
		if err != nil {
			return err
		}
		for _, match := range imports.FindAllStringSubmatch(string(data), -1) {
			if !strings.HasPrefix(match[1], "./") {
				t.Errorf("%s import %s escapes versioned asset prefix", name, match[1])
			}
			target := path.Join(path.Dir(name), match[1])
			if _, err := fs.Stat(assets, target); err != nil {
				t.Errorf("%s imports missing asset %s", name, target)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
