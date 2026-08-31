package webui

import (
	"bytes"
	"io/fs"
	"math"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

func TestThemeAssetsUseSharedTokens(t *testing.T) {
	files, err := fs.Glob(assets, "static/css/*.css")
	if err != nil {
		t.Fatal(err)
	}
	definitions := make(map[string]bool)
	references := make(map[string]bool)
	defined := regexp.MustCompile(`(--[a-z0-9-]+)\s*:`)
	used := regexp.MustCompile(`var\((--[a-z0-9-]+)`)
	literal := regexp.MustCompile(`(?i)#[0-9a-f]{3,8}\b|\b(?:rgba?|hsla?)\(`)
	for _, name := range files {
		data, err := assets.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.HasSuffix(name, "/tokens.css") && literal.Match(data) {
			t.Errorf("%s hard-codes colors outside the theme tokens", name)
		}
		for _, match := range defined.FindAllSubmatch(data, -1) {
			definitions[string(match[1])] = true
		}
		for _, match := range used.FindAllSubmatch(data, -1) {
			references[string(match[1])] = true
		}
	}
	for name := range references {
		if !definitions[name] {
			t.Errorf("undefined CSS token: %s", name)
		}
	}
	base, _ := assets.ReadFile("static/css/base.css")
	if !bytes.Contains(base, []byte("prefers-reduced-motion")) {
		t.Fatal("animations need a reduced-motion alternative")
	}
}

func TestThemeTextContrast(t *testing.T) {
	data, err := assets.ReadFile("static/css/tokens.css")
	if err != nil {
		t.Fatal(err)
	}
	css := string(data)
	declaration := regexp.MustCompile(`(--color-[a-z-]+):\s*(#[0-9a-f]{6});`)
	palette := func(selector string) map[string]string {
		t.Helper()
		start := strings.Index(css, selector+" {")
		if start < 0 {
			t.Fatalf("missing palette %s", selector)
		}
		block, _, _ := strings.Cut(css[start:], "}")
		values := make(map[string]string)
		for _, match := range declaration.FindAllStringSubmatch(block, -1) {
			values[match[1]] = match[2]
		}
		return values
	}
	light, dark := palette(":root"), palette(`:root[data-theme="dark"]`)
	automatic := palette(":root:not([data-theme])")
	for name, value := range dark {
		if automatic[name] != value {
			t.Errorf("system and explicit dark themes disagree on %s", name)
		}
	}
	for mode, values := range map[string]map[string]string{"light": light, "dark": dark} {
		for _, pair := range [][2]string{
			{"text", "bg"}, {"text", "surface"}, {"text-secondary", "surface"},
			{"text-muted", "surface"}, {"text-muted", "surface-subtle"}, {"text-muted", "sidebar"}, {"text-muted", "hover"},
			{"on-accent", "accent"}, {"accent-text", "surface"},
			{"todo-text", "todo-bg"}, {"doing-text", "doing-bg"}, {"done-text", "done-bg"},
			{"danger-text", "danger-bg"}, {"success-text", "success-bg"}, {"info-text", "info-bg"},
		} {
			foreground, background := values["--color-"+pair[0]], values["--color-"+pair[1]]
			if foreground == "" || background == "" {
				t.Fatalf("%s missing palette pair %v", mode, pair)
			}
			a, b := luminance(foreground), luminance(background)
			ratio := (max(a, b) + .05) / (min(a, b) + .05)
			if ratio < 4.5 {
				t.Errorf("%s %s/%s contrast=%.2f, want>=4.5", mode, pair[0], pair[1], ratio)
			}
		}
	}
}

func luminance(hex string) float64 {
	var channels [3]float64
	for i := range channels {
		n, _ := strconv.ParseUint(hex[1+2*i:3+2*i], 16, 8)
		value := float64(n) / 255
		if value <= .04045 {
			channels[i] = value / 12.92
		} else {
			channels[i] = math.Pow((value+.055)/1.055, 2.4)
		}
	}
	return .2126*channels[0] + .7152*channels[1] + .0722*channels[2]
}
