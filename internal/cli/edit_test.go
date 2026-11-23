package cli

import (
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/shivamshivanshu/kira/internal/codec"
	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/gitx"
	"github.com/shivamshivanshu/kira/internal/storage"
)

func TestEditPromotesTicketToEpic(t *testing.T) {
	for _, tc := range []struct {
		source, target string
		subtype        *string
		manual         bool
	}{
		{source: "TODO", target: "PLANNED", subtype: new("bug")},
		{source: "REVIEW", target: "ACTIVE", subtype: new("task"), manual: true},
		{source: "DONE", target: "DONE"},
	} {
		t.Run(tc.source, func(t *testing.T) {
			dir := initFixture(t)
			s, cfg := reopen(t, dir)
			parent, err := s.Create(cfg, core.CreateOpts{Type: datamodel.TypeEpic, Title: "Parent", NoEdit: true})
			if err != nil {
				t.Fatal(err)
			}
			blocker, err := s.Create(cfg, core.CreateOpts{Type: datamodel.TypeTicket, Title: "Blocker", NoEdit: true})
			if err != nil {
				t.Fatal(err)
			}
			it := &datamodel.Item{
				ID: "01HZ0TEST00000000000000000", Number: "KIRA-3", Aliases: []string{"KIRA-99"}, Title: "Ticket to promote",
				Type: datamodel.TypeTicket, State: tc.source, Subtype: tc.subtype, Resolution: new(datamodel.ResolutionDropped),
				Priority: new("P1"), Rank: new("a"), Owner: new("alice"), Reporter: new("bob"),
				Parent: &parent.ID, BlockedBy: []string{blocker.ID}, Labels: []string{"team"},
				Links: map[string][]string{"relates": {blocker.ID}}, Due: new("2026-12-01"), Estimate: new(2.5),
				Created: "2026-01-01T10:00:00Z", Updated: "2026-01-02T10:00:00Z",
			}
			it.Body = codec.AppendComment("## Description\n\nSaved details\n", datamodel.Comment{
				ID: "01HZ0TEST00000000000000001", Author: "charlie", Ts: it.Updated, Body: "Saved comment",
			})
			fs := storage.New(dir)
			if _, err := fs.WriteItem(it); err != nil {
				t.Fatal(err)
			}
			git := promotionGit(t, dir)
			git("add", ".kira")
			git("commit", "-m", "ticket fixture")
			if tc.manual {
				if _, err := s.ConfigSet(cfg, "commit.mode", "manual"); err != nil {
					t.Fatal(err)
				}
			}
			head := git("rev-parse", "HEAD")
			path := fs.ItemPath(it.ID)
			before := readPromotionItem(t, path)
			run := commandRunner(&globalFlags{chdir: dir})
			out, err := run([]string{"edit", it.Aliases[0], "--to-epic", "--json"})
			var result datamodel.MutationResult
			if err != nil || json.Unmarshal([]byte(out), &result) != nil {
				t.Fatalf("promote = %q, %v", out, err)
			}
			got := readPromotionItem(t, path)
			want := *before
			want.Type, want.State, want.Subtype = datamodel.TypeEpic, tc.target, nil
			if tc.target != "DONE" {
				want.Resolution = nil
			}
			want.Updated = got.Updated
			wantChanged := datamodel.ChangedFields(before, &want)
			if got.Updated == before.Updated || !reflect.DeepEqual(*got, want) ||
				result.ID != it.ID || result.Number != it.Number || !slices.Equal(result.Changed, wantChanged) {
				t.Fatalf("promoted item = %+v, want %+v, JSON = %q", got, want, out)
			}
			staged := git("diff", "--cached", "--name-only")
			if tc.manual {
				if git("rev-parse", "HEAD") != head || staged != fs.RelToRoot(path) || git("diff", "--name-only") != "" {
					t.Fatalf("manual promotion did not stage only the item: %q", staged)
				}
			} else if git("rev-parse", "HEAD^") != head || git("status", "--porcelain") != "" {
				t.Fatal("auto promotion did not make one clean commit")
			}
			if tc.target == "DONE" {
				if !slices.Equal(result.Changed, []string{datamodel.KeyType}) {
					t.Fatalf("type-only promotion reported %v", result.Changed)
				}
				out, err := run([]string{"diff", head, "--json"})
				var diff datamodel.DiffResult
				if err != nil || json.Unmarshal([]byte(out), &diff) != nil || len(diff.Items) != 1 ||
					!slices.Equal(diff.Items[0].Changes, []datamodel.FieldChange{{Field: "type", From: "ticket", To: "epic"}}) {
					t.Fatalf("committed diff = %q, %v", out, err)
				}
				if out, err := run([]string{"create", "ticket", "--title", "Child", "--parent", it.Number, "--no-edit"}); err != nil {
					t.Fatalf("attach child = %q, %v", out, err)
				}
				_ = verifyPromotionRejections(t, dir, path, [][]string{{"edit", it.Number, "--to-epic"}}, "only tickets")
			}
		})
	}
}

