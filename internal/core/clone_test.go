package core

import (
	"testing"

	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func TestCloneItemDeepCopiesExtras(t *testing.T) {
	src := &datamodel.Item{ExtraLists: map[string][]datamodel.Rendered{"watchers": {"alice"}}}
	dst := cloneItem(src)
	dst.ExtraLists["watchers"][0] = "mallory"
	if got := src.ExtraLists["watchers"][0]; got != "alice" {
		t.Fatalf("original mutated to %q; cloneItem must deep-copy ExtraLists", got)
	}
	dst.ExtraScalars = map[string]datamodel.Rendered{"team": "infra"}
	if src.ExtraScalars != nil {
		t.Fatalf("original grew an ExtraScalars map: %v", src.ExtraScalars)
	}
}
