// Package seed generates deterministic kira fixtures from a size-parametric recipe shared by the perf harness and, later, the tour/vhs demos.
package seed

import (
	"fmt"
	"math/rand"

	"github.com/shivamshivanshu/kira/internal/datamodel"
)

type Spec struct {
	Type     string
	Subtype  string
	Title    string
	Priority string
	Owner    string
	Labels   []string
	Category datamodel.Category
	Parent   int
	Comments []string
	Body     string
}

const itemsPerEpic = 15

var (
	owners      = []string{"shivam", "alice"}
	subtypes    = []string{"bug", "story", "task", "spike"}
	labelVocab  = []string{"bug", "feature", "perf", "tech-debt", "orderbook", "infra"}
	verbs       = []string{"Fix", "Add", "Refactor", "Investigate", "Optimize", "Document", "Remove", "Harden"}
	nouns       = []string{"parser", "index", "scheduler", "cache", "allocator", "codec", "hook", "resolver", "queue", "snapshot"}
	epicThemes  = []string{"Query engine", "Storage layer", "CLI ergonomics", "Sync protocol", "Telemetry", "Onboarding"}
	commentPool = []string{
		"Reproduced on the 1k fixture.",
		"Blocked pending review of the sibling change.",
		"Picking this up after the sprint cutover.",
		"Root cause is a stale watermark; patch incoming.",
		"Deferring to next milestone per triage.",
	}
)

func Recipe(size int, seed int64) []Spec {
	if size <= 0 {
		return nil
	}
	rng := rand.New(rand.NewSource(seed))
	epicCount := size / itemsPerEpic
	if epicCount == 0 {
		epicCount = 1
	}
	specs := make([]Spec, 0, size)
	for i := 0; i < epicCount; i++ {
		specs = append(specs, epicSpec(rng, i))
	}
	for i := epicCount; i < size; i++ {
		specs = append(specs, ticketSpec(rng, epicCount))
	}
	return specs
}

func epicSpec(rng *rand.Rand, i int) Spec {
	return Spec{
		Type:     datamodel.TypeEpic,
		Title:    epicThemes[i%len(epicThemes)] + fmt.Sprintf(" (phase %d)", i+1),
		Priority: pick(rng, []weighted[string]{{"P0", 1}, {"P1", 1}}),
		Owner:    owners[rng.Intn(len(owners))],
		Labels:   pickLabels(rng),
		Category: pick(rng, []weighted[datamodel.Category]{{datamodel.CategoryTodo, 3}, {datamodel.CategoryDoing, 5}, {datamodel.CategoryDone, 2}}),
		Parent:   -1,
		Comments: nil,
		Body:     "Tracking epic for related work.",
	}
}

func ticketSpec(rng *rand.Rand, epicCount int) Spec {
	parent := -1
	if rng.Intn(100) < 85 {
		parent = rng.Intn(epicCount)
	}
	return Spec{
		Type:     datamodel.TypeTicket,
		Subtype:  subtypes[rng.Intn(len(subtypes))],
		Title:    verbs[rng.Intn(len(verbs))] + " " + nouns[rng.Intn(len(nouns))],
		Priority: pick(rng, []weighted[string]{{"P0", 1}, {"P1", 3}, {"P2", 5}, {"P3", 3}}),
		Owner:    owners[rng.Intn(len(owners))],
		Labels:   pickLabels(rng),
		Category: pick(rng, []weighted[datamodel.Category]{{datamodel.CategoryTodo, 9}, {datamodel.CategoryDoing, 7}, {datamodel.CategoryDone, 4}}),
		Parent:   parent,
		Comments: pickComments(rng),
		Body:     "Details captured during triage.",
	}
}

func pickLabels(rng *rand.Rand) []string {
	n := rng.Intn(3)
	out := make([]string, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, labelVocab[rng.Intn(len(labelVocab))])
	}
	return dedup(out)
}

func pickComments(rng *rand.Rand) []string {
	if rng.Intn(100) >= 30 {
		return nil
	}
	n := 1 + rng.Intn(3)
	out := make([]string, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, commentPool[rng.Intn(len(commentPool))])
	}
	return out
}

func dedup(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

type weighted[T any] struct {
	value  T
	weight int
}

func pick[T any](rng *rand.Rand, ws []weighted[T]) T {
	total := 0
	for _, w := range ws {
		total += w.weight
	}
	if total <= 0 {
		return ws[len(ws)-1].value
	}
	r := rng.Intn(total)
	for _, w := range ws {
		if r < w.weight {
			return w.value
		}
		r -= w.weight
	}
	return ws[len(ws)-1].value
}
