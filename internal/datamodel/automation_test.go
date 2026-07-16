package datamodel

import (
	"testing"
	"time"
)

func TestTimeoutDuration(t *testing.T) {
	cases := []struct {
		name    string
		timeout string
		want    time.Duration
		wantErr bool
	}{
		{"empty uses default", "", DefaultAutomationTimeout, false},
		{"blank uses default", "   ", DefaultAutomationTimeout, false},
		{"valid duration", "5s", 5 * time.Second, false},
		{"unparsable", "banana", 0, true},
		{"zero is rejected", "0s", 0, true},
		{"negative is rejected", "-1s", 0, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := AutomationHook{Timeout: tc.timeout}
			got, err := h.TimeoutDuration()
			if tc.wantErr {
				if err == nil {
					t.Fatalf("TimeoutDuration(%q) = %v, nil; want error", tc.timeout, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("TimeoutDuration(%q) unexpected error: %v", tc.timeout, err)
			}
			if got != tc.want {
				t.Fatalf("TimeoutDuration(%q) = %v, want %v", tc.timeout, got, tc.want)
			}
		})
	}
}
