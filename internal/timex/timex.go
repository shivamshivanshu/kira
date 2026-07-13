package timex

import "time"

func CompareRFC3339(a, b string) (cmp int, aOK, bOK bool) {
	ta, ea := time.Parse(time.RFC3339, a)
	tb, eb := time.Parse(time.RFC3339, b)
	aOK, bOK = ea == nil, eb == nil
	if aOK && bOK {
		return ta.Compare(tb), true, true
	}
	return 0, aOK, bOK
}
