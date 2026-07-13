package tui

import (
	"strings"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/tui/theme"
)

func detailMeta(t theme.Theme, res *datamodel.ShowResult) string {
	parts := []string{t.CategoryStyle(datamodel.Category(res.Category)).Render("[" + res.State + "]")}
	if res.Owner != nil {
		parts = append(parts, t.Dim.Render("owner ")+*res.Owner)
	}
	if res.Priority != nil {
		parts = append(parts, t.PriorityStyle(*res.Priority).Render(*res.Priority))
	}
	if len(res.Labels) > 0 {
		parts = append(parts, t.Dim.Render(strings.Join(res.Labels, " ")))
	}
	return strings.Join(parts, "   ")
}