func TestEditRejectsInvalidPromotionWithoutMutation(t *testing.T) {
	dir := initFixture(t)
	s, cfg := reopen(t, dir)
	res, err := s.Create(cfg, core.CreateOpts{Type: datamodel.TypeTicket, Title: "Ticket", NoEdit: true})
	if err != nil {
		t.Fatal(err)
	}
	fs := storage.New(dir)
	path := fs.ItemPath(res.ID)
	promote := []string{"edit", res.Number, "--to-epic"}
	_ = verifyPromotionRejections(t, dir, path, [][]string{append(slices.Clone(promote), "KIRA-99")}, "exactly one")
	var conflictingCommands [][]string
	for _, flag := range []string{
		"--field=owner=a", "--label=", "--from-file=", "--force=false", "--title=", "--subtype=",
		"--priority=", "--rank=", "--owner=", "--sprint=", "--due=", "--resolution=",
	} {
		conflictingCommands = append(conflictingCommands, append(slices.Clone(promote), flag))
	}
	_ = verifyPromotionRejections(t, dir, path, conflictingCommands, "cannot be combined")
	_ = verifyPromotionRejections(t, dir, path, [][]string{{"edit", res.Number, "--field", "type=epic"}}, "immutable")
	_ = verifyPromotionRejections(t, dir, path, [][]string{{"restore", res.Number, "--at", "HEAD", "--field", "type"}}, "cannot be restored")

	it := readPromotionItem(t, path)
	it.State = "RETIRED"
	if _, err := fs.WriteItem(it); err != nil {
		t.Fatal(err)
	}
	err = verifyPromotionRejections(t, dir, path, [][]string{promote}, "no known workflow category")
	if userErr, ok := errors.AsType[*errx.Error](err); !ok || !strings.Contains(userErr.Hint, "declared state") {
		t.Fatalf("unknown state error = %#v, want hint to move to a declared state", err)
	}
}

func verifyPromotionRejections(t *testing.T, dir, path string, commands [][]string, message string) error {
	t.Helper()
	git := promotionGit(t, dir)
	snapshot := func() [4]string {
		t.Helper()
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		return [4]string{string(data), git("rev-parse", "HEAD"), git("ls-files", "--stage"), git("status", "--porcelain")}
	}
	before := snapshot()
	run := commandRunner(&globalFlags{chdir: dir})
	var lastErr error
	for _, args := range commands {
		if _, err := run(args); err == nil || !strings.Contains(err.Error(), message) {
			t.Fatalf("%v = %v, want %q", args, err, message)
		} else {
			lastErr = err
		}
	}
	if snapshot() != before {
		t.Fatalf("%v changed the item or Git state", commands)
	}
	return lastErr
}

func promotionGit(t *testing.T, dir string) func(...string) string {
	t.Helper()
	return func(args ...string) string {
		t.Helper()
		out, err := (gitx.Repo{Dir: dir}).Output(args...)
		if err != nil {
			t.Fatalf("git %v: %v", args, err)
		}
		return out
	}
}

func readPromotionItem(t *testing.T, path string) *datamodel.Item {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	item, err := codec.Parse(string(data))
	if err != nil {
		t.Fatal(err)
	}
	return item
}
