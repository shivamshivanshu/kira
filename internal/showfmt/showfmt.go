// Package showfmt defines the item-reference string forms shared by the TUI
// yank picker and `kira show --format`, so both render identical text.
package showfmt

import (
	"fmt"
	"strings"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/id"
	"github.com/shivamshivanshu/kira/internal/workon"
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

func Format(form Form, it Item, w datamodel.Workon) (string, error) {
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
		return branch(it, w), nil
	}
	return "", fmt.Errorf("unknown format %q, want one of %s", form, strings.Join(Names(), ", "))
}

func numberTitle(it Item) string {
	return strings.TrimSpace(it.Number + " " + it.Title)
}

// branch renders through workon so the yanked name is exactly the branch
// `kira workon` would create for this item.
func branch(it Item, w datamodel.Workon) string {
	return workon.RenderBranch(w.BranchPattern, id.KeyOf(it.Number), it.Number, it.Title, w.Casing.Separator())
}
