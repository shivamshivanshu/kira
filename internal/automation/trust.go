package automation

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/shivamshivanshu/kira/internal/datamodel"
)

const trustFile = "automation.trust"

func Hash(cfg *datamodel.Config) string {
	data, _ := json.Marshal(cfg.Automation)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func GrantedHash(cacheDir string) string {
	data, err := os.ReadFile(filepath.Join(cacheDir, trustFile))
	if err != nil {
		return ""
	}
	return string(data)
}

func Trusted(cacheDir string, cfg *datamodel.Config) bool {
	granted := GrantedHash(cacheDir)
	return granted != "" && granted == Hash(cfg)
}

func Grant(cacheDir string, cfg *datamodel.Config) (string, error) {
	h := Hash(cfg)
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(cacheDir, trustFile), []byte(h), 0o644); err != nil {
		return "", err
	}
	return h, nil
}
