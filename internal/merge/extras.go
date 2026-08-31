package merge

import "github.com/shivamshivanshu/kira/internal/datamodel"

func extrasMerge(base, ours, theirs map[string]datamodel.ExtraValue, winner Side) map[string]datamodel.ExtraValue {
	out := map[string]datamodel.ExtraValue{}
	for k := range keyUnion(base, ours, theirs) {
		if merged := mergeExtra(base[k], ours[k], theirs[k], winner); !merged.Equal(datamodel.ExtraValue{}) {
			out[k] = merged
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func mergeExtra(base, ours, theirs datamodel.ExtraValue, winner Side) datamodel.ExtraValue {
	merged := datamodel.ExtraValue{
		Scalar: threeWayPtr(base.Scalar, ours.Scalar, theirs.Scalar, winner),
		List:   setMerge(base.List, ours.List, theirs.List),
	}
	if merged.Scalar != nil && len(merged.List) > 0 {
		// concurrent scalar/list retype: the winner side's arm decides the kind
		w := ours
		if winner == Theirs {
			w = theirs
		}
		if w.Scalar != nil {
			merged.List = nil
		} else {
			merged.Scalar = nil
		}
	}
	return merged
}
