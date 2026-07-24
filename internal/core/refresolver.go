package core

import (
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/id"
)

// refResolver adapts the item store to entityschema.RefResolver. It conflates
// dangling and wrong-type refs into a single verdict, so doctor keeps its own
// dangling-vs-wrong-type logic rather than sharing this.
type refResolver struct {
	resolver *id.Resolver
	byID     map[string]*datamodel.Item
}

func newRefResolver(resolver *id.Resolver, items []*datamodel.Item) refResolver {
	return refResolver{resolver: resolver, byID: byULID(items)}
}

func (r refResolver) Exists(target, ref string) bool {
	ulid, err := r.resolver.Resolve(ref)
	if err != nil {
		return false
	}
	it, ok := r.byID[ulid]
	return ok && it.Type == target
}
