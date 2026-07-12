// Package contract holds the frozen --json contract suite. It execs the built
// kira binary against seeded fixture repos and compares each command's --json
// output to a checked-in golden, with volatile tokens (ULIDs, timestamps, temp
// paths) scrubbed so the goldens are stable across runs (docs/design/09-testing.md
// §3). After WP-1.6 the shapes are additive-only forever; a shape change fails
// here until the golden is regenerated with -update in the same change.
package contract

import (
	"bytes"
	"errors"
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "regenerate golden files instead of comparing")

// kiraBin is the compiled binary under test; toolBin is a PATH holding only git
// and a no-op editor, so rg/fzf always read as absent and find takes its
// deterministic pure-Go fallback regardless of the host.
var (
	kiraBin string
	toolBin string
)

func TestMain(m *testing.M) {
	flag.Parse()
	os.Exit(run(m))
}

func run(m *testing.M) int {
	dir, err := os.MkdirTemp("", "kira-contract")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)

	kiraBin = filepath.Join(dir, "kira")
	build := exec.Command("go", "build", "-o", kiraBin, "github.com/shivamshivanshu/kira/cmd/kira")
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		panic("build kira: " + err.Error())
	}

	toolBin = filepath.Join(dir, "toolbin")
	if err := os.MkdirAll(toolBin, 0o777); err != nil {
		panic(err)
	}
	for _, tool := range []string{"git", "true"} {
		p, err := exec.LookPath(tool)
		if err != nil {
			panic(err)
		}
		if err := os.Symlink(p, filepath.Join(toolBin, tool)); err != nil {
			panic(err)
		}
	}
	return m.Run()
}

// baseEnv is the hermetic git environment shared by every exec: the tool PATH
// (rg/fzf deliberately absent) and neutralized global/system git config so the
// host's ~/.gitconfig can never leak into a fixture.
func baseEnv() []string {
	return []string{
		"PATH=" + toolBin,
		"GIT_CONFIG_GLOBAL=/dev/null",
		"GIT_CONFIG_SYSTEM=/dev/null",
	}
}

// kira runs the binary in dir and returns stdout, stderr, and the exit code.
func kira(t *testing.T, dir string, args ...string) (stdout, stderr string, code int) {
	t.Helper()
	cmd := exec.Command(kiraBin, args...)
	cmd.Dir = dir
	cmd.Env = append(baseEnv(), "HOME="+dir,
		"GIT_AUTHOR_NAME=tester", "GIT_AUTHOR_EMAIL=tester@example.com",
		"GIT_COMMITTER_NAME=tester", "GIT_COMMITTER_EMAIL=tester@example.com",
		"EDITOR=true",
	)
	var out, errBuf bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errBuf
	err := cmd.Run()
	code = 0
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			code = ee.ExitCode()
		} else {
			t.Fatalf("exec kira %v: %v", args, err)
		}
	}
	return out.String(), errBuf.String(), code
}

// withJSON returns a fresh args slice with --json appended, leaving the case's
// shared args slice untouched.
func withJSON(args []string) []string {
	return append(append([]string{}, args...), "--json")
}

// mustKira runs a setup command and fails the test on a non-zero exit, so a
// broken fixture surfaces as a fixture error, not a mismatched golden.
func mustKira(t *testing.T, dir string, args ...string) {
	t.Helper()
	if _, stderr, code := kira(t, dir, args...); code != 0 {
		t.Fatalf("setup kira %v: exit %d: %s", args, code, stderr)
	}
}

// gitRepo is a bare git repo with no .kira/ (the init and env-error fixtures).
func gitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	cmd := exec.Command("git", "init")
	cmd.Dir = dir
	cmd.Env = baseEnv()
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, out)
	}
	return dir
}

// kiraRepo is an initialized store with no items.
func kiraRepo(t *testing.T) string {
	t.Helper()
	dir := gitRepo(t)
	mustKira(t, dir, "init", "--key", "KIRA")
	return dir
}

