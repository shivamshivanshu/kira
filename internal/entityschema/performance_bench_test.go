package entityschema

import (
	"fmt"
	"strings"
	"testing"

	"github.com/shivamshivanshu/kira/internal/codec"
	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func BenchmarkValidateExtras(b *testing.B) {
	cfg := &datamodel.Config{}
	schemas := map[string]Schema{"ticket": {Fields: []FieldDef{{Name: "points", Type: FieldInt}}}}
	for _, n := range []int{0, 100, 1000} {
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			it := &datamodel.Item{
				Type:         "ticket",
				Body:         "## Description\n" + strings.Repeat("An ordinary line of description that is unrelated to frontmatter validation.\n", n),
				ExtraScalars: map[string]string{"points": "1"},
			}
			b.ReportAllocs()
			for b.Loop() {
				if v := ValidateExtras(schemas, cfg, it, codec.DecodeExtraScalar); len(v) > 0 {
					b.Fatal(v)
				}
			}
		})
	}
}
