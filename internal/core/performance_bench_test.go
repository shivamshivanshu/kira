package core

import (
	"fmt"
	"testing"

	"github.com/shivamshivanshu/kira/internal/config"
	"github.com/shivamshivanshu/kira/internal/datamodel"
)

var benchmarkProgress map[string]datamodel.EpicProgress

func BenchmarkEpicProgress(b *testing.B) {
	for _, n := range []int{100, 1000} {
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			cfg := config.Default()
			items := make([]*datamodel.Item, n+1)
			for i := range n {
				items[i] = &datamodel.Item{ID: fmt.Sprint(i), Number: fmt.Sprintf("KIRA-%d", i+1), Type: datamodel.TypeEpic}
				if i > 0 {
					items[i].Parent = new(items[i-1].ID)
				}
			}
			items[n] = &datamodel.Item{ID: "ticket", Number: "KIRA-99999", Type: datamodel.TypeTicket, State: "DONE", Parent: new(items[n-1].ID)}
			b.ReportAllocs()
			for b.Loop() {
				p, err := allEpicProgress(cfg, items)
				if err != nil {
					b.Fatal(err)
				}
				benchmarkProgress = p
			}
		})
	}
}