// seededRepo is the canonical read fixture: an epic (KIRA-1), a fully populated
// ticket under it (KIRA-2) with owner/labels/estimate/priority/reporter plus
// the M1.5 parity fields (subtype/rank/sprint/due and a relates link), a
// blocked-by edge, a comment, and an IN_PROGRESS state, plus a plain blocker
// ticket (KIRA-3). It exercises every field the read shapes carry.
func seededRepo(t *testing.T) string {
	t.Helper()
	dir := kiraRepo(t)
	addSprint(t, dir)
	mustKira(t, dir, "create", "epic", "--title", "Epic one", "--no-edit")
	mustKira(t, dir, "create", "ticket", "--title", "Rich ticket", "--no-edit",
		"--owner", "shivam", "--label", "bug", "--label", "perf", "--estimate", "3", "--parent", "KIRA-1")
	mustKira(t, dir, "create", "ticket", "--title", "Blocker", "--no-edit")
	mustKira(t, dir, "edit", "KIRA-2", "--field", "priority=P1", "--field", "reporter=alice",
		"--subtype", "bug", "--rank", "0|hzzzzz:", "--sprint", "2026-S14", "--due", "2026-07-20")
	mustKira(t, dir, "link", "KIRA-2", "--blocked-by", "KIRA-3")
	mustKira(t, dir, "link", "KIRA-2", "--relates", "KIRA-3")
	mustKira(t, dir, "comment", "KIRA-2", "-m", "first comment")
	mustKira(t, dir, "move", "KIRA-2", "IN_PROGRESS")
	return dir
}

// addSprint swaps the scaffolded empty sprints list for one sprint, so the
// fixture can exercise the sprint field (a sprint key must exist in config).
func addSprint(t *testing.T, dir string) {
	t.Helper()
	path := filepath.Join(dir, ".kira", "config.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	patched := strings.Replace(string(data), "sprints: []",
		"sprints: [{ key: 2026-S14, name: Sprint 14, start: 2026-07-13, end: 2026-07-26 }]", 1)
	if patched == string(data) {
		t.Fatal("config template no longer has an empty sprints list to patch")
	}
	if err := os.WriteFile(path, []byte(patched), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
}

// oneTicket is the minimal write fixture: a single fresh ticket, KIRA-1.
func oneTicket(t *testing.T) string {
	t.Helper()
	dir := kiraRepo(t)
	mustKira(t, dir, "create", "ticket", "--title", "A ticket", "--no-edit")
	return dir
}

// reviewTicket is a ticket advanced to REVIEW, in front of the default config's
// guarded REVIEW -> DONE transition (require: [resolution]).
func reviewTicket(t *testing.T) string {
	t.Helper()
	dir := oneTicket(t)
	mustKira(t, dir, "move", "KIRA-1", "IN_PROGRESS")
	mustKira(t, dir, "move", "KIRA-1", "REVIEW")
	return dir
}

// wipLoaded fills REVIEW to its default wip: 2 limit with KIRA-1/KIRA-2 and
// stages KIRA-3 in IN_PROGRESS, so moving KIRA-3 to REVIEW breaches the limit.
func wipLoaded(t *testing.T) string {
	t.Helper()
	dir := kiraRepo(t)
	for _, title := range []string{"W1", "W2", "W3"} {
		mustKira(t, dir, "create", "ticket", "--title", title, "--no-edit")
	}
	for _, num := range []string{"KIRA-1", "KIRA-2"} {
		mustKira(t, dir, "move", num, "IN_PROGRESS")
		mustKira(t, dir, "move", num, "REVIEW")
	}
	mustKira(t, dir, "move", "KIRA-3", "IN_PROGRESS")
	return dir
}

// TestJSONContract is the golden shape suite: one case per command's --json
// output. Read cases are additionally asserted stable across a repeat run.
func TestJSONContract(t *testing.T) {
	cases := []struct {
		name     string
		repo     func(*testing.T) string
		args     []string
		readOnly bool // read commands: additionally asserted stable across a re-run
	}{
		{"init", gitRepo, []string{"init", "--key", "KIRA"}, false},
		{"create-ticket", kiraRepo, []string{"create", "ticket", "--title", "New ticket", "--no-edit"}, false},
		{"create-epic", kiraRepo, []string{"create", "epic", "--title", "New epic", "--no-edit"}, false},
		{"print-template", kiraRepo, []string{"create", "ticket", "--print-template"}, true},
		{"move", oneTicket, []string{"move", "KIRA-1", "IN_PROGRESS"}, false},
		{"move-resolution", oneTicket, []string{"move", "KIRA-1", "WONT_DO", "--resolution", "dropped"}, false},
		{"move-wip-warn", wipLoaded, []string{"move", "KIRA-3", "REVIEW"}, false},
		{"assign", oneTicket, []string{"assign", "KIRA-1", "shivam"}, false},
		{"link", seededRepo, []string{"link", "KIRA-2", "--blocked-by", "KIRA-1"}, false},
		{"link-relates", seededRepo, []string{"link", "KIRA-2", "--relates", "KIRA-1"}, false},
		{"edit", oneTicket, []string{"edit", "KIRA-1", "--field", "title=Renamed"}, false},
		{"edit-parity", oneTicket, []string{"edit", "KIRA-1", "--subtype", "task", "--priority", "P2", "--rank", "0|m:", "--due", "2026-08-01"}, false},
		{"comment", oneTicket, []string{"comment", "KIRA-1", "-m", "hello"}, false},
		{"show", seededRepo, []string{"show", "KIRA-2"}, true},
		{"list", seededRepo, []string{"list"}, true},
		{"list-tree", seededRepo, []string{"list", "--tree"}, true},
		{"query-tree", seededRepo, []string{"query", "type=ticket"}, true},
		{"query-flat", seededRepo, []string{"query", "type=ticket", "--flat"}, true},
		{"tree", seededRepo, []string{"tree"}, true},
		{"tree-epic", seededRepo, []string{"tree", "KIRA-1"}, true},
		{"find", seededRepo, []string{"find", "Blocker"}, true},
		{"sprint-create", kiraRepo, []string{"sprint", "create", "--key", "2026-S15", "--name", "Sprint 15", "--start", "2026-07-27", "--end", "2026-08-09"}, false},
		{"sprint-list", seededRepo, []string{"sprint", "list"}, true},
		{"sprint-activate", seededRepo, []string{"sprint", "activate", "2026-S14"}, false},
		{"sprint-close", seededRepo, []string{"sprint", "close", "2026-S14"}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dir := c.repo(t)
			out, stderr, code := kira(t, dir, withJSON(c.args)...)
			if code != 0 {
				t.Fatalf("exit %d, stderr: %s", code, stderr)
			}
			got := scrub(out, dir)
			checkGolden(t, c.name+".json", got)
			if c.readOnly {
				out2, _, _ := kira(t, dir, withJSON(c.args)...)
				if got != scrub(out2, dir) {
					t.Errorf("%s not stable across runs:\n%s\n---\n%s", c.name, out, out2)
				}
			}
		})
	}
}

