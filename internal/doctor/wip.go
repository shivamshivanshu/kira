package doctor

import (
	"fmt"

	"github.com/shivamshivanshu/kira/internal/datamodel"
)

// WipOverruns reports wip-capped states holding more items than their limit.
// The write path enforces the cap on moves, but a hand edit, merge, or sync
// auto-resolve can push a state over it without any move ever failing.
func WipOverruns(items []*datamodel.Item) []Finding {
	counts := map[string]map[string]int{}
	for _, it := range items {
		if counts[it.Type] == nil {
			counts[it.Type] = map[string]int{}
		}
		counts[it.Type][it.State]++
	}
	var out []Finding
	for _, typ := range datamodel.ItemTypes {
		wf, _ := datamodel.WorkflowOf(typ)
		for _, st := range wf.States {
			n := counts[typ][st.Key]
			if st.Wip <= 0 || n <= st.Wip {
				continue
			}
			out = append(out, Finding{
				Class:    ClassWipLimit,
				Severity: SeverityWarning,
				Field:    datamodel.KeyState,
				Message:  fmt.Sprintf("%s %s is over its WIP limit (%d > %d)", typ, st.Key, n, st.Wip),
			})
		}
	}
	return out
}
