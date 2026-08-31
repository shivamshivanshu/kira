package entityschema

import (
	"testing"

	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func TestNewRefResolverExists(t *testing.T) {
	t.Parallel()
	refs := NewRefResolver([]*datamodel.Item{
		{ID: "01T", Type: datamodel.TypeTicket},
		{ID: "01E", Type: datamodel.TypeEpic},
		{Type: datamodel.TypeTicket},
	})
	tests := []struct {
		name   string
		target string
		id     string
		want   bool
	}{
		{"matching type", "ticket", "01T", true},
		{"wrong type", "epic", "01T", false},
		{"unknown id", "ticket", "01X", false},
		{"empty target matches any type", "", "01E", true},
		{"empty id never exists", "ticket", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := refs.Exists(tt.target, tt.id); got != tt.want {
				t.Errorf("Exists(%q, %q) = %v, want %v", tt.target, tt.id, got, tt.want)
			}
		})
	}
}
