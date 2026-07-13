package cli

import (
	"fmt"
	"io"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
)

func runBulk[T any](out, errW io.Writer, jsonMode bool, ids []string, apply func(string) (T, error), line func(T) string) error {
	outcomes := make([]datamodel.BulkOutcome, 0, len(ids))
	failed := 0
	for _, id := range ids {
		res, err := apply(id)
		if err != nil {
			failed++
			outcomes = append(outcomes, datamodel.BulkOutcome{Ref: id, Error: err.Error()})
			if !jsonMode {
				fmt.Fprintln(errW, msgPrefix, id+":", err)
			}
			continue
		}
		outcomes = append(outcomes, datamodel.BulkOutcome{Ref: id, Result: res})
		if !jsonMode {
			fmt.Fprintln(out, line(res))
		}
	}
	if jsonMode {
		if err := emitJSON(out, outcomes); err != nil {
			return err
		}
	}
	if failed > 0 {
		return errx.User("%d of %d items failed", failed, len(ids))
	}
	return nil
}
