package entityschema

import (
	"fmt"
	"strings"
	"testing"

	"github.com/shivamshivanshu/kira/internal/config"
	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func BenchmarkValidateExtras(b *testing.B) {
	cfg := config.Default()
	schemas := map[string]Schema{"ticket": {Fields: []FieldDef{{Name: "points", Type: FieldInt, Placement: PlacementFrontmatter}}}}
	for _, n := range []int{0, 100, 1000} {
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			it := &datamodel.Item{Type: "ticket", Body: "## Description\n" + strings.Repeat("An ordinary line of description that is unrelated to frontmatter validation.\n", n), Extras: map[string]datamodel.ExtraValue{"points": {Scalar: new("1")}}}
			b.ReportAllocs()
			for b.Loop() {
				if v := ValidateExtras(schemas, cfg, it); len(v) > 0 {
					b.Fatal(v)
				}
			}
		})
	}
}
