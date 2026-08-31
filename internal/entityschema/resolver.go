package entityschema

import "github.com/shivamshivanshu/kira/internal/datamodel"

// NewRefResolver resolves refs against a loaded item set: an id exists when an
// item carries it and matches when its type equals target ("" matches any).
func NewRefResolver(items []*datamodel.Item) RefResolver {
	return itemRefResolver{byID: datamodel.IndexByID(items)}
}

type itemRefResolver struct{ byID map[string]*datamodel.Item }

func (r itemRefResolver) Exists(target, id string) bool {
	it, ok := r.byID[id]
	return ok && (target == "" || it.Type == target)
}
