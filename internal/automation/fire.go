package automation

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func Fire(w io.Writer, root, cacheDir string, cfg *datamodel.Config, ev Event, actor func() Actor) {
	if os.Getenv(RecursionGuardEnv) != "" {
		return
	}
	repo := matchedHooks(cfg.Automation, ev)
	user := matchedHooks(cfg.UserAutomation, ev)
	if len(repo) == 0 && len(user) == 0 {
		return
	}
	repoMayFire := len(repo) == 0 || Trusted(cacheDir, cfg)
	if len(repo) > 0 && !repoMayFire {
		hookWord := "hooks"
		if len(repo) == 1 {
			hookWord = "hook"
		}
		fmt.Fprintf(w, "kira: %d automation %s defined but not trusted — run `kira automation trust`\n", len(repo), hookWord)
	}
	run := firingSet(repoMayFire, repo, user)
	if len(run) == 0 {
		return
	}
	stdin, err := Payload(ev, root, time.Now().Format(time.RFC3339), actor())
	if err != nil {
		fmt.Fprintf(w, "kira: automation: building payload: %v\n", err)
		return
	}
	env := append(os.Environ(), envMirror(ev, root)...)
	for _, h := range run {
		runHook(w, root, h, stdin, env)
	}
}

func firingSet(repoMayFire bool, repo, user []datamodel.AutomationHook) []datamodel.AutomationHook {
	if repoMayFire {
		return append(slices.Clone(repo), user...)
	}
	return user
}

func matchedHooks(hooks []datamodel.AutomationHook, ev Event) []datamodel.AutomationHook {
	var matched []datamodel.AutomationHook
	for _, h := range hooks {
		if h.IsEnabled() && Matches(h, ev) {
			matched = append(matched, h)
		}
	}
	return matched
}

func envMirror(ev Event, root string) []string {
	return []string{
		RecursionGuardEnv + "=1",
		"KIRA_EVENT=" + string(ev.Name),
		"KIRA_ITEM=" + ev.itemID(),
		"KIRA_NUMBER=" + ev.itemNumber(),
		"KIRA_TYPE=" + ev.itemType(),
		"KIRA_TITLE=" + ev.itemTitle(),
		"KIRA_FROM=" + ev.From,
		"KIRA_TO=" + ev.To,
		"KIRA_TO_CATEGORY=" + ev.ToCategory,
		"KIRA_FROM_CATEGORY=" + ev.FromCategory,
		"KIRA_SOURCE=" + string(ev.Source),
		"KIRA_ROOT=" + root,
		"KIRA_COMMIT=" + ev.Commit,
	}
}

func runHook(w io.Writer, root string, h datamodel.AutomationHook, stdin []byte, env []string) {
	name := hookName(h)
	run := strings.TrimSpace(h.Run)
	if run == "" {
		return
	}
	timeout, _ := h.TimeoutDuration()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sh", "-c", run)
	cmd.Dir = root
	cmd.Env = env
	cmd.Stdin = bytes.NewReader(stdin)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) }
	cmd.WaitDelay = time.Second
	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &out
	err := cmd.Run()

	emitPrefixed(w, name, out.Bytes())
	switch {
	case ctx.Err() == context.DeadlineExceeded:
		fmt.Fprintf(w, "[automation:%s] timed out after %s\n", name, timeout)
	case err != nil:
		fmt.Fprintf(w, "[automation:%s] %v\n", name, err)
	}
}

func hookName(h datamodel.AutomationHook) string {
	if h.Name != "" {
		return h.Name
	}
	return string(h.On)
}

func emitPrefixed(w io.Writer, name string, out []byte) {
	for _, line := range strings.Split(strings.TrimRight(string(out), "\n"), "\n") {
		if line == "" {
			continue
		}
		fmt.Fprintf(w, "[automation:%s] %s\n", name, line)
	}
}
