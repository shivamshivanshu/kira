package automation

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func Fire(w io.Writer, root, cacheDir string, cfg *datamodel.Config, ev Event, actor func() Actor) {
	if os.Getenv(RecursionGuardEnv) != "" {
		return
	}
	var matched []datamodel.AutomationHook
	for _, h := range cfg.Automation {
		if h.IsEnabled() && Matches(h, ev) {
			matched = append(matched, h)
		}
	}
	if len(matched) == 0 {
		return
	}
	if !Trusted(cacheDir, cfg) {
		fmt.Fprintf(w, "kira: %d automation hooks defined but not trusted — run `kira automation trust`\n", len(cfg.Automation))
		return
	}
	stdin, err := Payload(ev, root, time.Now().Format(time.RFC3339), actor())
	if err != nil {
		fmt.Fprintf(w, "kira: automation: building payload: %v\n", err)
		return
	}
	env := append(os.Environ(), envMirror(ev, root)...)
	for _, h := range matched {
		runHook(w, root, h, stdin, env)
	}
}

func runHook(w io.Writer, root string, h datamodel.AutomationHook, stdin []byte, env []string) {
	name := hookName(h)
	argv := strings.Fields(h.Run)
	if len(argv) == 0 {
		return
	}
	timeout, _ := h.TimeoutDuration()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	cmd.Dir = root
	cmd.Env = env
	cmd.Stdin = bytes.NewReader(stdin)
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
	return h.On
}

func emitPrefixed(w io.Writer, name string, out []byte) {
	for _, line := range strings.Split(strings.TrimRight(string(out), "\n"), "\n") {
		if line == "" {
			continue
		}
		fmt.Fprintf(w, "[automation:%s] %s\n", name, line)
	}
}
