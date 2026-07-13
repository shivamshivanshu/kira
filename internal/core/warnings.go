package core

import (
	"fmt"
	"os"

	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func emitWarnings(warns []error) {
	for _, w := range warns {
		fmt.Fprintln(os.Stderr, "kira: warning:", w.Error())
	}
}

func literalWarnings(msgs []string) []datamodel.Warning {
	if len(msgs) == 0 {
		return nil
	}
	out := make([]datamodel.Warning, len(msgs))
	for i, m := range msgs {
		out[i] = datamodel.Warning{Code: datamodel.WarnLiteral, Args: []string{m}}
	}
	return out
}
