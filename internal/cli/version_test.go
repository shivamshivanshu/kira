package cli

import (
	"encoding/json"
	"testing"

	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func TestVersionResultHandshake(t *testing.T) {
	data, err := json.Marshal(versionResult())
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got datamodel.VersionResult
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.JSONContract != datamodel.JSONContractVersion {
		t.Errorf("json_contract = %d, want %d", got.JSONContract, datamodel.JSONContractVersion)
	}
	if got.Version == "" {
		t.Error("version is empty")
	}
	if got.Go == "" {
		t.Error("go is empty")
	}
}
