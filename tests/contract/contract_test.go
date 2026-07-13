// Package contract execs the built kira binary and compares each command's --json output to a checked-in golden.
package contract

import (
	"bytes"
	"encoding/json"
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

// rg/fzf are deliberately absent from toolBin, so find always takes its
// deterministic pure-Go fallback regardless of what the host has installed.
func baseEnv() []string {
	return []string{
		"PATH=" + toolBin,
		"GIT_CONFIG_GLOBAL=/dev/null",
		"GIT_CONFIG_SYSTEM=/dev/null",
	}
}

func kira(t *testing.T, dir string, args ...string) (stdout, stderr string, code int) {
	t.Helper()
	cmd := exec.Command(kiraBin, args...)
	cmd.Dir = dir
	cmd.Env = append(baseEnv(), "HOME="+dir,
		"GIT_AUTHOR_NAME=tester", "GIT_AUTHOR_EMAIL=tester@example.com",
		"GIT_COMMITTER_NAME=tester", "GIT_COMMITTER_EMAIL=tester@example.com",
		"GIT_AUTHOR_DATE=2026-07-13T12:00:00Z", "GIT_COMMITTER_DATE=2026-07-13T12:00:00Z",
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

func withJSON(args []string) []string {
	return append(append([]string{}, args...), "--json")
}

func mustKira(t *testing.T, dir string, args ...string) {
	t.Helper()
	if _, stderr, code := kira(t, dir, args...); code != 0 {
		t.Fatalf("setup kira %v: exit %d: %s", args, code, stderr)
	}
}

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

func kiraRepo(t *testing.T) string {
	t.Helper()
	dir := gitRepo(t)
	mustKira(t, dir, "init", "--key", "KIRA")
	return dir
}

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

func addSprint(t *testing.T, dir string) {
	t.Helper()
	mustKira(t, dir, "sprint", "create", "--key", "2026-S14", "--name", "Sprint 14",
		"--start", "2026-07-13", "--end", "2026-07-26")
}

func oneTicket(t *testing.T) string {
	t.Helper()
	dir := kiraRepo(t)
	mustKira(t, dir, "create", "ticket", "--title", "A ticket", "--no-edit")
	return dir
}

func automationRepo(t *testing.T) string {
	t.Helper()
	dir := kiraRepo(t)
	cfgPath := filepath.Join(dir, ".kira", "config.yaml")
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	block := "\nautomation:\n  - name: notify\n    on: item.state_changed\n    match:\n      to: done\n    run: bash notify.sh\n"
	if err := os.WriteFile(cfgPath, append(data, block...), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return dir
}

func reviewTicket(t *testing.T) string {
	t.Helper()
	dir := oneTicket(t)
	mustKira(t, dir, "move", "KIRA-1", "IN_PROGRESS")
	mustKira(t, dir, "move", "KIRA-1", "REVIEW")
	return dir
}

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

func gitCmd(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(baseEnv(), "HOME="+dir,
		"GIT_AUTHOR_NAME=tester", "GIT_AUTHOR_EMAIL=tester@example.com",
		"GIT_COMMITTER_NAME=tester", "GIT_COMMITTER_EMAIL=tester@example.com")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v: %s", args, err, out)
	}
}

func gitOutput(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(baseEnv(), "HOME="+dir)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git %v: %v", args, err)
	}
	return strings.TrimSpace(string(out))
}

func mutateTicketFile(t *testing.T, dir, number, commitMsg string, transform func(string) string) {
	t.Helper()
	tdir := filepath.Join(dir, ".kira", "tickets")
	entries, err := os.ReadDir(tdir)
	if err != nil {
		t.Fatalf("read tickets: %v", err)
	}
	for _, e := range entries {
		p := filepath.Join(tdir, e.Name())
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		s := string(data)
		if !strings.Contains(s, "number: "+number+"\n") {
			continue
		}
		if err := os.WriteFile(p, []byte(transform(s)), 0o644); err != nil {
			t.Fatalf("write ticket %s: %v", number, err)
		}
		gitCmd(t, dir, "add", "-A")
		gitCmd(t, dir, "commit", "-m", commitMsg)
		return
	}
	t.Fatalf("ticket %s not found", number)
}

func renumberTicket(t *testing.T, dir, from, to string) {
	t.Helper()
	mutateTicketFile(t, dir, from, "renumber", func(s string) string {
		s = strings.Replace(s, "number: "+from+"\n", "number: "+to+"\n", 1)
		return strings.Replace(s, "aliases: []\n", "aliases: ["+from+"]\n", 1)
	})
}

func diffFixture(t *testing.T) string {
	t.Helper()
	dir := kiraRepo(t)
	mustKira(t, dir, "create", "ticket", "--title", "First", "--no-edit")
	mustKira(t, dir, "create", "ticket", "--title", "Second", "--no-edit")
	base := gitOutput(t, dir, "branch", "--show-current")
	gitCmd(t, dir, "checkout", "-b", "later")
	mustKira(t, dir, "move", "KIRA-1", "IN_PROGRESS")
	mustKira(t, dir, "create", "ticket", "--title", "Third", "--no-edit")
	renumberTicket(t, dir, "KIRA-2", "KIRA-9")
	gitCmd(t, dir, "checkout", base)
	return dir
}

func changesFixture(t *testing.T) string {
	t.Helper()
	dir := kiraRepo(t)
	mustKira(t, dir, "create", "ticket", "--title", "First", "--no-edit")
	mustKira(t, dir, "create", "ticket", "--title", "Second", "--no-edit")
	gitCmd(t, dir, "tag", "base")
	mustKira(t, dir, "move", "KIRA-1", "IN_PROGRESS")
	editBody(t, dir, "KIRA-1", "A body-only detail line.")
	mustKira(t, dir, "edit", "KIRA-2", "--field", "priority=P1")
	mustKira(t, dir, "create", "ticket", "--title", "Third", "--no-edit")
	renumberTicket(t, dir, "KIRA-2", "KIRA-9")
	return dir
}

func editBody(t *testing.T, dir, number, text string) {
	t.Helper()
	mutateTicketFile(t, dir, number, "body edit", func(s string) string {
		return strings.Replace(s, "## Description\n", "## Description\n\n"+text+"\n", 1)
	})
}

func TestJSONContract(t *testing.T) {
	cases := []struct {
		name     string
		repo     func(*testing.T) string
		args     []string
		readOnly bool
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
		{"list-query-tree", seededRepo, []string{"list", "--query", "type=ticket", "--tree"}, true},
		{"list-query-flat", seededRepo, []string{"list", "--query", "type=ticket"}, true},
		{"tree", seededRepo, []string{"tree"}, true},
		{"tree-epic", seededRepo, []string{"tree", "KIRA-1"}, true},
		{"find", seededRepo, []string{"find", "Blocker"}, true},
		{"diff", diffFixture, []string{"diff", "later", "--incoming"}, true},
		{"changes", changesFixture, []string{"changes", "--since", "base"}, true},
		{"sprint-create", kiraRepo, []string{"sprint", "create", "--key", "2026-S15", "--name", "Sprint 15", "--start", "2026-07-27", "--end", "2026-08-09"}, false},
		{"sprint-list", seededRepo, []string{"sprint", "list"}, true},
		{"sprint-activate", seededRepo, []string{"sprint", "activate", "2026-S14"}, false},
		{"sprint-close", seededRepo, []string{"sprint", "close", "2026-S14"}, true},
		{"index", seededRepo, []string{"index"}, false},
		{"doctor", kiraRepo, []string{"doctor"}, true},
		{"log", seededRepo, []string{"log", "KIRA-2"}, true},
		{"stats", seededRepo, []string{"stats"}, true},
		{"blame", seededRepo, []string{"blame", "KIRA-2"}, true},
		{"automation-list", automationRepo, []string{"automation", "list"}, true},
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

func TestJSONErrors(t *testing.T) {
	cases := []struct {
		name     string
		repo     func(*testing.T) string
		args     []string
		wantCode int
	}{
		{"err-unknown-id", seededRepo, []string{"show", "KIRA-999"}, 1},
		{"err-unknown-id-suggest", seededRepo, []string{"show", "KIRA-1X"}, 1},
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
			var obj struct {
				Error string `json:"error"`
				Hint  string `json:"hint"`
				Code  int    `json:"code"`
			}
			if err := json.Unmarshal([]byte(stderr), &obj); err != nil {
				t.Errorf("stderr is not a JSON error object: %v (got: %s)", err, stderr)
			} else if obj.Error == "" || obj.Code != c.wantCode {
				t.Errorf("json error object malformed: %+v", obj)
			}
			checkGolden(t, c.name+".err", scrub(stderr, dir))
		})
	}
}

var (
	ulidRE = regexp.MustCompile(`[0-9A-HJKMNP-TV-Z]{26}`)
	tsRE   = regexp.MustCompile(`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:\d{2})`)
	shaRE  = regexp.MustCompile(`\b[0-9a-f]{40}\b`)
)

func scrub(s, dir string) string {
	if dir != "" {
		s = strings.ReplaceAll(s, dir, "<DIR>")
	}
	s = shaRE.ReplaceAllString(s, "<SHA>")
	seen := map[string]string{}
	s = ulidRE.ReplaceAllStringFunc(s, func(u string) string {
		if r, ok := seen[u]; ok {
			return r
		}
		r := "<ULID-" + strconv.Itoa(len(seen)+1) + ">"
		seen[u] = r
		return r
	})
	s = shaRE.ReplaceAllString(s, "<SHA>")
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
		t.Fatalf("read golden %s: %v (run: go test ./tests/contract -update)", name, err)
	}
	if got != string(want) {
		t.Errorf("golden %s mismatch\n--- got ---\n%s\n--- want ---\n%s", name, got, want)
	}
}