// TestJSONErrors is the failure-shape + stream-discipline suite: each documented
// error class exits with its policy code (docs/design/04-cli.md §1), keeps
// stdout empty even under --json, and reports on stderr.
func TestJSONErrors(t *testing.T) {
	cases := []struct {
		name     string
		repo     func(*testing.T) string
		args     []string
		wantCode int
	}{
		{"err-unknown-id", seededRepo, []string{"show", "KIRA-999"}, 1},
		{"err-bad-transition", oneTicket, []string{"move", "KIRA-1", "DONE"}, 1},
		{"err-require-guard", reviewTicket, []string{"move", "KIRA-1", "DONE"}, 1},
		{"err-bad-field", oneTicket, []string{"edit", "KIRA-1", "--field", "nope=x"}, 1},
		{"err-unknown-sprint", oneTicket, []string{"edit", "KIRA-1", "--sprint", "2099-S1"}, 1},
		{"err-init-exists", kiraRepo, []string{"init", "--key", "KIRA"}, 1},
		{"err-no-store", gitRepo, []string{"show", "KIRA-1"}, 3},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dir := c.repo(t)
			out, stderr, code := kira(t, dir, withJSON(c.args)...)
			if code != c.wantCode {
				t.Errorf("exit code = %d, want %d (stderr: %s)", code, c.wantCode, stderr)
			}
			if out != "" {
				t.Errorf("stdout must be empty on error, got: %q", out)
			}
			if stderr == "" {
				t.Errorf("stderr must carry the error, got empty")
			}
			checkGolden(t, c.name+".err", scrub(stderr, dir))
		})
	}
}

var (
	ulidRE = regexp.MustCompile(`[0-9A-HJKMNP-TV-Z]{26}`)
	tsRE   = regexp.MustCompile(`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:\d{2})`)
)

// scrub normalizes the volatile parts of an output so goldens are byte-stable:
// each distinct ULID maps to <ULID-n> by first appearance (cross-references stay
// visible), timestamps collapse to <TS>, and the repo path collapses to <DIR>.
func scrub(s, dir string) string {
	if dir != "" {
		s = strings.ReplaceAll(s, dir, "<DIR>")
	}
	seen := map[string]string{}
	s = ulidRE.ReplaceAllStringFunc(s, func(u string) string {
		if r, ok := seen[u]; ok {
			return r
		}
		r := "<ULID-" + strconv.Itoa(len(seen)+1) + ">"
		seen[u] = r
		return r
	})
	return tsRE.ReplaceAllString(s, "<TS>")
}

func checkGolden(t *testing.T, name, got string) {
	t.Helper()
	path := filepath.Join("testdata", "golden", name)
	if *update {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden %s: %v (run: go test ./internal/contract -update)", name, err)
	}
	if got != string(want) {
		t.Errorf("golden %s mismatch\n--- got ---\n%s\n--- want ---\n%s", name, got, want)
	}
}
