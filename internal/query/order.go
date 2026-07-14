package query

import (
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/ptr"
)

type OrderKey struct {
	null    bool
	numeric bool
	num     float64
	str     string
}

func (o *Order) Keyer(cfg *datamodel.Config) func(*datamodel.Item) OrderKey {
	switch o.Field {
	case fieldPriority:
		index := PriorityIndex(cfg.Priorities.Values)
		return func(it *datamodel.Item) OrderKey {
			idx, ok := index[ptr.Deref(it.Priority)]
			if !ok {
				return OrderKey{null: true}
			}
			return OrderKey{numeric: true, num: float64(idx)}
		}
	case fieldDue, fieldCreated, fieldUpdated:
		get := scalarGet(o.Field)
		return func(it *datamodel.Item) OrderKey {
			t, err := parseDate(get(it, cfg))
			if err != nil {
				return OrderKey{null: true}
			}
			return OrderKey{numeric: true, num: float64(t.UnixNano())}
		}
	case fieldEstimate:
		return func(it *datamodel.Item) OrderKey {
			if it.Estimate == nil {
				return OrderKey{null: true}
			}
			return OrderKey{numeric: true, num: *it.Estimate}
		}
	default:
		get := scalarGet(o.Field)
		return func(it *datamodel.Item) OrderKey {
			s := get(it, cfg)
			if s == "" {
				return OrderKey{null: true}
			}
			return OrderKey{str: s}
		}
	}
}

func (o *Order) Less(a, b OrderKey) bool {
	if a.null || b.null {
		return !a.null && b.null
	}
	if a.numeric {
		if a.num == b.num {
			return false
		}
		return (a.num < b.num) != o.Desc
	}
	if a.str == b.str {
		return false
	}
	return (a.str < b.str) != o.Desc
}
