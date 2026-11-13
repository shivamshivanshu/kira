package warningfmt

import (
	"fmt"
	"strings"

	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func Message(w datamodel.Warning) string {
	var args [2]string
	copy(args[:], w.Args)
	switch w.Code {
	case datamodel.WarnIndexFallback:
		return fmt.Sprintf("index unavailable (%s), using linear scan", args[0])
	case datamodel.WarnNoActiveSprint:
		return "no active sprint set; sprint=active matches nothing (run `kira sprint activate <key>`)"
	case datamodel.WarnCloseUnknown:
		return fmt.Sprintf("unknown ticket %s in %s (hint: run `kira config explain` to see the configured close trailer)", args[0], args[1])
	case datamodel.WarnCloseFailed:
		return fmt.Sprintf("failed to close %s: %s", args[0], args[1])
	case datamodel.WarnLiteral:
		return args[0]
	}
	if len(w.Args) == 0 {
		return string(w.Code)
	}
	return string(w.Code) + ": " + strings.Join(w.Args, ": ")
}
