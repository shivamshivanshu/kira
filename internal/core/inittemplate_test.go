package core

import (
	"reflect"
	"strings"
	"testing"

	"github.com/shivamshivanshu/kira/internal/config"
)

// uncommentOptional enables the example sections and their indented settings
// so the parser checks the documented defaults as well as the active scaffold.
func uncommentOptional(seed string) string {
	lines := strings.Split(seed, "\n")
	example := false
	for i, line := range lines {
		switch line {
		case "# sprints:", "# git:", "# workon:", "# ui:":
			lines[i] = strings.TrimPrefix(line, "# ")
			example = true
		default:
			if strings.HasPrefix(line, "#   ") {
				if example {
					lines[i] = strings.TrimPrefix(line, "# ")
				}
			} else {
				example = false
			}
		}
	}
	return strings.Join(lines, "\n")
}

func seedConfigYAML(t *testing.T) string {
	t.Helper()
	seed, err := initConfigYAML("KIRA", "kira")
	if err != nil {
		t.Fatal(err)
	}
	return seed
}

func TestSeedCommentsAreInert(t *testing.T) {
	seed := seedConfigYAML(t)

	fromSeed, err := config.Parse([]byte(seed))
	if err != nil {
		t.Fatalf("seed does not parse: %v", err)
	}
	fromUncommented, err := config.Parse([]byte(uncommentOptional(seed)))
	if err != nil {
		t.Fatalf("uncommented seed does not parse: %v", err)
	}
	if !reflect.DeepEqual(fromSeed, fromUncommented) {
		t.Fatalf("uncommenting the optional blocks changed the parsed config:\n%+v\n!=\n%+v", fromSeed, fromUncommented)
	}

	// Uncommented, the optional blocks must reproduce exactly the documented
	// defaults; the seed otherwise differs from Default() only by the project
	// identity and its default board, the empty project vocabularies a fresh
	// repo starts with, and the shipped starter filter, which Default() leaves
	// empty so it stays user-removable.
	def := config.Default()
	fromUncommented.Project = def.Project
	fromUncommented.Boards = def.Boards
	fromUncommented.Labels = def.Labels
	fromUncommented.People = def.People
	fromUncommented.Filters = def.Filters
	if !reflect.DeepEqual(fromUncommented, def) {
		t.Errorf("uncommented seed diverges from Default() beyond project/boards/labels/people:\n%+v\n!=\n%+v", fromUncommented, def)
	}
}

func TestSeedConfigSetPreservesEveryOtherLine(t *testing.T) {
	seed := seedConfigYAML(t)
	out, err := config.SetScalar([]byte(seed), "commit.mode", "manual")
	if err != nil {
		t.Fatalf("SetScalar: %v", err)
	}
	before, after := strings.Split(seed, "\n"), strings.Split(string(out), "\n")
	if len(before) != len(after) {
		t.Fatalf("line count changed: %d -> %d", len(before), len(after))
	}
	changed := 0
	for i := range before {
		if before[i] != after[i] {
			changed++
			if !strings.HasPrefix(strings.TrimSpace(after[i]), "mode: manual") {
				t.Errorf("unexpected changed line %d: %q", i, after[i])
			}
		}
	}
	if changed != 1 {
		t.Errorf("changed %d lines, want exactly 1", changed)
	}
}
