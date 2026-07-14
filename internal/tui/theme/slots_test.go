package theme

import (
	"testing"

	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func TestPaletteSlotsMatchThemeSlots(t *testing.T) {
	slots := paletteSlots(&themePalette{})
	for _, name := range datamodel.ThemeSlots {
		if _, ok := slots[name]; !ok {
			t.Errorf("palette cannot apply slot %q from datamodel.ThemeSlots", name)
		}
	}
	if len(slots) != len(datamodel.ThemeSlots) {
		t.Errorf("palette applies %d slots, datamodel.ThemeSlots lists %d; keep them in sync", len(slots), len(datamodel.ThemeSlots))
	}
}
