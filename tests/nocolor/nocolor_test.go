package nocolor

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

var colorLiteral = regexp.MustCompile(`"#[0-9a-fA-F]{3,8}"|lipgloss\.(Color|AdaptiveColor|CompleteColor|CompleteAdaptiveColor|ANSIColor)|termenv\.(RGBColor|ANSIColor|ANSI256Color)`)

const themePkg = "internal/tui/theme"

func TestNoColorLiteralOutsideThemePackage(t *testing.T) {
	root := "../.."
	var offenders []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		rel := strings.TrimPrefix(filepath.ToSlash(path), "../../")
		if strings.HasPrefix(rel, themePkg) {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for i, line := range strings.Split(string(data), "\n") {
			if colorLiteral.MatchString(line) {
				offenders = append(offenders, filepath.ToSlash(rel)+":"+strconv.Itoa(i+1)+"  "+strings.TrimSpace(line))
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(offenders) > 0 {
		t.Fatalf("color literals found outside %s (the theme package is the only permitted home):\n%s",
			themePkg, strings.Join(offenders, "\n"))
	}
}
