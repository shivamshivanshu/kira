package setx_test

import (
	"testing"

	"github.com/shivamshivanshu/kira/internal/setx"
)

func TestDeduperAdd(t *testing.T) {
	d := setx.NewDeduper[string]()

	first := d.Add("a")

	if !first {
		t.Errorf("Add(%q) first call = %v, want true", "a", first)
	}

	second := d.Add("a")

	if second {
		t.Errorf("Add(%q) second call = %v, want false", "a", second)
	}

	other := d.Add("b")

	if !other {
		t.Errorf("Add(%q) first call = %v, want true", "b", other)
	}
}

func TestToSet(t *testing.T) {
	xs := []string{"a", "b", "a", "c"}

	got := setx.ToSet(xs)

	want := map[string]bool{"a": true, "b": true, "c": true}
	if len(got) != len(want) {
		t.Fatalf("ToSet(%v) = %v, want %v", xs, got, want)
	}
	for k := range want {
		if !got[k] {
			t.Errorf("ToSet(%v)[%q] = false, want true", xs, k)
		}
	}
}

func TestToSetEmpty(t *testing.T) {
	got := setx.ToSet[string](nil)

	if len(got) != 0 {
		t.Errorf("ToSet(nil) = %v, want empty", got)
	}
}
