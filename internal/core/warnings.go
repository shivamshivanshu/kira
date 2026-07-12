package core

import (
	"fmt"
	"os"
)

func emitWarnings(warns []error) {
	for _, w := range warns {
		fmt.Fprintln(os.Stderr, "kira: warning:", w.Error())
	}
}
