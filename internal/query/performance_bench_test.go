package query

import (
	"testing"

	"github.com/shivamshivanshu/kira/internal/datamodel"
)

var benchmarkMatches int

func BenchmarkSelectiveQuery(b *testing.B) {
	q, err := Compile(`state = DONE AND "important"`, Options{})
	if err != nil {
		b.Fatal(err)
	}
	items := make([]*datamodel.Item, 1000)
	for i := range items {
		items[i] = &datamodel.Item{State: "TODO", Title: "An Important Ticket With Mixed Case And A Moderately Long Title"}
	}
	cfg := &datamodel.Config{}
	b.ReportAllocs()
	for b.Loop() {
		n := 0
		for _, it := range items {
			if q.Pred(it, cfg) {
				n++
			}
		}
		benchmarkMatches = n
	}
}
