package id

import (
	"fmt"
	"slices"
	"strings"

	"github.com/oklog/ulid/v2"

	"github.com/shivamshivanshu/kira/internal/errx"
)

type Item struct {
	ULID    string
	Number  string
	Aliases []string
}

type Snapshot struct {
	Key   string
	Items []Item
}

type NotFoundError struct {
	Token      string
	Suggestion string
}

func (e *NotFoundError) Error() string { return fmt.Sprintf("id: %q resolves to no item", e.Token) }

type AmbiguousError struct {
	Token      string
	Candidates []string
}

func (e *AmbiguousError) Error() string {
	return fmt.Sprintf("id: prefix %q is ambiguous between %s", e.Token, strings.Join(e.Candidates, ", "))
}

type Resolver struct {
	sortedULIDs      []string
	liveByNumber     map[string]string
	aliasHolders     map[string][]string
	liveNumberByULID map[string]string
	numbers          []string
	numberByBareN    map[string][]string
}

func NewResolver(snap Snapshot) *Resolver {
	r := &Resolver{
		sortedULIDs:      make([]string, len(snap.Items)),
		liveByNumber:     make(map[string]string, len(snap.Items)),
		aliasHolders:     map[string][]string{},
		liveNumberByULID: make(map[string]string, len(snap.Items)),
		numbers:          make([]string, len(snap.Items)),
		numberByBareN:    map[string][]string{},
	}
	addBare := func(up string) {
		if i := strings.LastIndexByte(up, '-'); i >= 0 {
			bare := up[i+1:]
			if !slices.Contains(r.numberByBareN[bare], up) {
				r.numberByBareN[bare] = append(r.numberByBareN[bare], up)
			}
		}
	}
	for i, it := range snap.Items {
		u := strings.ToUpper(it.ULID)
		live := strings.ToUpper(it.Number)
		r.sortedULIDs[i] = u
		r.numbers[i] = live
		r.liveByNumber[live] = u
		r.liveNumberByULID[u] = live
		addBare(live)
		for _, a := range it.Aliases {
			up := strings.ToUpper(a)
			if !slices.Contains(r.aliasHolders[up], u) {
				r.aliasHolders[up] = append(r.aliasHolders[up], u)
			}
			addBare(up)
		}
	}
	slices.Sort(r.sortedULIDs)
	return r
}

func (r *Resolver) holdersOf(full string) []string {
	if u, ok := r.liveByNumber[full]; ok {
		return []string{u}
	}
	return r.aliasHolders[full]
}

func (r *Resolver) Resolve(token string) (string, error) {
	t := strings.TrimSpace(token)
	if t == "" {
		return "", &NotFoundError{Token: token}
	}
	up := strings.ToUpper(t)

	if len(up) == ulid.EncodedSize {
		if _, err := ParseULID(up); err == nil {
			if r.contains(up) {
				return up, nil
			}
			return "", &NotFoundError{Token: token}
		}
	}

	cands := r.prefixMatches(up)
	if len(cands) == 1 {
		return cands[0], nil
	}
	if len(cands) > 1 {
		return "", &AmbiguousError{Token: token, Candidates: cands}
	}

	if strings.Contains(up, "-") {
		h := r.holdersOf(up)
		switch len(h) {
		case 0:
			return "", &NotFoundError{Token: token, Suggestion: errx.Nearest(up, r.numbers)}
		case 1:
			return h[0], nil
		default:
			cands := make([]string, len(h))
			for i, u := range h {
				cands[i] = r.liveNumberByULID[u]
			}
			slices.Sort(cands)
			return "", &AmbiguousError{Token: token, Candidates: cands}
		}
	}
	return r.resolveBare(token, up)
}

func (r *Resolver) resolveBare(token, up string) (string, error) {
	fulls := r.numberByBareN[up]
	if len(fulls) == 0 {
		return "", &NotFoundError{Token: token, Suggestion: errx.Nearest(up, r.numbers)}
	}
	seen := map[string]bool{}
	var holder string
	for _, f := range fulls {
		for _, u := range r.holdersOf(f) {
			if !seen[u] {
				seen[u] = true
				holder = u
			}
		}
	}
	if len(seen) == 1 {
		return holder, nil
	}
	cands := append([]string(nil), fulls...)
	slices.Sort(cands)
	return "", &AmbiguousError{Token: token, Candidates: cands}
}

func (r *Resolver) contains(u string) bool {
	_, found := slices.BinarySearch(r.sortedULIDs, u)
	return found
}

func (r *Resolver) prefixMatches(up string) []string {
	if len(up) >= ulid.EncodedSize {
		return nil
	}
	var out []string
	start, _ := slices.BinarySearch(r.sortedULIDs, up)
	for _, u := range r.sortedULIDs[start:] {
		if !strings.HasPrefix(u, up) {
			break
		}
		out = append(out, u)
	}
	return out
}
