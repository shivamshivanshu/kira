package merge

import (
	"github.com/shivamshivanshu/kira/internal/ptr"
	"github.com/shivamshivanshu/kira/internal/timex"
)

func laterUpdated(oursUpdated, theirsUpdated string, remote Side) Side {
	cmp, oursOK, theirsOK := timex.CompareRFC3339(oursUpdated, theirsUpdated)
	switch {
	case !oursOK && !theirsOK:
		return remote
	case !oursOK:
		return Theirs
	case !theirsOK:
		return Ours
	case cmp > 0:
		return Ours
	case cmp < 0:
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
	case ptr.Equal(ours, theirs):
		return ours
	case ptr.Equal(ours, base):
		return theirs
	case ptr.Equal(theirs, base):
		return ours
	case winner == Ours:
		return ours
	default:
		return theirs
	}
}
