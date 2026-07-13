package merge

import (
	"time"

	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func laterUpdated(oursUpdated, theirsUpdated string, remote Side) Side {
	ot, oerr := time.Parse(time.RFC3339, oursUpdated)
	tt, terr := time.Parse(time.RFC3339, theirsUpdated)
	switch {
	case oerr != nil && terr != nil:
		return remote
	case oerr != nil:
		return Theirs
	case terr != nil:
		return Ours
	case ot.After(tt):
		return Ours
	case tt.After(ot):
		return Theirs
	default:
		return remote
	}
}

func threeWayScalar(base, ours, theirs string, winner Side) string {
	switch {
	case ours == theirs:
		return ours
	case ours == base:
		return theirs
	case theirs == base:
		return ours
	case winner == Ours:
		return ours
	default:
		return theirs
	}
}

func threeWayPtr[T comparable](base, ours, theirs *T, winner Side) *T {
	switch {
	case datamodel.EqualPtr(ours, theirs):
		return ours
	case datamodel.EqualPtr(ours, base):
		return theirs
	case datamodel.EqualPtr(theirs, base):
		return ours
	case winner == Ours:
		return ours
	default:
		return theirs
	}
}
