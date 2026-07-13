// Package showfmt defines the item-reference string forms shared by the TUI
// yank picker and `kira show --format`, so both render identical text.
package showfmt

import (
	"fmt"
	"strings"
)

type Form string

const (
	FormID          Form = "id"
	FormNumber      Form = "number"
	FormNumberTitle Form = "number-title"
	FormMarkdown    Form = "markdown"
	FormBranch      Form = "branch"
)

var Forms = []Form{FormID, FormNumber, FormNumberTitle, FormMarkdown, FormBranch}

var labels = map[Form]string{
	FormID:          "id",
	FormNumber:      "number",
	FormNumberTitle: "number + title",
	FormMarkdown:    "markdown link",
	FormBranch:      "branch name",
}

func (f Form) Label() string { return labels[f] }

func Names() []string {
	names := make([]string, len(Forms))
	for i, f := range Forms {
		names[i] = string(f)
	}
	return names
}

type Item struct {
	ID     string
	Number string
	Title  string
}

func Format(form Form, it Item) (string, error) {
	switch form {
	case FormID:
		return it.ID, nil
	case FormNumber:
		return it.Number, nil
	case FormNumberTitle:
		return numberTitle(it), nil
	case FormMarkdown:
		return "[" + numberTitle(it) + "](" + it.Number + ")", nil
	case FormBranch:
		return branch(it), nil
	}
	return "", fmt.Errorf("unknown format %q, want one of %s", form, strings.Join(Names(), ", "))
}

func numberTitle(it Item) string {
	return strings.TrimSpace(it.Number + " " + it.Title)
}

func branch(it Item) string {
	parts := make([]string, 0, 2)
	for _, s := range []string{it.Number, it.Title} {
		if slug := slugify(s); slug != "" {
			parts = append(parts, slug)
		}
	}
	return strings.Join(parts, "-")
}

func slugify(s string) string {
	var b strings.Builder
	prevDash := true
	for _, r := range strings.ToLower(s) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			prevDash = false
		case !prevDash:
			b.WriteByte('-')
			prevDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}
