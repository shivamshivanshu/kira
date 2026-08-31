package e2e

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/shivamshivanshu/kira/internal/testutil"
)

func TestServerDashboardLifecycle(t *testing.T) {
	bin := testutil.KiraBinary(t)
	dir, env := serverFixture(t, bin)
	const title = "Dashboard selected repository"
	mustRun(t, dir, env, bin, "create", "ticket", "--title", title, "--no-edit")

	for _, name := range []string{"server", "serve"} {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, bin, "-C", dir, name, "--listen", "127.0.0.1:0")
			cmd.Dir = t.TempDir()
			cmd.Env = env
			var stderr bytes.Buffer
			cmd.Stderr = &stderr
			stdout, err := cmd.StdoutPipe()
			if err != nil {
				t.Fatal(err)
			}
			if err := cmd.Start(); err != nil {
				t.Fatal(err)
			}
			done := make(chan struct{})
			var exitErr error
			go func() {
				exitErr = cmd.Wait()
				close(done)
			}()
			t.Cleanup(func() {
				cancel()
				<-done
			})
			ready := make(chan string, 1)
			go func() {
				line, _ := bufio.NewReader(stdout).ReadString('\n')
				ready <- strings.TrimSpace(line)
			}()

			var line string
			select {
			case line = <-ready:
			case <-done:
				t.Fatalf("server exited before announcing URL: %v\n%s", exitErr, stderr.String())
			case <-ctx.Done():
				<-done
				t.Fatalf("server did not announce its URL: %v\n%s", exitErr, stderr.String())
			}
			address, ok := strings.CutPrefix(line, "kira dashboard: ")
			if !ok {
				cancel()
				<-done
				t.Fatalf("startup output = %q, want dashboard URL; exit: %v\n%s", line, exitErr, stderr.String())
			}
			parsed, err := url.Parse(address)
			if err != nil || parsed.Scheme != "http" || parsed.Hostname() != "127.0.0.1" || parsed.Port() == "" || parsed.Port() == "0" {
				t.Fatalf("dashboard URL = %q, want bound loopback HTTP address", address)
			}
			client := &http.Client{Timeout: 5 * time.Second}
			response, err := client.Get(address)
			if err != nil {
				t.Fatalf("GET dashboard: %v", err)
			}
			body, err := io.ReadAll(response.Body)
			if closeErr := response.Body.Close(); closeErr != nil {
				t.Fatal(closeErr)
			}
			if err != nil {
				t.Fatal(err)
			}
			if response.StatusCode != http.StatusOK || !strings.HasPrefix(response.Header.Get("Content-Type"), "text/html") {
				t.Fatalf("GET dashboard: status %d, content type %q\n%s", response.StatusCode, response.Header.Get("Content-Type"), body)
			}
			if !bytes.Contains(body, []byte(title)) {
				t.Fatalf("dashboard did not load the -C repository's ticket:\n%s", body)
			}
			if name == "server" {
				token := regexp.MustCompile(`name="csrf" value="([^"]+)"`).FindSubmatch(body)
				if len(token) != 2 {
					t.Fatal("dashboard did not render a CSRF token")
				}
				client.CheckRedirect = func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }
				response, err := client.PostForm(address+"/actions/command", url.Values{
					"csrf":      {string(token[1])},
					"line":      {"move KIRA-1 IN_PROGRESS"},
					"return_to": {"/?item=KIRA-1"},
				})
				if err != nil {
					t.Fatalf("POST command: %v", err)
				}
				body, err := io.ReadAll(response.Body)
				if closeErr := response.Body.Close(); closeErr != nil {
					t.Fatal(closeErr)
				}
				if err != nil {
					t.Fatal(err)
				}
				if response.StatusCode != http.StatusSeeOther {
					t.Fatalf("POST command: status %d, want %d\n%s", response.StatusCode, http.StatusSeeOther, body)
				}
				show := exec.CommandContext(ctx, bin, "-C", dir, "show", "KIRA-1", "--json")
				show.Dir = cmd.Dir
				show.Env = env
				output, err := show.Output()
				if err != nil {
					t.Fatalf("show after web command: %v", err)
				}
				var item struct{ State string }
				if err := json.Unmarshal(output, &item); err != nil {
					t.Fatalf("decode show: %v\n%s", err, output)
				}
				if item.State != "IN_PROGRESS" {
					t.Fatalf("state after web command = %q, want IN_PROGRESS", item.State)
				}
			}
			if err := cmd.Process.Signal(os.Interrupt); err != nil {
				t.Fatalf("interrupt server: %v", err)
			}
			select {
			case <-done:
				if exitErr != nil {
					t.Fatalf("server shutdown: %v\n%s", exitErr, stderr.String())
				}
			case <-ctx.Done():
				t.Fatal("server did not shut down after interrupt")
			}
		})
	}
}

func TestServerRejectsUnsupportedFlags(t *testing.T) {
	bin := testutil.KiraBinary(t)
	dir, env := serverFixture(t, bin)
	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{"non-loopback", []string{"--listen", "0.0.0.0:0"}, "loopback"},
		{"wildcard", []string{"--listen", ":0"}, "loopback"},
		{"json", []string{"--json", "--listen", "127.0.0.1:0"}, "json"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, bin, append([]string{"server"}, tc.args...)...)
			cmd.Dir = dir
			cmd.Env = env
			output, err := cmd.CombinedOutput()
			if ctx.Err() != nil {
				t.Fatalf("server did not reject %v before timeout", tc.args)
			}
			var exitError *exec.ExitError
			if !errors.As(err, &exitError) {
				t.Fatalf("server %v: error = %v, want nonzero exit\n%s", tc.args, err, output)
			}
			if !strings.Contains(strings.ToLower(string(output)), tc.want) {
				t.Fatalf("server %v: output = %q, want %q", tc.args, output, tc.want)
			}
		})
	}
}

func serverFixture(t *testing.T, bin string) (string, []string) {
	t.Helper()
	dir := t.TempDir()
	env := append(testutil.HermeticEnvironment(), "PATH="+pathEnv())
	mustRun(t, dir, env, "git", "init")
	mustRun(t, dir, env, "git", "config", "user.email", "test@example.com")
	mustRun(t, dir, env, "git", "config", "user.name", "tester")
	mustRun(t, dir, env, bin, "init", "--key", "KIRA")
	return dir, env
}
