package index

import (
	"encoding/json"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
)

// itemExtras carries an item's schema-declared fields through the cache. They
// have no fixed column set — a repo defines them in .kira/schema — so they ride
// as one JSON blob rather than a table the schema would have to keep up with.
type itemExtras struct {
	Scalars map[string]datamodel.Rendered   `json:"scalars,omitempty"`
	Lists   map[string][]datamodel.Rendered `json:"lists,omitempty"`
}

func encodeExtras(it *datamodel.Item) (any, error) {
	if len(it.ExtraScalars) == 0 && len(it.ExtraLists) == 0 {
		return nil, nil
	}
	blob, err := json.Marshal(itemExtras{Scalars: it.ExtraScalars, Lists: it.ExtraLists})
	if err != nil {
		return nil, errx.Env("encoding index extras for %s: %v", it.ID, err)
	}
	return string(blob), nil
}

func decodeExtras(blob string, it *datamodel.Item) error {
	if blob == "" {
		return nil
	}
	var extras itemExtras
	if err := json.Unmarshal([]byte(blob), &extras); err != nil {
		return errx.Env("decoding index extras for %s: %v", it.ID, err)
	}
	it.ExtraScalars = extras.Scalars
	it.ExtraLists = extras.Lists
	return nil
}
