package codec_test

import (
	"fmt"
	"math/rand"
	"reflect"
	"testing"
	"time"

	"github.com/shivamshivanshu/kira/internal/codec"
	"github.com/shivamshivanshu/kira/internal/datamodel"
)

var tokens = []string{
	"bug", "orderbook", "shivam", "alice", "P1", "IN_PROGRESS",
	"null", "true", "false", "123", "3.14", "~", "",
	"a: b", "has #hash", "[bracket]", `quote"inside`, "line\nbreak",
	"01J8X8Q7RZTN5Y3VXW2A9K4E7F", "trailing ", " leading",
}

func pick(r *rand.Rand) string { return tokens[r.Intn(len(tokens))] }

func pickNonEmpty(r *rand.Rand) string {
	for {
		if s := pick(r); s != "" {
			return s
		}
	}
}

func maybe(r *rand.Rand) *string {
	if r.Intn(3) == 0 {
		return nil
	}
	s := pick(r)
	return &s
}

func randList(r *rand.Rand) []string {
	n := r.Intn(4)
	out := make([]string, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, pick(r))
	}
	return out
}

func randTimestamp(r *rand.Rand) string {
	base := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	offsets := []*time.Location{
		time.UTC,
		time.FixedZone("+05:30", 5*3600+1800),
		time.FixedZone("-08:00", -8*3600),
	}
	tm := base.Add(time.Duration(r.Int63n(int64(200000) * int64(time.Hour))))
	return tm.In(offsets[r.Intn(len(offsets))]).Format(time.RFC3339)
}

func randDate(r *rand.Rand) *string {
	switch r.Intn(4) {
	case 0:
		return nil
	case 1:
		return maybe(r)
	default:
		base := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
		s := base.AddDate(0, 0, r.Intn(4000)).Format(time.DateOnly)
		return &s
	}
}

func randLinks(r *rand.Rand) map[string][]string {
	var links map[string][]string
	for _, typ := range datamodel.LinkTypes {
		if r.Intn(3) != 0 {
			continue
		}
		targets := make([]string, 1+r.Intn(3))
		for i := range targets {
			targets[i] = pick(r)
		}
		if links == nil {
			links = map[string][]string{}
		}
		links[string(typ)] = targets
	}
	return links
}

func randItem(r *rand.Rand) *datamodel.Item {
	it := &datamodel.Item{
		ID:         pickNonEmpty(r),
		Number:     fmt.Sprintf("KIRA-%d", r.Intn(1000)),
		Aliases:    randList(r),
		Type:       []string{"ticket", "epic"}[r.Intn(2)],
		Subtype:    maybe(r),
		Title:      pickNonEmpty(r),
		State:      pickNonEmpty(r),
		Resolution: maybe(r),
		Priority:   maybe(r),
		Rank:       maybe(r),
		Owner:      maybe(r),
		Reporter:   maybe(r),
		Labels:     randList(r),
		Epic:       maybe(r),
		BlockedBy:  randList(r),
		Links:      randLinks(r),
		Sprint:     maybe(r),
		Due:        randDate(r),
		Created:    randTimestamp(r),
		Updated:    randTimestamp(r),
		Body:       "\n## Description\n" + pick(r) + "\n---\nmid-body rule\n",
	}
	if r.Intn(2) == 0 {
		e := float64(r.Intn(100)) + float64(r.Intn(4))*0.25
		it.Estimate = &e
	}
	return it
}

func TestRoundTripProperty(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	for i := 0; i < 2000; i++ {
		want := randItem(r)
		out := codec.Serialize(want)
		got, err := codec.Parse(out)
		if err != nil {
			t.Fatalf("iter %d: parse(serialize) failed: %v\n%s", i, err, out)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("iter %d: mismatch\nwant %+v\ngot  %+v\nserialized:\n%s", i, want, got, out)
		}
		if out2 := codec.Serialize(got); out2 != out {
			t.Fatalf("iter %d: serialize not idempotent\n%q\n%q", i, out, out2)
		}
	}
}

func FuzzParse(f *testing.F) {
	f.Add([]byte(readExample(f)))
	f.Add([]byte("---\n---\n"))
	f.Add([]byte("garbage"))
	f.Fuzz(func(t *testing.T, data []byte) {
		it, err := codec.Parse(string(data))
		if err != nil {
			return
		}
		out := codec.Serialize(it)
		reparsed, err := codec.Parse(out)
		if err != nil {
			t.Fatalf("re-parse of serialized output failed: %v\n%s", err, out)
		}
		if codec.Serialize(reparsed) != out {
			t.Fatalf("serialize not a fixed point for input %q", data)
		}
	})
}
