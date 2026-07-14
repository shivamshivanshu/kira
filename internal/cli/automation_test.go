package cli

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func TestAutomationTrustJSONIsPureJSON(t *testing.T) {
	dir := initFixture(t)
	root, _ := newRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"-C", dir, "automation", "trust", "--json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("automation trust --json: %v\n%s", err, out.String())
	}
	var res datamodel.AutomationTrustResult
	if err := json.Unmarshal(out.Bytes(), &res); err != nil {
		t.Fatalf("stdout not pure JSON: %v\noutput: %q", err, out.String())
	}
}
