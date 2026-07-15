package ptr

func Deref[T any](p *T) T {
	if p == nil {
		var zero T
		return zero
	}
	return *p
}

func DerefOr[T comparable](p *T, fallback T) T {
	var zero T
	if p == nil || *p == zero {
		return fallback
	}
	return *p
}

func To[T any](v T) *T { return &v }

func NilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func Equal[T comparable](a, b *T) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}
