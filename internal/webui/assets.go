package webui

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/shivamshivanshu/kira/internal/codec"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/ptr"
	"github.com/shivamshivanshu/kira/internal/timex"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
)

//go:embed static
var assets embed.FS

//go:embed templates/*.html
var templates embed.FS

func displayTime(s string) string {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return s
	}
	return t.Format("Jan 2, 2006 · 15:04 MST")
}

var markdownEngine = goldmark.New(goldmark.WithExtensions(extension.GFM))

func markdown(s string) (template.HTML, error) {
	var out bytes.Buffer
	if err := markdownEngine.Convert([]byte(s), &out); err != nil {
		return "", err
	}
	return template.HTML(out.String()), nil
}

var pageTemplate = template.Must(template.New("page").Funcs(template.FuncMap{
	"asset":    assetURL,
	"markdown": markdown,
	"prose":    codec.Description,
	"mul100":   func(n float64) float64 { return n * 100 },
	"samples": func(p *datamodel.Percentiles) *datamodel.Percentiles {
		if p == nil || p.N == 0 {
			return nil
		}
		return p
	},
	"label": func(s string) string { return strings.ReplaceAll(s, "_", " ") },
	"str":   ptr.Deref[string],
	"number": func(n *float64) string {
		if n == nil {
			return ""
		}
		return fmt.Sprintf("%g", *n)
	},
	"date": displayTime,
	"join": func(s []string) string { return strings.Join(s, ", ") },
	"short": func(s string) string {
		if len(s) > 8 {
			return s[:8]
		}
		return s
	},
	"depth": func(n int) int { return min(max(n, 0), 6) },
	"inc":   func(n int) int { return n + 1 },
	"peak": func(values []int) int {
		peak := 1
		for _, n := range values {
			peak = max(peak, n)
		}
		return peak
	},
	"overdue": func(due *string, category string) bool {
		return datamodel.IsOverdue(due, category, timex.Now())
	},
}).ParseFS(templates, "templates/*.html"))

func renderPage(w io.Writer, p pageData) error {
	return pageTemplate.ExecuteTemplate(w, "page", p)
}

func writePage(w http.ResponseWriter, p pageData, status int) {
	var body bytes.Buffer
	if err := renderPage(&body, p); err != nil {
		http.Error(w, "rendering dashboard: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write(body.Bytes())
}
