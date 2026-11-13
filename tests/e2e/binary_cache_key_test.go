package e2e

// The kira binary these tests exec is built at runtime by testutil.KiraBinary,
// which go test's result cache cannot see; without this import an edit to kira's
// sources replays a stale cached pass.
import _ "github.com/shivamshivanshu/kira/internal/cli"
