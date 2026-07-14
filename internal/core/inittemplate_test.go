package core

import (
	"reflect"
	"strings"
	"testing"

	"github.com/shivamshivanshu/kira/internal/config"
)

// uncommentOptional re-enables the commented-out config blocks. Documentation
// prose uses "# " (hash-space) and stays inert; disabled config uses a bare "#"
// prefixing the key, so stripping a leading hash not followed by a space turns
// exactly the optional blocks back on.
func uncommentOptional(seed string) string {
	lines := strings.Split(seed, "\n")
	for i, l := range lines {
		if len(l) >= 2 && l[0] == '#' && l[1] != ' ' {
			lines[i] = l[1:]
		}
	}
	return strings.Join(lines, "\n")
}

func TestSeedCommentsAreInert(t *testing.T) {
	seed := initConfigYAML("KIRA", "kira")

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
	// defaults; the seed otherwise differs from Default() only by the empty
	// project vocabularies a fresh repo starts with and the shipped starter
	// filter, which Default() leaves empty so it stays user-removable.
	def := config.Default()
	fromUncommented.Project = def.Project
	fromUncommented.Labels = def.Labels
	fromUncommented.People = def.People
	fromUncommented.Filters = def.Filters
	if !reflect.DeepEqual(fromUncommented, def) {
		t.Errorf("uncommented seed diverges from Default() beyond project/labels/people:\n%+v\n!=\n%+v", fromUncommented, def)
	}
}

func TestSeedConfigSetPreservesEveryOtherLine(t *testing.T) {
	seed := initConfigYAML("KIRA", "kira")
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
