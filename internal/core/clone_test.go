package core

import (
	"testing"

	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func TestCloneItemDeepCopiesExtras(t *testing.T) {
	src := &datamodel.Item{Extras: map[string]datamodel.ExtraValue{"watchers": {List: []string{"alice"}}}}
	dst := cloneItem(src)
	dst.Extras["watchers"].List[0] = "mallory"
	if got := src.Extras["watchers"].List[0]; got != "alice" {
		t.Fatalf("original mutated to %q; cloneItem must deep-copy the Extras list arm", got)
	}
}
