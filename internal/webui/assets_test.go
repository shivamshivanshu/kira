package webui

import (
	"bytes"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strings"
	"testing"
)

func TestDashboardStaticAssets(t *testing.T) {
	d := newDashboardTest(t, Options{})
	page := d.get("/").Body.String()
	for _, entry := range []string{"css/main.css", "js/appearance.js", "js/main.mjs"} {
		if !strings.Contains(page, assetURL(entry)) {
			t.Errorf("dashboard does not reference %s", assetURL(entry))
		}
		if response := d.get(assetURL(entry)); response.Code != http.StatusOK {
			t.Errorf("entrypoint %s returned %d", entry, response.Code)
		}
	}
	err := fs.WalkDir(assets, "static", func(name string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		switch path.Ext(name) {
		case ".css", ".js", ".mjs":
		default:
			return nil
		}
		t.Run(name, func(t *testing.T) {
			response := d.get(assetURL(strings.TrimPrefix(name, "static/")))
			if response.Code != http.StatusOK {
				t.Fatalf("asset returned %d: %s", response.Code, response.Body.String())
			}
			mediaType, _, err := mime.ParseMediaType(response.Header().Get("Content-Type"))
			if err != nil {
				t.Fatal(err)
			}
			if path.Ext(name) == ".css" {
				if mediaType != "text/css" {
					t.Errorf("CSS content type = %q", mediaType)
				}
			} else if mediaType != "text/javascript" && mediaType != "application/javascript" {
				t.Errorf("JavaScript content type = %q", mediaType)
			}
			want, err := assets.ReadFile(name)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(response.Body.Bytes(), want) {
				t.Error("served content differs from embedded asset")
			}
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestMarkdownConcurrent(t *testing.T) {
	for range 16 {
		t.Run("conversion", func(t *testing.T) {
			t.Parallel()
			for range 20 {
				result, err := markdown("# Concurrent\n\n- [x] Safe **content**\n\n| A | B |\n| --- | --- |\n| 1 | 2 |\n")
				if err != nil || !strings.Contains(string(result), "<strong>content</strong>") || !strings.Contains(string(result), "<table>") {
					t.Fatalf("concurrent conversion: %v\n%s", err, result)
				}
			}
		})
	}
}
