package doctor

import (
	"fmt"
	"strings"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/id"
)

const (
	outlierSuffixWidth = 6
	outlierGapFactor   = 100
)

func SequentialOutliers(items []*datamodel.Item) []Finding {
	type outlier struct {
		itemID string
		number string
		key    string
		n      int
	}
	baseMax := map[string]int{}
	var candidates []outlier
	for _, it := range items {
		num, err := id.ParseNumber(it.Number)
		if err != nil {
			continue
		}
		key := strings.ToUpper(num.Key)
		if len(it.Number)-len(num.Key)-1 >= outlierSuffixWidth {
			candidates = append(candidates, outlier{it.ID, it.Number, key, num.N})
			continue
		}
		if num.N > baseMax[key] {
			baseMax[key] = num.N
		}
	}
	var out []Finding
	for _, c := range candidates {
		base := baseMax[c.key]
		if base == 0 || c.n <= base*outlierGapFactor {
			continue
		}
		out = append(out, Finding{
			Class:    ClassNumberOutlier,
			Severity: SeverityWarning,
			ItemID:   c.itemID,
			Number:   c.number,
			Field:    datamodel.KeyNumber,
			Message: fmt.Sprintf("number %s has an all-digit %d-character suffix while board %s otherwise tops out at %d; it inflates sequential allocation — renumber it with `kira board move` (the old number survives as an alias)",
				c.number, len(c.number)-len(c.key)-1, c.key, base),
		})
	}
	return out
}
