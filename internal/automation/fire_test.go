package automation

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func TestMatchedHooksFiltersByEnabledAndEvent(t *testing.T) {
	disabled := false
	hooks := []datamodel.AutomationHook{
		{Name: "enabled-match", On: datamodel.EventItemCreated, Run: "true"},
		{Name: "disabled", On: datamodel.EventItemCreated, Run: "true", Enabled: &disabled},
		{Name: "other-event", On: datamodel.EventItemStateChanged, Run: "true"},
	}
	got := matchedHooks(hooks, Event{Name: datamodel.EventItemCreated})
	if len(got) != 1 || got[0].Name != "enabled-match" {
		t.Fatalf("matchedHooks = %+v, want only the enabled hook on the fired event", got)
	}
}

func TestFiringSetTrustPartition(t *testing.T) {
	repo := []datamodel.AutomationHook{{Name: "repo"}}
	user := []datamodel.AutomationHook{{Name: "user"}}

	untrusted := firingSet(false, repo, user)
	if len(untrusted) != 1 || untrusted[0].Name != "user" {
		t.Fatalf("untrusted firing set = %+v, want user hooks only", untrusted)
	}

	trusted := firingSet(true, repo, user)
	if len(trusted) != 2 || trusted[0].Name != "repo" || trusted[1].Name != "user" {
		t.Fatalf("trusted firing set = %+v, want repo then user", trusted)
	}
}

func nonEmptyLines(s string) []string {
	var lines []string
	for _, l := range strings.Split(s, "\n") {
		if l != "" {
			lines = append(lines, l)
		}
	}
	return lines
}

func TestRunHookShellContract(t *testing.T) {
	cases := []struct {
		name string
		hook datamodel.AutomationHook
		want []string
	}{
		{
			"quoted arg preserved",
			datamodel.AutomationHook{Name: "quote", Run: `printf '%s\n' "one two"`},
			[]string{"[automation:quote] one two"},
		},
		{
			"shell operators work",
			datamodel.AutomationHook{Name: "pipe", Run: "echo a && echo b"},
			[]string{"[automation:pipe] a", "[automation:pipe] b"},
		},
		{
			"timeout kills the hook",
			datamodel.AutomationHook{Name: "slow", Run: "sleep 5", Timeout: "50ms"},
			[]string{"[automation:slow] timed out after 50ms"},
		},
		{
			"failure exit propagates",
			datamodel.AutomationHook{Name: "fail", Run: "false"},
			[]string{"[automation:fail] exit status 1"},
		},
		{
			"both streams prefixed",
			datamodel.AutomationHook{Name: "streams", Run: "echo out; echo err >&2"},
			[]string{"[automation:streams] out", "[automation:streams] err"},
		},
		{
			"blank run is a no-op",
			datamodel.AutomationHook{Name: "blank", Run: "   "},
			nil,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var w bytes.Buffer
			runHook(&w, t.TempDir(), tc.hook, nil, os.Environ())
			if got := nonEmptyLines(w.String()); !slices.Equal(got, tc.want) {
				t.Errorf("runHook output = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestRunHookKillsBackgroundedChildOnTimeout(t *testing.T) {
	dir := t.TempDir()
	sentinel := filepath.Join(dir, "survived")
	hook := datamodel.AutomationHook{
		Name:    "leak",
		Run:     "(sleep 1; touch " + sentinel + ") | cat",
		Timeout: "200ms",
	}
	var w bytes.Buffer
	runHook(&w, dir, hook, nil, os.Environ())

	time.Sleep(2 * time.Second)
	if _, err := os.Stat(sentinel); err == nil {
		t.Fatal("backgrounded child outlived the hook timeout: process group not killed")
	}
	if !strings.Contains(w.String(), "timed out after 200ms") {
		t.Errorf("output = %q, want timeout notice", w.String())
	}
}

func TestFireRecursionGuardSkipsHooks(t *testing.T) {
	t.Setenv(RecursionGuardEnv, "1")
	var w bytes.Buffer
	cfg := &datamodel.Config{UserAutomation: []datamodel.AutomationHook{{Name: "guarded", On: datamodel.EventItemCreated, Run: "echo fired"}}}
	Fire(&w, t.TempDir(), t.TempDir(), cfg, Event{Name: datamodel.EventItemCreated}, func() Actor { return Actor{} })
	if w.Len() != 0 {
		t.Fatalf("Fire under recursion guard wrote %q, want nothing", w.String())
	}
}

func TestEnvMirrorAgreesWithPayloadItem(t *testing.T) {
	item := &datamodel.ShowResult{ID: "01ABC", Number: "KIRA-9", Type: "epic", Title: "Widen scope"}
	ev := Event{Name: datamodel.EventItemCreated, Item: item}

	env := envMirror(ev, "/repo")
	get := func(key string) string {
		prefix := key + "="
		for _, kv := range env {
			if strings.HasPrefix(kv, prefix) {
				return strings.TrimPrefix(kv, prefix)
			}
		}
		t.Fatalf("env var %s not set", key)
		return ""
	}
	if got := get("KIRA_ITEM"); got != item.ID {
		t.Errorf("KIRA_ITEM = %q, want %q", got, item.ID)
	}
	if got := get("KIRA_NUMBER"); got != item.Number {
		t.Errorf("KIRA_NUMBER = %q, want %q", got, item.Number)
	}
	if got := get("KIRA_TYPE"); got != item.Type {
		t.Errorf("KIRA_TYPE = %q, want %q", got, item.Type)
	}
	if got := get("KIRA_TITLE"); got != item.Title {
		t.Errorf("KIRA_TITLE = %q, want %q", got, item.Title)
	}

	raw, err := Payload(ev, "/repo", "2026-07-16T00:00:00Z", Actor{})
	if err != nil {
		t.Fatalf("payload: %v", err)
	}
	var got HookPayload
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Item.Number != item.Number || got.Item.Type != item.Type {
		t.Fatalf("json payload item = %+v, want it to match the env-mirrored item", got.Item)
	}
}

func TestFireExportsEventEnvAndGuard(t *testing.T) {
	t.Setenv(RecursionGuardEnv, "")
	var w bytes.Buffer
	hook := datamodel.AutomationHook{Name: "env", On: datamodel.EventItemStateChanged, Run: `printf '%s %s %s\n' "$KIRA_EVENT" "$KIRA_TO" "$` + RecursionGuardEnv + `"`}
	cfg := &datamodel.Config{UserAutomation: []datamodel.AutomationHook{hook}}
	Fire(&w, t.TempDir(), t.TempDir(), cfg, Event{Name: datamodel.EventItemStateChanged, To: "DONE"}, func() Actor { return Actor{Name: "t"} })
	want := "[automation:env] " + string(datamodel.EventItemStateChanged) + " DONE 1\n"
	if w.String() != want {
		t.Fatalf("Fire output = %q, want %q", w.String(), want)
	}
}
