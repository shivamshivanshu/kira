package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"testing"

	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func applyEven(id string) (string, error) {
	n, err := strconv.Atoi(id)
	if err != nil {
		return "", err
	}
	if n%2 != 0 {
		return "", errors.New("odd id")
	}
	return "ok-" + id, nil
}

func lineEcho(v string) string { return v }

func TestRunBulkOutcomeContract(t *testing.T) {
	cases := []struct {
		name       string
		ids        []string
		wantErr    bool
		wantFailed int
	}{
		{"all succeed", []string{"2", "4", "6"}, false, 0},
		{"mixed", []string{"2", "3", "4"}, true, 1},
		{"all fail", []string{"1", "3"}, true, 2},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var out, errW bytes.Buffer
			err := runBulk(&out, &errW, false, c.ids, applyEven, lineEcho, nil)
			if (err != nil) != c.wantErr {
				t.Fatalf("runBulk() err = %v, wantErr %v", err, c.wantErr)
			}
			if c.wantFailed > 0 {
				want := fmt.Sprintf("%d of %d items failed", c.wantFailed, len(c.ids))
				if err == nil || err.Error() != want {
					t.Errorf("err = %v, want %q", err, want)
				}
			}
			gotFailedLines := bytes.Count(errW.Bytes(), []byte("odd id"))
			if gotFailedLines != c.wantFailed {
				t.Errorf("stderr has %d per-id failure lines, want %d:\n%s", gotFailedLines, c.wantFailed, errW.String())
			}
		})
	}
}

func TestRunBulkJSONEmitsOutcomesArray(t *testing.T) {
	var out, errW bytes.Buffer
	ids := []string{"2", "3"}
	err := runBulk(&out, &errW, true, ids, applyEven, lineEcho, nil)
	if err == nil {
		t.Fatal("expected an error for one failing id")
	}
	var outcomes []datamodel.BulkOutcome
	if jerr := json.Unmarshal(out.Bytes(), &outcomes); jerr != nil {
		t.Fatalf("stdout not a BulkOutcome array: %v\noutput: %q", jerr, out.String())
	}
	if len(outcomes) != len(ids) {
		t.Fatalf("outcomes has %d entries, want %d", len(outcomes), len(ids))
	}
	if outcomes[0].Ref != "2" || outcomes[0].Result != "ok-2" || outcomes[0].Error != "" {
		t.Errorf("outcomes[0] = %+v, want a success entry for id 2", outcomes[0])
	}
	if outcomes[1].Ref != "3" || outcomes[1].Result != nil || outcomes[1].Error == "" {
		t.Errorf("outcomes[1] = %+v, want a failure entry for id 3", outcomes[1])
	}
	if errW.Len() != 0 {
		t.Errorf("json mode must not write per-id lines to stderr, got %q", errW.String())
	}
}

func TestRunSingleOrBulkDispatchesByIDCount(t *testing.T) {
	var out, errW bytes.Buffer
	if err := runSingleOrBulk(&out, &errW, false, []string{"3"}, applyEven, lineEcho, nil); err == nil {
		t.Fatal("expected the single-id path to surface the apply error directly")
	}
	if bytes.Contains(out.Bytes(), []byte("[")) {
		t.Errorf("single-id path must not emit a bulk outcomes array, got %q", out.String())
	}

	out.Reset()
	errW.Reset()
	if err := runSingleOrBulk(&out, &errW, false, []string{"2", "4"}, applyEven, lineEcho, nil); err != nil {
		t.Fatalf("runSingleOrBulk() err = %v, want nil", err)
	}
}
