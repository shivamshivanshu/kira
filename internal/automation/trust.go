package automation

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/shivamshivanshu/kira/internal/datamodel"
)

const trustFile = "automation.trust"

// The grant lives under the git dir, never in the worktree: a marker inside
// .kira is committable, so a hostile repo could ship its own grant and get
// its hooks executed on clone. An empty dir means the git dir could not be
// resolved, and is refused rather than resolved against the process cwd.

func Hash(cfg *datamodel.Config) string {
	data, _ := json.Marshal(cfg.Automation)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func grantedHash(dir string) string {
	if dir == "" {
		return ""
	}
	data, err := os.ReadFile(filepath.Join(dir, trustFile))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func Trusted(dir string, cfg *datamodel.Config) bool {
	granted := grantedHash(dir)
	return granted != "" && granted == Hash(cfg)
}

func Grant(dir string, cfg *datamodel.Config) (string, error) {
	if dir == "" {
		return "", errors.New("cannot locate the git dir to record trust in")
	}
	h := Hash(cfg)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(dir, trustFile), []byte(h), 0o600); err != nil {
		return "", err
	}
	return h, nil
}
