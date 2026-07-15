package statsfmt

import (
	"fmt"
	"strconv"

	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func CompletionLine(c *datamodel.Completion) string {
	line := fmt.Sprintf("%d/%d done (%.0f%%)", c.Done, c.Total, c.Pct*100)
	if c.Dropped > 0 {
		line += fmt.Sprintf(", %d dropped", c.Dropped)
	}
	return line
}

func PercentileLine(p *datamodel.Percentiles) string {
	return fmt.Sprintf("p50 %s  p90 %s  n=%d", formatFloat(p.P50), formatFloat(p.P90), p.N)
}

func formatFloat(f float64) string { return strconv.FormatFloat(f, 'f', -1, 64) }
