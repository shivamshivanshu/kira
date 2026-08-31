package webui

import (
	"crypto/sha256"
	"fmt"
	"io/fs"
	"net/http"
	"strings"
)

var assetVersion = func() string {
	version, err := assetFingerprint(assets)
	if err != nil {
		panic(fmt.Errorf("fingerprinting embedded assets: %w", err))
	}
	return version
}()

func assetFingerprint(files fs.FS) (string, error) {
	hash := sha256.New()
	err := fs.WalkDir(files, "static", func(name string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		data, err := fs.ReadFile(files, name)
		if err != nil {
			return err
		}
		_, _ = fmt.Fprintf(hash, "%s:%d:", name, len(data))
		_, _ = hash.Write(data)
		return nil
	})
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", hash.Sum(nil)[:16]), nil
}

func assetURL(path string) string { return "/static/" + assetVersion + "/" + path }

func staticHandler(files fs.FS) http.Handler {
	prefix := assetURL("")
	fileServer := http.StripPrefix(prefix, http.FileServerFS(files))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, prefix)
		info, err := fs.Stat(files, name)
		if err != nil || info.IsDir() {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		w.Header().Set("ETag", `"`+assetVersion+`"`)
		fileServer.ServeHTTP(w, r)
	})
}
