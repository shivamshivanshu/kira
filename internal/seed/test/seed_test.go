package seed_test

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/gitx"
	"github.com/shivamshivanshu/kira/internal/id"
	"github.com/shivamshivanshu/kira/internal/seed"
	"github.com/shivamshivanshu/kira/internal/testutil"
)

func TestRecipeDeterministic(t *testing.T) {
	a := seed.Recipe(1000, 42)
	b := seed.Recipe(1000, 42)
	if !reflect.DeepEqual(a, b) {
		t.Fatal("same size and seed produced different recipes")
	}
	if reflect.DeepEqual(seed.Recipe(1000, 42), seed.Recipe(1000, 43)) {
		t.Fatal("different seeds produced identical recipes")
	}
}

func TestRecipeShape(t *testing.T) {
	for _, size := range []int{0, 6, 100, 1000} {
		specs := seed.Recipe(size, 7)
		if len(specs) != size {
			t.Fatalf("size %d: got %d specs", size, len(specs))
		}
		epics := 0
		for i, sp := range specs {
			if sp.Type == datamodel.TypeEpic {
				epics++
			}
			if sp.Parent >= 0 {
				if sp.Parent >= i {
					t.Fatalf("size %d spec %d: parent %d not earlier", size, i, sp.Parent)
				}
				if specs[sp.Parent].Type != datamodel.TypeEpic {
					t.Fatalf("size %d spec %d: parent is not an epic", size, i)
				}
			}
		}
		if size > 0 && epics == 0 {
			t.Fatalf("size %d: no epics", size)
		}
	}
}

func TestSeedProducesItems(t *testing.T) {
	root := testutil.InitGitRepo(t)
	if _, err := core.Init(root, "KIRA", false); err != nil {
		t.Fatalf("Init: %v", err)
	}
	s, err := core.Discover(root)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	cfg, err := s.Config()
	if err != nil {
		t.Fatalf("Config: %v", err)
	}

	sum, err := seed.Run(root, cfg, seed.Opts{Size: 120, Seed: 3})
	if err != nil {
		t.Fatalf("Seed: %v", err)
	}
	if sum.Items() != 120 {
		t.Fatalf("summary mismatch: %+v", sum)
	}

	items, _, err := s.LoadAll()
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	if len(items) != 120 {
		t.Fatalf("loaded %d items, want 120", len(items))
	}

	numbers := make(map[string]*datamodel.Item, len(items))
	for _, it := range items {
		if it.Number == "" || it.State == "" || it.Title == "" {
			t.Fatalf("item %s missing required field", it.ID)
		}
		numbers[it.Number] = it
	}
	for _, it := range items {
		if it.Epic == nil {
			continue
		}
		parent, ok := numbers[*it.Epic]
		if !ok {
			t.Fatalf("item %s epic %q resolves to nothing", it.Number, *it.Epic)
		}
		if parent.Type != datamodel.TypeEpic {
			t.Fatalf("item %s epic %q is not an epic", it.Number, *it.Epic)
		}
	}

	if out, err := (gitx.Repo{Dir: root}).Output("status", "--porcelain"); err != nil {
		t.Fatalf("git status: %v", err)
	} else if out != "" {
		t.Fatalf("seeded tree not clean:\n%s", out)
	}
}

func TestSeedHashStyleSurvivesSuffixCollisions(t *testing.T) {
	root := testutil.InitGitRepo(t)
	if _, err := core.Init(root, "KIRA", false); err != nil {
		t.Fatalf("Init: %v", err)
	}
	s, err := core.Discover(root)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	cfg, err := s.Config()
	if err != nil {
		t.Fatalf("Config: %v", err)
	}
	cfg.ID.Style = datamodel.IDStyleHash

	minted := 0
	mint := func() id.ULID {
		u, err := id.ParseULID(fmt.Sprintf("01%018d9X4MV3", minted))
		if err != nil {
			t.Fatalf("mint %d: %v", minted, err)
		}
		minted++
		return u
	}
	if _, err := seed.Run(root, cfg, seed.Opts{Size: 12, Seed: 3, Mint: mint}); err != nil {
		t.Fatalf("Seed: %v", err)
	}

	items, _, err := s.LoadAll()
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	if len(items) != 12 {
		t.Fatalf("loaded %d items, want 12", len(items))
	}
	seen := map[string]bool{}
	for _, it := range items {
		if seen[it.Number] {
			t.Fatalf("duplicate live number %q allocated within one seed run", it.Number)
		}
		seen[it.Number] = true
	}
}
