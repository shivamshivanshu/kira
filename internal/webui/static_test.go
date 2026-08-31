package webui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func TestStaticAssetCacheDoesNotCacheDashboardData(t *testing.T) {
	d := newDashboardTest(t, Options{})
	url := assetURL("css/main.css")
	res := d.get(url)
	if res.Code != http.StatusOK || !strings.Contains(res.Header().Get("Cache-Control"), "immutable") || res.Header().Get("ETag") == "" {
		t.Fatalf("versioned asset is not cacheable: %d %v", res.Code, res.Header())
	}
	req := httptest.NewRequest(http.MethodGet, "http://localhost:7432"+url, nil)
	req.Header.Set("If-None-Match", res.Header().Get("ETag"))
	w := httptest.NewRecorder()
	d.handler.ServeHTTP(w, req)
	if w.Code != http.StatusNotModified || w.Body.Len() != 0 {
		t.Fatalf("cached asset transferred again: status=%d bytes=%d", w.Code, w.Body.Len())
	}
	for _, path := range []string{"/", assetURL("missing.css"), assetURL("css/"), "/static/old/css/main.css"} {
		res := d.get(path)
		if res.Header().Get("Cache-Control") != "no-store" {
			t.Errorf("%s cached live data or a missing asset", path)
		}
		if path != "/" && res.Code != http.StatusNotFound {
			t.Errorf("%s returned %d, want404", path, res.Code)
		}
	}
}

func TestAssetVersionChangesWithContentAndPaths(t *testing.T) {
	files := fstest.MapFS{
		"static/css/main.css":   {Data: []byte("@import './tokens.css';")},
		"static/css/tokens.css": {Data: []byte("original")},
	}
	version, err := assetFingerprint(files)
	if err != nil {
		t.Fatal(err)
	}
	if again, err := assetFingerprint(files); err != nil || again != version {
		t.Fatalf("asset version is not stable: %s, %v", again, err)
	}
	files["static/css/tokens.css"].Data = []byte("updated")
	changed, err := assetFingerprint(files)
	if err != nil || changed == version {
		t.Fatal("changing an imported asset must invalidate the bundle URL")
	}
	files["static/css/theme.css"] = files["static/css/tokens.css"]
	delete(files, "static/css/tokens.css")
	if renamed, err := assetFingerprint(files); err != nil || renamed == changed {
		t.Fatal("renaming an asset must invalidate the bundle URL")
	}
}
