package config

import "testing"

func TestValidateFilters(t *testing.T) {
	cases := []struct {
		name, expr string
		wantErr    bool
	}{
		{"builtin fields", "owner=shivam ORDER BY due", false},
		{"unknown field passes unresolved", "team = infra", false},
		{"unknown field in ORDER BY passes unresolved", "x ORDER BY team", false},
		{"syntax error still rejected", "state =", true},
		{"bad date still rejected", "created > notadate", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := Default()
			c.Filters = map[string]string{"f": tc.expr}
			err := Validate(c)
			if tc.wantErr && err == nil {
				t.Fatalf("Validate accepted filter %q", tc.expr)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("Validate rejected filter %q: %v", tc.expr, err)
			}
		})
	}
}
