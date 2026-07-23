package merge

import "github.com/shivamshivanshu/kira/internal/setx"

func setMerge(base, ours, theirs []string) []string {
	b, o, t := setx.ToSet(base), setx.ToSet(ours), setx.ToSet(theirs)
	var out []string
	dedup := setx.NewDeduper[string]()
	push := func(e string) {
		if dedup.Add(e) {
			out = append(out, e)
		}
	}
	for _, e := range base {
		if o[e] && t[e] {
			push(e)
		}
	}
	for _, e := range ours {
		if !b[e] {
			push(e)
		}
	}
	for _, e := range theirs {
		if !b[e] {
			push(e)
		}
	}
	return out
}

func aliasUnion(lists ...[]string) []string {
	var out []string
	dedup := setx.NewDeduper[string]()
	for _, l := range lists {
		for _, e := range l {
			if dedup.Add(e) {
				out = append(out, e)
			}
		}
	}
	return out
}

func linkMerge(base, ours, theirs map[string][]string) map[string][]string {
	types := map[string]bool{}
	for t := range base {
		types[t] = true
	}
	for t := range ours {
		types[t] = true
	}
	for t := range theirs {
		types[t] = true
	}
	out := map[string][]string{}
	for t := range types {
		if merged := setMerge(base[t], ours[t], theirs[t]); len(merged) > 0 {
			out[t] = merged
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
